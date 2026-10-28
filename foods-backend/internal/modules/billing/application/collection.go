package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// PermissionAuthorizer reuses Identity's request-scoped authorization. A shift
// assignment is a narrow billing capability, never a role or cash.manage grant.
type PermissionAuthorizer interface {
	Authorize(context.Context, scope, string) error
}

func (a *Payments) WithAuthorization(authorizer PermissionAuthorizer) *Payments {
	a.authorizer = authorizer
	return a
}

func forbiddenPermission(err error) bool {
	var failure *apperror.Error
	return errors.As(err, &failure) && failure.Kind == apperror.Forbidden
}

func (a *Payments) CollectionAccess(ctx context.Context, s scope) (*domain.CollectionShift, error) {
	shift, err := a.repo.CollectionShift(ctx, s)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, paymentFailure(apperror.Unavailable, "cash_collection_unavailable", "No pudimos validar tu autorización de cobro.")
	}
	return shift, nil
}

func (a *Payments) CanCollect(ctx context.Context, s scope) (bool, error) {
	shift, err := a.CollectionAccess(ctx, s)
	return shift != nil, err
}

func (a *Payments) AuthorizeCollection(ctx context.Context, s scope) error {
	if a.authorizer != nil {
		err := a.authorizer.Authorize(ctx, s, "cash.manage")
		if err == nil {
			return nil
		}
		if !forbiddenPermission(err) {
			return err
		}
	}
	allowed, err := a.CanCollect(ctx, s)
	if err != nil {
		return err
	}
	if !allowed {
		return paymentFailure(apperror.Forbidden, "cash_collection_forbidden", "Necesitas autorización y un turno abierto para cobrar.")
	}
	return nil
}

// Called after locking the order, inside the payment transaction. The assignment
// cannot change between this check and persisting either a single or split charge.
func (a *Payments) requireCollectionOrder(ctx context.Context, s scope, id string) error {
	if a.authorizer == nil {
		return paymentFailure(apperror.Unavailable, "cash_collection_unavailable", "No pudimos validar tu autorización de cobro.")
	}
	err := a.authorizer.Authorize(ctx, s, "cash.manage")
	if err == nil || !forbiddenPermission(err) {
		return err
	}
	owned, err := a.repo.OwnsCollectionOrder(ctx, s, id)
	if err != nil {
		return paymentFailure(apperror.Unavailable, "cash_collection_unavailable", "No pudimos validar quién está atendiendo esta mesa.")
	}
	if !owned {
		return paymentFailure(apperror.Forbidden, "cash_collection_order_forbidden", "Solo puedes cobrar las mesas que estás atendiendo.")
	}
	return nil
}

// Existing financial readers retain their scope. Temporary operators see only
// their own table accounts and payments in their active shift.
func (a *Payments) AuthorizeCollectionRead(ctx context.Context, s scope) (string, error) {
	if a.authorizer == nil {
		shift, err := a.CollectionAccess(ctx, s)
		if err != nil {
			return "", err
		}
		if shift != nil {
			return shift.ID, nil
		}
		return "", paymentFailure(apperror.Forbidden, "cash_collection_forbidden", "Necesitas autorización para consultar los cobros.")
	}
	err := a.authorizer.Authorize(ctx, s, "cash.read")
	if err == nil {
		return "", nil
	}
	if !forbiddenPermission(err) {
		return "", err
	}
	shift, err := a.CollectionAccess(ctx, s)
	if err != nil {
		return "", err
	}
	if shift != nil {
		return shift.ID, nil
	}
	// Preserve pre-existing explicit orders.read access, without granting it to
	// somebody whose only authority came from a removed shift assignment.
	return "", a.authorizer.Authorize(ctx, s, "orders.read")
}

type collectionReadKey struct{}

func WithCollectionRead(ctx context.Context, shiftID string) context.Context {
	return context.WithValue(ctx, collectionReadKey{}, shiftID)
}
func collectionReadScope(ctx context.Context) string {
	shift, _ := ctx.Value(collectionReadKey{}).(string)
	return shift
}
