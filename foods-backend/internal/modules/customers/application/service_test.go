package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/customers/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"testing"
)

type memoryCustomers struct {
	organizationID, id string
	input              domain.Input
	filter             Filter
	err                error
	saves, audits      int
	scope              requestscope.Scope
}

func (r *memoryCustomers) List(_ context.Context, org string, f Filter) (Page, error) {
	r.organizationID = org
	r.filter = f
	return Page{Items: []domain.Customer{}}, r.err
}
func (r *memoryCustomers) Get(_ context.Context, org, id string) (domain.Customer, error) {
	r.organizationID = org
	r.id = id
	return domain.Customer{}, r.err
}
func (r *memoryCustomers) Save(_ context.Context, org, id string, in domain.Input) (domain.Customer, error) {
	r.organizationID = org
	r.id = id
	r.input = in
	r.saves++
	return domain.Customer{ID: "saved"}, r.err
}
func (r *memoryCustomers) SetActive(_ context.Context, org, id string, _ bool) error {
	r.organizationID = org
	r.id = id
	return r.err
}
func (r *memoryCustomers) Audit(_ context.Context, s requestscope.Scope, _, _ string) {
	r.audits++
	r.scope = s
}
func TestSaveValidatesBeforePersistenceAndKeepsTenant(t *testing.T) {
	r := &memoryCustomers{}
	s := New(r)
	scope := requestscope.Scope{OrganizationID: "org-a", LocationID: "loc-a", UserID: "user-a"}
	_, err := s.Save(context.Background(), scope, "", domain.Input{DisplayName: " "})
	if err == nil || r.saves != 0 {
		t.Fatal("invalid customer persisted")
	}
	_, err = s.Save(context.Background(), scope, "customer-a", domain.Input{DisplayName: " Ana "})
	if err != nil {
		t.Fatal(err)
	}
	if r.organizationID != "org-a" || r.id != "customer-a" || r.input.DisplayName != "Ana" || r.input.CustomerType != "person" || r.input.PreferredChannel != "none" || r.scope != scope || r.audits != 1 {
		t.Fatalf("wrong call: %#v", r)
	}
}
func TestFailedSaveDoesNotAuditSuccess(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code string
	}{{ErrNotFound, "customer_not_found"}, {ErrConflict, "customer_conflict"}, {ErrAddress, "invalid_address"}, {errors.New("secret database message"), "customer_unavailable"}} {
		r := &memoryCustomers{err: tc.err}
		_, err := New(r).Save(context.Background(), requestscope.Scope{OrganizationID: "org"}, "id", domain.Input{DisplayName: "Ana"})
		var e *apperror.Error
		if !errors.As(err, &e) || e.Code != tc.code || r.audits != 0 {
			t.Fatalf("failure mapping: %v audits=%d", err, r.audits)
		}
	}
}
func TestListBoundsPaginationAndTenant(t *testing.T) {
	r := &memoryCustomers{}
	_, err := New(r).List(context.Background(), "org-a", Filter{Page: -1, PageSize: 10000, Query: " Ana "})
	if err != nil || r.organizationID != "org-a" || r.filter.Page != 1 || r.filter.PageSize != 100 || r.filter.Query != "Ana" {
		t.Fatalf("invalid filter %#v %v", r, err)
	}
}
