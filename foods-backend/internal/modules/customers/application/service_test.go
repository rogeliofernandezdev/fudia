package application

import (
	"context"
	"errors"
	customercontracts "github.com/foods-platform/foods-backend/internal/modules/customers/contracts"
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
func (r *memoryCustomers) Save(_ context.Context, s requestscope.Scope, id string, in domain.Input) (domain.Customer, error) {
	r.organizationID = s.OrganizationID
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
func (r *memoryCustomers) WhatsAppProfile(_ context.Context, org, digits string) (customercontracts.WhatsAppProfile, error) {
	r.organizationID = org
	r.id = digits
	return customercontracts.WhatsAppProfile{MarketingConsent: customercontracts.ConsentPending}, r.err
}
func (r *memoryCustomers) RecordWhatsAppConsent(_ context.Context, s requestscope.Scope, digits string, in customercontracts.ConsentEvidence) (customercontracts.WhatsAppProfile, error) {
	r.scope = s
	r.id = digits
	r.saves++
	return customercontracts.WhatsAppProfile{Known: true, MarketingConsent: in.Decision}, r.err
}

func TestWhatsAppConsentValidatesBeforePersistence(t *testing.T) {
	r := &memoryCustomers{}
	s := New(r)
	scope := requestscope.Scope{OrganizationID: "org", LocationID: "loc"}
	for _, in := range []customercontracts.ConsentEvidence{
		{CustomerPhone: "123", RequestID: "r1", Decision: customercontracts.ConsentGranted},
		{CustomerPhone: "+51 987 654 321", RequestID: "r1", Decision: "maybe"},
		{CustomerPhone: "+51 987 654 321", RequestID: " ", Decision: customercontracts.ConsentGranted},
	} {
		var failure *apperror.Error
		if _, err := s.RecordWhatsAppConsent(context.Background(), scope, in); !errors.As(err, &failure) || failure.Code != "invalid_customer_contact" {
			t.Fatalf("invalid consent must be rejected: %+v %v", in, err)
		}
	}
	if r.saves != 0 {
		t.Fatal("invalid consent reached persistence")
	}
	out, err := s.RecordWhatsAppConsent(context.Background(), scope, customercontracts.ConsentEvidence{CustomerPhone: "+51 987 654 321", RequestID: "r1", Decision: customercontracts.ConsentDeclined, Evidence: " No, gracias "})
	if err != nil || out.MarketingConsent != customercontracts.ConsentDeclined || r.id != "51987654321" || r.scope.OrganizationID != "org" {
		t.Fatalf("consent keeps tenant and phone digits: %+v %v %+v", out, err, r)
	}
	r.err = ErrNotFound
	var failure *apperror.Error
	if _, err = s.RecordWhatsAppConsent(context.Background(), scope, customercontracts.ConsentEvidence{CustomerPhone: "987654321", RequestID: "r2", Decision: customercontracts.ConsentAsked}); !errors.As(err, &failure) || failure.Code != "customer_not_found" {
		t.Fatalf("unknown customer: %v", err)
	}
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
func TestSaveRejectsIncompleteOrOutOfRangeAddressPoint(t *testing.T) {
	lat, lng, far := -12.0464, -77.0428, 120.0
	for _, address := range []domain.Address{
		{Address: "Av. Arequipa 1234", Latitude: &lat},
		{Address: "Av. Arequipa 1234", Longitude: &lng},
		{Address: "Av. Arequipa 1234", Latitude: &far, Longitude: &lng},
	} {
		r := &memoryCustomers{}
		if _, err := New(r).Save(context.Background(), requestscope.Scope{OrganizationID: "org"}, "", domain.Input{DisplayName: "Ana", Addresses: []domain.Address{address}}); err == nil || r.saves != 0 {
			t.Fatalf("invalid point persisted: %#v", address)
		}
	}
	r := &memoryCustomers{}
	if _, err := New(r).Save(context.Background(), requestscope.Scope{OrganizationID: "org"}, "", domain.Input{DisplayName: "Ana", Addresses: []domain.Address{{Address: "Av. Arequipa 1234", Latitude: &lat, Longitude: &lng}}}); err != nil {
		t.Fatal(err)
	}
	if got := r.input.Addresses[0]; got.Latitude == nil || *got.Latitude != lat || *got.Longitude != lng {
		t.Fatalf("point lost: %#v", got)
	}
}
