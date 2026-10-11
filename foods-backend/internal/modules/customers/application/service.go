package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/customers/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"strings"
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
	Save(context.Context, string, string, domain.Input) (domain.Customer, error)
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
	c, err := s.repository.Save(ctx, scope.OrganizationID, id, in)
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
