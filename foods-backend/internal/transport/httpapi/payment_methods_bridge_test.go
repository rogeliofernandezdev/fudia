package httpapi

import (
	"context"
	"errors"
	billingapp "github.com/foods-platform/foods-backend/internal/modules/billing/application"
	billingdomain "github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	billingpg "github.com/foods-platform/foods-backend/internal/modules/billing/infrastructure/postgres"
	platformpg "github.com/foods-platform/foods-backend/internal/modules/platformadmin/infrastructure/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type paymentMethodView = billingdomain.PaymentMethodView

func seedOrganizationPaymentMethods(ctx context.Context, pool *pgxpool.Pool, organizationID string) error {
	return platformpg.New(pool).SeedPaymentMethods(ctx, organizationID)
}
func getPaymentMethod(ctx context.Context, pool *pgxpool.Pool, organizationID, code, usage string) (paymentMethodView, error) {
	out, err := billingpg.NewMethods(pool).Find(ctx, organizationID, code, usage)
	if errors.Is(err, billingapp.ErrNotFound) {
		err = pgx.ErrNoRows
	}
	return out, err
}
