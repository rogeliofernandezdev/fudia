package application

import (
	"context"
	"errors"
	"regexp"

	"github.com/foods-platform/foods-backend/internal/modules/notifications/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

type scope = requestscope.Scope

var ErrNotFound = errors.New("notice not found")

// FeedLimit acota la campana a los avisos más recientes de las últimas 24 horas.
const FeedLimit = 30

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Repository interface {
	Insert(context.Context, scope, domain.Input) error
	Feed(ctx context.Context, s scope, audiences []string, limit int) (domain.Feed, error)
	MarkRead(ctx context.Context, s scope, id string, audiences []string) error
	MarkAllRead(ctx context.Context, s scope, audiences []string) error
	InsertCustomer(context.Context, scope, domain.CustomerInput) error
	ClaimCustomer(ctx context.Context, limit int) ([]domain.CustomerNotice, error)
	MarkCustomerDispatched(ctx context.Context, id string) error
}

// PermissionAuthorizer reutiliza la autorización de Identity para decidir qué
// audiencias ve el usuario: salón (tables.read) y caja (cash.manage).
type PermissionAuthorizer interface {
	Authorize(context.Context, scope, string) error
}

type Notices struct {
	repo       Repository
	authorizer PermissionAuthorizer
}

func New(repo Repository, authorizer PermissionAuthorizer) *Notices {
	return &Notices{repo: repo, authorizer: authorizer}
}

func failure(kind apperror.Kind, code, message string) error {
	return apperror.New(kind, code, message)
}

// Publish implementa contracts.Publisher. Un aviso inválido se descarta.
func (a *Notices) Publish(ctx context.Context, s scope, in domain.Input) error {
	in, ok := domain.Normalize(in)
	if !ok || s.OrganizationID == "" || s.LocationID == "" {
		return nil
	}
	return a.repo.Insert(ctx, s, in)
}

func (a *Notices) audiences(ctx context.Context, s scope) []string {
	allowed := func(permission string) bool {
		return a.authorizer != nil && a.authorizer.Authorize(ctx, s, permission) == nil
	}
	return domain.Audiences(allowed("tables.read"), allowed("cash.manage"))
}

func (a *Notices) Feed(ctx context.Context, s scope) (domain.Feed, error) {
	audiences := a.audiences(ctx, s)
	if len(audiences) == 0 {
		return domain.Feed{Items: []domain.Notice{}}, nil
	}
	out, err := a.repo.Feed(ctx, s, audiences, FeedLimit)
	if err != nil {
		return domain.Feed{}, failure(apperror.Unavailable, "notices_unavailable", "No pudimos cargar los avisos.")
	}
	return out, nil
}

func (a *Notices) MarkRead(ctx context.Context, s scope, id string) error {
	audiences := a.audiences(ctx, s)
	if !uuidPattern.MatchString(id) || len(audiences) == 0 {
		return failure(apperror.NotFound, "notice_not_found", "El aviso no existe.")
	}
	err := a.repo.MarkRead(ctx, s, id, audiences)
	if errors.Is(err, ErrNotFound) {
		return failure(apperror.NotFound, "notice_not_found", "El aviso no existe.")
	}
	if err != nil {
		return failure(apperror.Unavailable, "notices_unavailable", "No pudimos marcar el aviso como leído.")
	}
	return nil
}

func (a *Notices) MarkAllRead(ctx context.Context, s scope) error {
	audiences := a.audiences(ctx, s)
	if len(audiences) == 0 {
		return nil
	}
	if err := a.repo.MarkAllRead(ctx, s, audiences); err != nil {
		return failure(apperror.Unavailable, "notices_unavailable", "No pudimos marcar los avisos como leídos.")
	}
	return nil
}
