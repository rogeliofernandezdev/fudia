package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"testing"
)

type collectionRepo struct {
	PaymentRepository
	shift *domain.CollectionShift
	err   error
	calls int
}

func (r *collectionRepo) CollectionShift(context.Context, scope) (*domain.CollectionShift, error) {
	r.calls++
	return r.shift, r.err
}

type collectionPermissions struct {
	allowed map[string]bool
	err     error
}

func (p collectionPermissions) Authorize(_ context.Context, _ scope, permission string) error {
	if p.err != nil {
		return p.err
	}
	if p.allowed[permission] {
		return nil
	}
	return paymentFailure(apperror.Forbidden, "forbidden", "Sin permiso")
}
func TestCollectionUsesAssignmentWithoutAdministrativePermission(t *testing.T) {
	repo := &collectionRepo{shift: &domain.CollectionShift{ID: "shift"}}
	permissions := collectionPermissions{allowed: map[string]bool{"orders.read": true}}
	service := NewPayments(repo, nil, nil, nil, nil).WithAuthorization(permissions)
	if err := service.AuthorizeCollection(context.Background(), scope{}); err != nil {
		t.Fatal(err)
	}
	if permissions.allowed["cash.manage"] {
		t.Fatal("assignment changed permanent permissions")
	}
	shift, err := service.AuthorizeCollectionRead(context.Background(), scope{})
	if err != nil || shift != "shift" {
		t.Fatalf("temporary read scope: %q %v", shift, err)
	}
	repo.shift = nil
	repo.err = ErrNotFound
	if err := service.AuthorizeCollection(context.Background(), scope{}); !forbiddenPermission(err) {
		t.Fatalf("removed assignment allowed: %v", err)
	}
}
func TestCollectionFailsClosedOnAuthorizationAndAssignmentErrors(t *testing.T) {
	repo := &collectionRepo{shift: &domain.CollectionShift{ID: "shift"}}
	failure := paymentFailure(apperror.Unavailable, "permissions_unavailable", "No disponible")
	service := NewPayments(repo, nil, nil, nil, nil).WithAuthorization(collectionPermissions{err: failure})
	if err := service.AuthorizeCollection(context.Background(), scope{}); !errors.Is(err, failure) || repo.calls != 0 {
		t.Fatal("permission outage must not fall back to assignment")
	}
	service.WithAuthorization(collectionPermissions{})
	repo.err = errors.New("database unavailable")
	if err := service.AuthorizeCollection(context.Background(), scope{}); err == nil || forbiddenPermission(err) {
		t.Fatalf("expected explicit unavailable, got %v", err)
	}
}
func TestCollectionAdministrativeAuthorizationRemainsUnchanged(t *testing.T) {
	repo := &collectionRepo{err: ErrNotFound}
	service := NewPayments(repo, nil, nil, nil, nil).WithAuthorization(collectionPermissions{allowed: map[string]bool{"cash.manage": true, "cash.read": true}})
	if err := service.AuthorizeCollection(context.Background(), scope{}); err != nil {
		t.Fatal(err)
	}
	shift, err := service.AuthorizeCollectionRead(context.Background(), scope{})
	if err != nil || shift != "" || repo.calls != 0 {
		t.Fatalf("administrative scope: %q %v calls=%d", shift, err, repo.calls)
	}
}

func (*collectionRepo) CanReadAllShifts(context.Context, scope) (bool, error) { return false, nil }
