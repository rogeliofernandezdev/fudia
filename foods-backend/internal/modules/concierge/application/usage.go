package application

import (
	"context"
	"errors"
	"regexp"
	"strings"

	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/concierge/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

// UsageRow is the raw monthly state of one company; the application derives
// the effective cap, percentage and alert from it.
type UsageRow struct {
	OrganizationID, OrganizationName, Month string
	PlanName                                *string
	PlanMax                                 *int
	HasOverride                             bool
	OverrideMax                             *int
	Conversations, Rejected                 int
}

func (r UsageRow) Limit() domain.Limit {
	return domain.EffectiveLimit(r.PlanMax, r.HasOverride, r.OverrideMax)
}

type QRResolver interface {
	ResolveQR(context.Context, string) (QRContext, error)
}
type UsageRepository interface {
	Settings(context.Context) (domain.UsageSettings, error)
	SaveSettings(ctx context.Context, userID string, in domain.UsageSettingsInput) error
	// LockMonth creates the company's counter for its current month (company
	// timezone) and locks it until the transaction ends.
	LockMonth(ctx context.Context, organizationID string) (month string, conversations int, err error)
	Conversation(ctx context.Context, organizationID, conversationID string) (ref string, view domain.ConversationView, err error)
	Limit(ctx context.Context, organizationID string) (domain.Limit, error)
	InsertConversation(ctx context.Context, qr QRContext, in domain.ConversationInput, month string) (string, error)
	RecordRejected(ctx context.Context, organizationID string) error
	RecordTokens(ctx context.Context, organizationID, conversationRef, month, requestID string, tiers []domain.TierTokens) error
	UsageRows(context.Context) ([]UsageRow, error)
	OrganizationUsage(ctx context.Context, organizationID string) (UsageRow, error)
	Tokens(ctx context.Context, organizationID, month string) ([]domain.TierTokens, error)
	History(ctx context.Context, organizationID string, months int) ([]domain.UsageMonth, error)
	SaveOverride(ctx context.Context, userID, organizationID string, unlimited bool, max *int) error
	DeleteOverride(ctx context.Context, organizationID string) error
}

// Usage measures Concierge conversations and applies the monthly cap.
type Usage struct {
	repo     UsageRepository
	resolver QRResolver
	work     unitofwork.Manager
	audit    audit.Repository
}

func NewUsage(repo UsageRepository, resolver QRResolver, work unitofwork.Manager, audit audit.Repository) *Usage {
	return &Usage{repo, resolver, work, audit}
}

const historyMonths = 6
const maxListedOrganizations = 100

// A single turn never consumes this much; larger values are malformed reports.
const maxTurnTokens = 50_000_000

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func usageUnavailable(err error) error {
	return apperror.Wrap(apperror.Unavailable, "concierge_usage_unavailable", "No pudimos cargar el uso de Fudia Concierge.", err)
}

func (u *Usage) resolve(ctx context.Context, token string) (QRContext, error) {
	qr, err := u.resolver.ResolveQR(ctx, token)
	if errors.Is(err, ErrNotFound) {
		return qr, conciergeFailure(apperror.NotFound, "table_not_found", "La mesa no existe o el QR no está activo.")
	}
	if err != nil {
		return qr, conciergeFailure(apperror.Unavailable, "concierge_unavailable", "No pudimos resolver el QR.")
	}
	return qr, nil
}

// OpenConversation registers a new attention once (idempotent by
// conversationId). In enforce mode a company at its cap gets an explicit
// error; a conversation already registered is always accepted.
func (u *Usage) OpenConversation(ctx context.Context, token string, in domain.ConversationInput) (domain.ConversationView, bool, error) {
	in.ConversationID = strings.TrimSpace(in.ConversationID)
	in.Kind = strings.TrimSpace(in.Kind)
	if in.ConversationID == "" || len(in.ConversationID) > 160 || (in.Kind != domain.ConversationKindTable && in.Kind != domain.ConversationKindDelivery) {
		return domain.ConversationView{}, false, conciergeFailure(apperror.Invalid, "invalid_conversation", "Revisa la conversación enviada.")
	}
	qr, err := u.resolve(ctx, token)
	if err != nil {
		return domain.ConversationView{}, false, err
	}
	org := qr.Scope.OrganizationID
	failed := func(err error) error {
		return apperror.Wrap(apperror.Unavailable, "conversation_unavailable", "No pudimos registrar la conversación.", err)
	}
	tx, err := u.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return domain.ConversationView{}, false, failed(err)
	}
	defer tx.Rollback(ctx)
	month, count, err := u.repo.LockMonth(tx.Context(), org)
	if err != nil {
		return domain.ConversationView{}, false, failed(err)
	}
	if _, existing, err := u.repo.Conversation(tx.Context(), org, in.ConversationID); err == nil {
		return existing, false, nil
	} else if !errors.Is(err, ErrNotFound) {
		return domain.ConversationView{}, false, failed(err)
	}
	settings, err := u.repo.Settings(tx.Context())
	if err != nil {
		return domain.ConversationView{}, false, failed(err)
	}
	limit, err := u.repo.Limit(tx.Context(), org)
	if err != nil {
		return domain.ConversationView{}, false, failed(err)
	}
	if domain.Blocks(settings.Mode, limit, count) {
		_ = tx.Rollback(ctx)
		_ = u.repo.RecordRejected(ctx, org)
		return domain.ConversationView{}, false, conciergeFailure(apperror.Conflict, "concierge_conversation_limit_reached", "Fudia Concierge no puede iniciar conversaciones nuevas para este restaurante este mes.")
	}
	if _, err = u.repo.InsertConversation(tx.Context(), qr, in, month); err != nil {
		return domain.ConversationView{}, false, failed(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.ConversationView{}, false, failed(err)
	}
	return domain.ConversationView{ConversationID: in.ConversationID, Kind: in.Kind, Month: month}, true, nil
}

// ReportUsage stores the tokens of one turn. It never rejects: a conversation
// that was not registered (for example, opened before this version) is
// registered and counted without applying the cap.
func (u *Usage) ReportUsage(ctx context.Context, token, conversationID string, in domain.UsageReport) error {
	conversationID = strings.TrimSpace(conversationID)
	in.RequestID = strings.TrimSpace(in.RequestID)
	invalid := conciergeFailure(apperror.Invalid, "invalid_usage_report", "Revisa el consumo enviado.")
	if conversationID == "" || len(conversationID) > 160 || in.RequestID == "" || len(in.RequestID) > 160 || len(in.Tiers) == 0 || len(in.Tiers) > 2 {
		return invalid
	}
	seen := map[string]bool{}
	for _, t := range in.Tiers {
		if (t.Tier != domain.TierLite && t.Tier != domain.TierFlash) || seen[t.Tier] ||
			t.InputTokens < 0 || t.OutputTokens < 0 || t.InputTokens > maxTurnTokens || t.OutputTokens > maxTurnTokens {
			return invalid
		}
		seen[t.Tier] = true
	}
	qr, err := u.resolve(ctx, token)
	if err != nil {
		return err
	}
	org := qr.Scope.OrganizationID
	failed := func(err error) error {
		return apperror.Wrap(apperror.Unavailable, "usage_unavailable", "No pudimos registrar el consumo.", err)
	}
	tx, err := u.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return failed(err)
	}
	defer tx.Rollback(ctx)
	ref, view, err := u.repo.Conversation(tx.Context(), org, conversationID)
	if errors.Is(err, ErrNotFound) {
		var month string
		if month, _, err = u.repo.LockMonth(tx.Context(), org); err == nil {
			view.Month = month
			ref, err = u.repo.InsertConversation(tx.Context(), qr, domain.ConversationInput{ConversationID: conversationID, Kind: domain.ConversationKindTable}, month)
		}
	}
	if err != nil {
		return failed(err)
	}
	if err = u.repo.RecordTokens(tx.Context(), org, ref, view.Month, in.RequestID, in.Tiers); err != nil {
		return failed(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return failed(err)
	}
	return nil
}

func (u *Usage) Settings(ctx context.Context) (domain.UsageSettings, error) {
	out, err := u.repo.Settings(ctx)
	if err != nil {
		return out, usageUnavailable(err)
	}
	return out, nil
}

func (u *Usage) SaveSettings(ctx context.Context, s scope, in domain.UsageSettingsInput) (domain.UsageSettings, error) {
	in.Mode = strings.TrimSpace(in.Mode)
	if !domain.ValidUsageMode(in.Mode) || in.WarnPercent < 1 || in.WarnPercent > 100 {
		return domain.UsageSettings{}, conciergeFailure(apperror.Invalid, "invalid_concierge_usage_settings", "Elige un modo válido y un porcentaje de aviso entre 1 y 100.")
	}
	if err := u.repo.SaveSettings(ctx, s.UserID, in); err != nil {
		return domain.UsageSettings{}, apperror.Wrap(apperror.Unavailable, "concierge_usage_settings_unavailable", "No pudimos guardar la configuración de Fudia Concierge.", err)
	}
	return u.Settings(ctx)
}

// List returns the current month of every company (or of the requested ones);
// the alert summary always covers every company.
func (u *Usage) List(ctx context.Context, organizationIDs []string) (domain.UsagePage, error) {
	out := domain.UsagePage{Items: []domain.UsageSummary{}}
	wanted := map[string]bool{}
	for _, raw := range organizationIDs {
		for _, id := range strings.Split(raw, ",") {
			if id = strings.ToLower(strings.TrimSpace(id)); id == "" {
				continue
			}
			if !uuidPattern.MatchString(id) {
				return out, conciergeFailure(apperror.Invalid, "invalid_organization", "La empresa no es válida.")
			}
			wanted[id] = true
		}
	}
	if len(wanted) > maxListedOrganizations {
		return out, conciergeFailure(apperror.Invalid, "too_many_organizations", "Consulta como máximo 100 empresas a la vez.")
	}
	settings, err := u.repo.Settings(ctx)
	if err != nil {
		return out, usageUnavailable(err)
	}
	rows, err := u.repo.UsageRows(ctx)
	if err != nil {
		return out, usageUnavailable(err)
	}
	out.Settings = settings
	for _, row := range rows {
		item := summary(row, settings)
		switch item.Alert {
		case domain.UsageAlertWarning:
			out.Summary.Warning++
		case domain.UsageAlertReached:
			out.Summary.Reached++
		}
		if len(wanted) == 0 || wanted[strings.ToLower(row.OrganizationID)] {
			out.Items = append(out.Items, item)
		}
	}
	return out, nil
}

func summary(row UsageRow, settings domain.UsageSettings) domain.UsageSummary {
	item := domain.UsageSummary{
		OrganizationID:        row.OrganizationID,
		OrganizationName:      row.OrganizationName,
		Month:                 row.Month,
		PlanName:              row.PlanName,
		Conversations:         row.Conversations,
		RejectedConversations: row.Rejected,
	}
	item.Apply(settings, row.Limit())
	return item
}

func (u *Usage) Organization(ctx context.Context, organizationID string) (domain.UsageDetail, error) {
	var out domain.UsageDetail
	organizationID = strings.TrimSpace(organizationID)
	if !uuidPattern.MatchString(organizationID) {
		return out, conciergeFailure(apperror.NotFound, "organization_not_found", "La empresa no existe.")
	}
	row, err := u.repo.OrganizationUsage(ctx, organizationID)
	if errors.Is(err, ErrNotFound) {
		return out, conciergeFailure(apperror.NotFound, "organization_not_found", "La empresa no existe.")
	}
	if err != nil {
		return out, usageUnavailable(err)
	}
	settings, err := u.repo.Settings(ctx)
	if err != nil {
		return out, usageUnavailable(err)
	}
	out.UsageSummary = summary(row, settings)
	out.PlanMaxConversations = row.PlanMax
	out.Settings = settings
	if out.Tokens, err = u.repo.Tokens(ctx, organizationID, row.Month); err != nil {
		return out, usageUnavailable(err)
	}
	if out.History, err = u.repo.History(ctx, organizationID, historyMonths); err != nil {
		return out, usageUnavailable(err)
	}
	return out, nil
}

// SaveOverride sets the company exception over its plan cap. It applies from
// the next conversation; the current month's count is kept.
func (u *Usage) SaveOverride(ctx context.Context, s scope, organizationID string, in domain.LimitOverrideInput) (domain.UsageDetail, error) {
	organizationID = strings.TrimSpace(organizationID)
	in.LimitSource = strings.TrimSpace(in.LimitSource)
	invalid := conciergeFailure(apperror.Invalid, "invalid_concierge_limit", "Elige el tope del plan, un tope propio mayor que cero o ilimitado.")
	switch in.LimitSource {
	case domain.LimitSourcePlan, domain.LimitSourceUnlimited:
	case domain.LimitSourceCustom:
		if in.MaxConversations == nil || *in.MaxConversations < 1 {
			return domain.UsageDetail{}, invalid
		}
	default:
		return domain.UsageDetail{}, invalid
	}
	if _, err := u.Organization(ctx, organizationID); err != nil {
		return domain.UsageDetail{}, err
	}
	var err error
	switch in.LimitSource {
	case domain.LimitSourcePlan:
		err = u.repo.DeleteOverride(ctx, organizationID)
	case domain.LimitSourceUnlimited:
		err = u.repo.SaveOverride(ctx, s.UserID, organizationID, true, nil)
	default:
		err = u.repo.SaveOverride(ctx, s.UserID, organizationID, false, in.MaxConversations)
	}
	if err != nil {
		return domain.UsageDetail{}, apperror.Wrap(apperror.Unavailable, "concierge_limit_unavailable", "No pudimos guardar el tope de Fudia Concierge.", err)
	}
	_ = u.audit.Record(ctx, s, "concierge.limit.updated", "organization", organizationID)
	return u.Organization(ctx, organizationID)
}
