// Package application coordina Delivery Fudia: centrales de la plataforma (en
// un orden global), activación por empresa, enlace de cada local y despacho de
// pedidos a las centrales.
package application

import (
	"context"
	"errors"
	"regexp"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

type scope = requestscope.Scope

var (
	ErrNotFound             = errors.New("delivery record not found")
	ErrDuplicatePhone       = errors.New("delivery partner phone already registered")
	ErrOrderMismatch        = errors.New("delivery partner order does not match the registered partners")
	ErrInvalidCoverage      = errors.New("invalid delivery coverage")
	uuidPattern             = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	errPartnerNotFound      = failure(apperror.NotFound, "delivery_partner_not_found", "La central no existe.")
	errLocationNotFound     = failure(apperror.NotFound, "location_not_found", "El local no existe.")
	errOrganizationNotFound = failure(apperror.NotFound, "organization_not_found", "La empresa no existe.")
)

func failure(kind apperror.Kind, code, message string) error {
	return apperror.New(kind, code, message)
}

func unavailable(code, message string, cause error) error {
	return apperror.Wrap(apperror.Unavailable, code, message, cause)
}

// PartnerRepository guarda las centrales, su orden global, la activación por
// empresa y el enlace de cada local.
type PartnerRepository interface {
	ListPartners(ctx context.Context) ([]domain.Partner, error)
	// CreatePartner agrega la central al final del orden global.
	CreatePartner(ctx context.Context, in domain.PartnerInput) (domain.Partner, error)
	UpdatePartner(ctx context.Context, id string, in domain.PartnerInput) (domain.Partner, error)
	SetCoverage(ctx context.Context, id string, codes []string) error
	// ReorderPartners devuelve ErrOrderMismatch si ids no son todas las centrales.
	ReorderPartners(ctx context.Context, ids []string) error
	// RotatePanelToken reemplaza el enlace del panel web de la central.
	RotatePanelToken(ctx context.Context, id string) (domain.PanelLink, error)
	OrganizationDelivery(ctx context.Context, organizationID string) (domain.OrganizationDelivery, error)
	SetOrganizationDelivery(ctx context.Context, organizationID string, enabled bool) (domain.OrganizationDelivery, error)
	DeliveryLink(ctx context.Context, s scope, locationID string) (domain.DeliveryLink, error)
	// RotateDeliveryLink crea el enlace o le asigna un token nuevo (el anterior deja de abrir).
	RotateDeliveryLink(ctx context.Context, s scope, locationID string) (domain.DeliveryLink, error)
}

// Partners administra las centrales y la activación por empresa (solo
// plataforma) y el enlace de delivery de cada local.
type Partners struct {
	repo PartnerRepository
	work unitofwork.Manager
}

func NewPartners(repo PartnerRepository, work unitofwork.Manager) *Partners {
	return &Partners{repo: repo, work: work}
}

func (a *Partners) List(ctx context.Context) ([]domain.Partner, error) {
	out, err := a.repo.ListPartners(ctx)
	if err != nil {
		return nil, unavailable("delivery_partners_unavailable", "No pudimos cargar las centrales de delivery.", err)
	}
	return out, nil
}

func (a *Partners) Create(ctx context.Context, in domain.PartnerInput) (domain.Partner, error) {
	in, problem := domain.NormalizePartner(in)
	if problem != "" {
		return domain.Partner{}, failure(apperror.Invalid, "invalid_delivery_partner", problem)
	}
	return a.save(ctx, "", in)
}

func (a *Partners) Update(ctx context.Context, id string, in domain.PartnerInput) (domain.Partner, error) {
	if !uuidPattern.MatchString(id) {
		return domain.Partner{}, errPartnerNotFound
	}
	in, problem := domain.NormalizePartner(in)
	if problem != "" {
		return domain.Partner{}, failure(apperror.Invalid, "invalid_delivery_partner", problem)
	}
	return a.save(ctx, id, in)
}

func (a *Partners) save(ctx context.Context, id string, in domain.PartnerInput) (domain.Partner, error) {
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return domain.Partner{}, partnerError(err)
	}
	defer tx.Rollback(ctx)
	var out domain.Partner
	if id == "" {
		out, err = a.repo.CreatePartner(tx.Context(), in)
	} else {
		out, err = a.repo.UpdatePartner(tx.Context(), id, in)
	}
	if err == nil {
		err = a.repo.SetCoverage(tx.Context(), out.ID, in.CoverageCodes)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return domain.Partner{}, partnerError(err)
	}
	// One query returns the canonical names of all coverage areas.
	items, err := a.repo.ListPartners(ctx)
	if err != nil {
		return domain.Partner{}, partnerError(err)
	}
	for _, item := range items {
		if item.ID == out.ID {
			return item, nil
		}
	}
	return out, nil
}

func partnerError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrNotFound):
		return errPartnerNotFound
	case errors.Is(err, ErrDuplicatePhone):
		return failure(apperror.Conflict, "delivery_partner_phone_taken", "Ya existe una central con ese WhatsApp.")
	case errors.Is(err, ErrInvalidCoverage):
		return failure(apperror.Invalid, "invalid_delivery_coverage", "Selecciona zonas de cobertura del catálogo geográfico.")
	}
	return unavailable("delivery_partners_unavailable", "No pudimos guardar la central.", err)
}

// Reorder fija el orden global de las centrales (la primera recibe primero
// cada pedido). Debe incluir todas las centrales registradas, sin repetir.
func (a *Partners) Reorder(ctx context.Context, in domain.PartnerOrderInput) ([]domain.Partner, error) {
	ids, problem := domain.NormalizeOrder(in)
	if problem != "" {
		return nil, failure(apperror.Invalid, "invalid_delivery_partner_order", problem)
	}
	for _, id := range ids {
		if !uuidPattern.MatchString(id) {
			return nil, failure(apperror.Invalid, "invalid_delivery_partner_order", "Una de las centrales ya no existe.")
		}
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return nil, unavailable("delivery_partners_unavailable", "No pudimos guardar el orden de las centrales.", err)
	}
	defer tx.Rollback(ctx)
	err = a.repo.ReorderPartners(tx.Context(), ids)
	if errors.Is(err, ErrOrderMismatch) {
		return nil, failure(apperror.Conflict, "delivery_partner_order_outdated", "La lista de centrales cambió. Recarga e inténtalo de nuevo.")
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return nil, unavailable("delivery_partners_unavailable", "No pudimos guardar el orden de las centrales.", err)
	}
	return a.List(ctx)
}

// OrganizationDelivery indica si la empresa usa Delivery Fudia (solo plataforma).
func (a *Partners) OrganizationDelivery(ctx context.Context, organizationID string) (domain.OrganizationDelivery, error) {
	if !uuidPattern.MatchString(organizationID) {
		return domain.OrganizationDelivery{}, errOrganizationNotFound
	}
	out, err := a.repo.OrganizationDelivery(ctx, organizationID)
	if errors.Is(err, ErrNotFound) {
		return out, errOrganizationNotFound
	}
	if err != nil {
		return out, unavailable("delivery_settings_unavailable", "No pudimos cargar el delivery de la empresa.", err)
	}
	return out, nil
}

// SetOrganizationDelivery activa o desactiva Delivery Fudia para todos los
// locales de la empresa. Los pedidos ya ofrecidos siguen su curso.
func (a *Partners) SetOrganizationDelivery(ctx context.Context, organizationID string, in domain.OrganizationDeliveryInput) (domain.OrganizationDelivery, error) {
	if !uuidPattern.MatchString(organizationID) {
		return domain.OrganizationDelivery{}, errOrganizationNotFound
	}
	out, err := a.repo.SetOrganizationDelivery(ctx, organizationID, in.Enabled)
	if errors.Is(err, ErrNotFound) {
		return out, errOrganizationNotFound
	}
	if err != nil {
		return out, unavailable("delivery_settings_unavailable", "No pudimos guardar el delivery de la empresa.", err)
	}
	return out, nil
}

// Link devuelve el enlace de delivery del local (token vacío si no existe).
func (a *Partners) Link(ctx context.Context, s scope, locationID string) (domain.DeliveryLink, error) {
	if !uuidPattern.MatchString(locationID) {
		return domain.DeliveryLink{}, errLocationNotFound
	}
	out, err := a.repo.DeliveryLink(ctx, s, locationID)
	if errors.Is(err, ErrNotFound) {
		return out, errLocationNotFound
	}
	if err != nil {
		return out, unavailable("delivery_link_unavailable", "No pudimos cargar el enlace de delivery.", err)
	}
	return out, nil
}

// RotateLink genera el enlace de delivery del local o lo reemplaza por uno
// nuevo; los QR impresos con el token anterior dejan de funcionar.
func (a *Partners) RotateLink(ctx context.Context, s scope, locationID string) (domain.DeliveryLink, error) {
	if !uuidPattern.MatchString(locationID) {
		return domain.DeliveryLink{}, errLocationNotFound
	}
	out, err := a.repo.RotateDeliveryLink(ctx, s, locationID)
	if errors.Is(err, ErrNotFound) {
		return out, errLocationNotFound
	}
	if err != nil {
		return out, unavailable("delivery_link_unavailable", "No pudimos generar el enlace de delivery.", err)
	}
	return out, nil
}

// RotatePanel reemplaza el enlace privado del panel web de la central; quien
// tenga el enlace anterior deja de ver sus pedidos.
func (a *Partners) RotatePanel(ctx context.Context, id string) (domain.PanelLink, error) {
	if !uuidPattern.MatchString(id) {
		return domain.PanelLink{}, errPartnerNotFound
	}
	out, err := a.repo.RotatePanelToken(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return out, errPartnerNotFound
	}
	if err != nil {
		return out, unavailable("delivery_partners_unavailable", "No pudimos generar el enlace del panel.", err)
	}
	return out, nil
}
