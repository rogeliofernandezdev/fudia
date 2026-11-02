package application

import (
	"context"
	"errors"
	customercontracts "github.com/foods-platform/foods-backend/internal/modules/customers/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/customers/domain"
	deliverycontracts "github.com/foods-platform/foods-backend/internal/modules/delivery/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"strings"
	"unicode/utf8"
)

var (
	ErrNotFound = errors.New("customer not found")
	ErrConflict = errors.New("customer conflict")
	ErrAddress  = errors.New("invalid customer address")
)

type Filter struct {
	Page, PageSize         int
	Query, Status, Segment string
}
type Page struct {
	Items []domain.Customer
	Total int
}

// Repository persists a customer aggregate, including its addresses atomically.
type Repository interface {
	List(context.Context, string, Filter) (Page, error)
	Get(context.Context, string, string) (domain.Customer, error)
	Save(context.Context, requestscope.Scope, string, domain.Input) (domain.Customer, error)
	WhatsAppProfile(ctx context.Context, organizationID, digits string) (customercontracts.WhatsAppProfile, error)
	RecordWhatsAppConsent(ctx context.Context, s requestscope.Scope, digits string, in customercontracts.ConsentEvidence) (customercontracts.WhatsAppProfile, error)
	SetActive(context.Context, string, string, bool) error
	Audit(context.Context, requestscope.Scope, string, string)
}
type Service struct{ repository Repository }

func New(repository Repository) *Service { return &Service{repository} }
func (s *Service) List(ctx context.Context, organizationID string, f Filter) (Page, error) {
	f.Query = strings.TrimSpace(f.Query)
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
	out, err := s.repository.List(ctx, organizationID, f)
	if err != nil {
		return out, apperror.New(apperror.Unavailable, "customers_unavailable", "No pudimos cargar los clientes.")
	}
	return out, nil
}
func (s *Service) Get(ctx context.Context, organizationID, id string) (domain.Customer, error) {
	c, err := s.repository.Get(ctx, organizationID, id)
	if errors.Is(err, ErrNotFound) {
		return c, notFound()
	}
	if err != nil {
		return c, unavailable("No pudimos cargar el cliente.")
	}
	return c, nil
}
func (s *Service) Save(ctx context.Context, scope requestscope.Scope, id string, in domain.Input) (domain.Customer, error) {
	if !domain.Normalize(&in) {
		return domain.Customer{}, apperror.New(apperror.Invalid, "invalid_customer", "El nombre y los datos del cliente no son válidos.")
	}
	for _, address := range in.Addresses {
		if !validPoint(address.Latitude, address.Longitude) {
			return domain.Customer{}, apperror.New(apperror.Invalid, "invalid_address", "Revisa la ubicación marcada en las direcciones del cliente.")
		}
	}
	c, err := s.repository.Save(ctx, scope, id, in)
	switch {
	case errors.Is(err, ErrNotFound):
		return c, notFound()
	case errors.Is(err, ErrConflict):
		return c, apperror.New(apperror.Conflict, "customer_conflict", "Ya existe un cliente con ese documento o los datos no son válidos.")
	case errors.Is(err, ErrAddress):
		return c, apperror.New(apperror.Invalid, "invalid_address", "Revisa las direcciones del cliente.")
	case err != nil:
		return c, unavailable("No pudimos guardar el cliente.")
	}
	action := "created"
	if id != "" {
		action = "updated"
	}
	s.repository.Audit(ctx, scope, action, c.ID)
	return c, nil
}
func (s *Service) SetActive(ctx context.Context, scope requestscope.Scope, id string, active bool) error {
	err := s.repository.SetActive(ctx, scope.OrganizationID, id, active)
	if err != nil {
		return notFound()
	}
	s.repository.Audit(ctx, scope, "status_updated", id)
	return nil
}
func notFound() error {
	return apperror.New(apperror.NotFound, "customer_not_found", "El cliente no existe.")
}
func unavailable(message string) error {
	return apperror.New(apperror.Unavailable, "customer_unavailable", message)
}

// validPoint acepta una dirección sin ubicar o con ambas coordenadas dentro de rango.
func validPoint(latitude, longitude *float64) bool {
	if latitude == nil || longitude == nil {
		return latitude == nil && longitude == nil
	}
	return deliverycontracts.ValidCoordinates(*latitude, *longitude)
}

// WhatsAppProfile reconoce al comensal por su número dentro de la empresa del
// QR: si ya es cliente, su nombre y su consentimiento promocional.
func (s *Service) WhatsAppProfile(ctx context.Context, scope requestscope.Scope, phone string) (customercontracts.WhatsAppProfile, error) {
	digits, ok := whatsAppDigits(phone)
	if !ok {
		return customercontracts.WhatsAppProfile{}, invalidWhatsApp()
	}
	out, err := s.repository.WhatsAppProfile(ctx, scope.OrganizationID, digits)
	if err != nil {
		return out, unavailable("No pudimos consultar al cliente.")
	}
	return out, nil
}

// RecordWhatsAppConsent guarda la pregunta o la respuesta promocional con su
// fecha y el texto literal del cliente. Rechazar o retirar deja al cliente
// fuera de las campañas (marketing_consent=false).
func (s *Service) RecordWhatsAppConsent(ctx context.Context, scope requestscope.Scope, in customercontracts.ConsentEvidence) (customercontracts.WhatsAppProfile, error) {
	digits, ok := whatsAppDigits(in.CustomerPhone)
	in.ConversationID = strings.TrimSpace(in.ConversationID)
	in.RequestID = strings.TrimSpace(in.RequestID)
	in.Evidence = strings.TrimSpace(in.Evidence)
	if !ok || !customercontracts.ValidConsentDecision(in.Decision) || in.RequestID == "" || len(in.RequestID) > 160 ||
		len(in.ConversationID) > 160 || utf8.RuneCountInString(in.Evidence) > 500 {
		return customercontracts.WhatsAppProfile{}, invalidWhatsApp()
	}
	out, err := s.repository.RecordWhatsAppConsent(ctx, scope, digits, in)
	if errors.Is(err, ErrNotFound) {
		return out, notFound()
	}
	if err != nil {
		return out, unavailable("No pudimos registrar la preferencia del cliente.")
	}
	return out, nil
}

func whatsAppDigits(phone string) (string, bool) {
	var digits strings.Builder
	for _, c := range phone {
		if c >= '0' && c <= '9' {
			digits.WriteRune(c)
		}
	}
	value := digits.String()
	return value, len(value) >= 7 && len(value) <= 15
}
func invalidWhatsApp() error {
	return apperror.New(apperror.Invalid, "invalid_customer_contact", "Revisa el número y la respuesta del cliente.")
}
