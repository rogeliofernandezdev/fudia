package main

import (
	legacy "github.com/foods-platform/foods-backend/internal/legacy/httpapi"
	auditpg "github.com/foods-platform/foods-backend/internal/modules/audit/infrastructure/postgres"
	billingapp "github.com/foods-platform/foods-backend/internal/modules/billing/application"
	billingpg "github.com/foods-platform/foods-backend/internal/modules/billing/infrastructure/postgres"
	billinghttp "github.com/foods-platform/foods-backend/internal/modules/billing/transport/http"
	reportapp "github.com/foods-platform/foods-backend/internal/modules/reporting/application"
	reportpg "github.com/foods-platform/foods-backend/internal/modules/reporting/infrastructure/postgres"
	reporthttp "github.com/foods-platform/foods-backend/internal/modules/reporting/transport/http"
	serviceapp "github.com/foods-platform/foods-backend/internal/modules/service/application"
	servicepg "github.com/foods-platform/foods-backend/internal/modules/service/infrastructure/postgres"
	servicehttp "github.com/foods-platform/foods-backend/internal/modules/service/transport/http"
	subscriptionapp "github.com/foods-platform/foods-backend/internal/modules/subscriptions/application"
	subscriptionpg "github.com/foods-platform/foods-backend/internal/modules/subscriptions/infrastructure/postgres"
	"github.com/foods-platform/foods-backend/internal/platform/database"

	customerapp "github.com/foods-platform/foods-backend/internal/modules/customers/application"
	customerpg "github.com/foods-platform/foods-backend/internal/modules/customers/infrastructure/postgres"
	customerhttp "github.com/foods-platform/foods-backend/internal/modules/customers/transport/http"
	identityapp "github.com/foods-platform/foods-backend/internal/modules/identity/application"
	"github.com/foods-platform/foods-backend/internal/modules/identity/infrastructure/credentials"
	identitypg "github.com/foods-platform/foods-backend/internal/modules/identity/infrastructure/postgres"
	identityhttp "github.com/foods-platform/foods-backend/internal/modules/identity/transport/http"
	inventoryapp "github.com/foods-platform/foods-backend/internal/modules/inventory/application"
	inventorypg "github.com/foods-platform/foods-backend/internal/modules/inventory/infrastructure/postgres"
	inventoryhttp "github.com/foods-platform/foods-backend/internal/modules/inventory/transport/http"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
)

func newAPI(db *pgxpool.Pool) *legacy.API {
	return legacy.NewWithHandlers(db, legacy.Handlers{
		PaymentMethods:         billinghttp.NewMethods(billingapp.NewMethods(billingpg.NewMethods(db), database.NewUnitOfWork(db))),
		Floor:                  servicehttp.NewFloor(serviceapp.NewFloor(servicepg.NewFloor(db), database.NewUnitOfWork(db), auditpg.New(db))),
		Reservations:           servicehttp.NewReservations(serviceapp.NewReservations(servicepg.NewReservations(db), database.NewUnitOfWork(db), auditpg.New(db))),
		Reporting:              reporthttp.New(reportapp.New(reportpg.New(db), database.NewUnitOfWork(db), identitypg.NewAdministration(db))),
		Expenses:               billinghttp.NewExpenses(billingapp.NewExpenses(billingpg.NewExpenses(db), billingapp.NewMethods(billingpg.NewMethods(db), database.NewUnitOfWork(db)), auditpg.New(db))),
		IdentityAdministration: identityhttp.NewAdministration(identityapp.NewAdministration(identitypg.NewAdministration(db), database.NewUnitOfWork(db), credentials.Credentials{}, subscriptionapp.NewAccess(subscriptionpg.NewAccess(db)), auditpg.New(db))),
		Identity:               identityhttp.New(identityapp.New(identitypg.New(db), credentials.Credentials{}), os.Getenv("FOODS_COOKIE_SECURE") == "true"),
		Customers:              customerhttp.New(customerapp.New(customerpg.New(db))),
		InventoryUnits:         inventoryhttp.NewUnits(inventoryapp.NewUnits(inventorypg.NewUnits(db))),
	})
}
