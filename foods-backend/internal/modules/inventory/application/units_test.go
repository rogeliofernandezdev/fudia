package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"testing"
)

type memoryUnits struct {
	scope requestscope.Scope
	input domain.UnitInput
	calls int
	err   error
}

func (r *memoryUnits) List(context.Context, string) ([]domain.Unit, error) {
	return []domain.Unit{}, r.err
}
func (r *memoryUnits) Create(_ context.Context, s requestscope.Scope, in domain.UnitInput) (domain.Unit, error) {
	r.calls++
	r.scope = s
	r.input = in
	return domain.Unit{Code: in.Code, Name: in.Name}, r.err
}
func TestUnitCreationValidatesBeforeRepositoryAndPreservesScope(t *testing.T) {
	r := &memoryUnits{}
	s := NewUnits(r)
	scope := requestscope.Scope{OrganizationID: "org", LocationID: "loc", UserID: "user"}
	if _, err := s.Create(context.Background(), scope, domain.UnitInput{Code: "bad code", Name: "Name"}); err == nil || r.calls != 0 {
		t.Fatal("invalid input reached database")
	}
	out, err := s.Create(context.Background(), scope, domain.UnitInput{Code: " SACO ", Name: " Saco "})
	if err != nil || out.Code != "saco" || r.input.Name != "Saco" || r.scope != scope {
		t.Fatalf("normalization/scope: %#v %v", r, err)
	}
}
func TestUnitRepositoryFailuresRemainStable(t *testing.T) {
	for _, tc := range []struct {
		err  error
		kind apperror.Kind
		code string
	}{{ErrUnitConflict, apperror.Conflict, "inventory_unit_conflict"}, {ErrUnitPresentations, apperror.Unavailable, "inventory_units_unavailable"}, {errors.New("database secret"), apperror.Unavailable, "inventory_units_unavailable"}} {
		r := &memoryUnits{err: tc.err}
		_, err := NewUnits(r).Create(context.Background(), requestscope.Scope{}, domain.UnitInput{Code: "saco", Name: "Saco"})
		var e *apperror.Error
		if !errors.As(err, &e) || e.Kind != tc.kind || e.Code != tc.code {
			t.Fatalf("wrong mapping: %v", err)
		}
	}
}
