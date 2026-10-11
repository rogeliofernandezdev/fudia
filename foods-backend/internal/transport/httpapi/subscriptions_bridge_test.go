package httpapi

import (
	"context"
	subscriptionapp "github.com/foods-platform/foods-backend/internal/modules/subscriptions/application"
	subscriptiondomain "github.com/foods-platform/foods-backend/internal/modules/subscriptions/domain"
	subscriptionpg "github.com/foods-platform/foods-backend/internal/modules/subscriptions/infrastructure/postgres"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

type subscriptionPlanView = subscriptiondomain.SubscriptionPlanView
type organizationSubscriptionView = subscriptiondomain.OrganizationSubscriptionView

var subscriptionPeriodEnd = subscriptiondomain.SubscriptionPeriodEnd

func createOrganizationSubscription(ctx context.Context, tx pgx.Tx, organizationID, administratorID, planID, billingCycle string, termsAccepted bool) error {
	return subscriptionapp.New(subscriptionpg.New(nil), nil, nil).Create(database.WithTransaction(ctx, nil, tx), organizationID, administratorID, planID, billingCycle, termsAccepted)
}
