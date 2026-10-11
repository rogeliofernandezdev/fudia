package main

import (
	auditpg "github.com/foods-platform/foods-backend/internal/modules/audit/infrastructure/postgres"
	billingapp "github.com/foods-platform/foods-backend/internal/modules/billing/application"
	billingpg "github.com/foods-platform/foods-backend/internal/modules/billing/infrastructure/postgres"
	billinghttp "github.com/foods-platform/foods-backend/internal/modules/billing/transport/http"
	conciergeapp "github.com/foods-platform/foods-backend/internal/modules/concierge/application"
	conciergepg "github.com/foods-platform/foods-backend/internal/modules/concierge/infrastructure/postgres"
	conciergehttp "github.com/foods-platform/foods-backend/internal/modules/concierge/transport/http"
	menuapp "github.com/foods-platform/foods-backend/internal/modules/menu/application"
	menufiles "github.com/foods-platform/foods-backend/internal/modules/menu/infrastructure/files"
	menupg "github.com/foods-platform/foods-backend/internal/modules/menu/infrastructure/postgres"
	menuhttp "github.com/foods-platform/foods-backend/internal/modules/menu/transport/http"
	orderapp "github.com/foods-platform/foods-backend/internal/modules/orders/application"
	orderpg "github.com/foods-platform/foods-backend/internal/modules/orders/infrastructure/postgres"
	orderhttp "github.com/foods-platform/foods-backend/internal/modules/orders/transport/http"
	orgapp "github.com/foods-platform/foods-backend/internal/modules/organizations/application"
	orgpg "github.com/foods-platform/foods-backend/internal/modules/organizations/infrastructure/postgres"
	orghttp "github.com/foods-platform/foods-backend/internal/modules/organizations/transport/http"
	platformapp "github.com/foods-platform/foods-backend/internal/modules/platformadmin/application"
	platformdiag "github.com/foods-platform/foods-backend/internal/modules/platformadmin/infrastructure/diagnostics"
	platformpg "github.com/foods-platform/foods-backend/internal/modules/platformadmin/infrastructure/postgres"
	platformhttp "github.com/foods-platform/foods-backend/internal/modules/platformadmin/transport/http"
	purchaseapp "github.com/foods-platform/foods-backend/internal/modules/purchasing/application"
	purchasepg "github.com/foods-platform/foods-backend/internal/modules/purchasing/infrastructure/postgres"
	purchasehttp "github.com/foods-platform/foods-backend/internal/modules/purchasing/transport/http"
	reportapp "github.com/foods-platform/foods-backend/internal/modules/reporting/application"
	reportpg "github.com/foods-platform/foods-backend/internal/modules/reporting/infrastructure/postgres"
	reporthttp "github.com/foods-platform/foods-backend/internal/modules/reporting/transport/http"
	serviceapp "github.com/foods-platform/foods-backend/internal/modules/service/application"
	servicepg "github.com/foods-platform/foods-backend/internal/modules/service/infrastructure/postgres"
	servicehttp "github.com/foods-platform/foods-backend/internal/modules/service/transport/http"
	subscriptionapp "github.com/foods-platform/foods-backend/internal/modules/subscriptions/application"
	subscriptionpg "github.com/foods-platform/foods-backend/internal/modules/subscriptions/infrastructure/postgres"
	subscriptionhttp "github.com/foods-platform/foods-backend/internal/modules/subscriptions/transport/http"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	httpapi "github.com/foods-platform/foods-backend/internal/transport/httpapi"

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

func newAPI(db *pgxpool.Pool) *httpapi.API {
	// Composition is the only layer that knows concrete PostgreSQL adapters.
	// A single unit of work and shared stateless adapters preserve transactions
	// across Orders, Inventory, Billing and tenant onboarding.
	work := database.NewUnitOfWork(db)
	audit := auditpg.New(db)
	inventoryRepo := inventorypg.NewStock(db)
	stock := inventoryapp.NewStock(inventoryRepo, work, audit)
	menu := menuapp.NewCatalog(menupg.NewCatalog(db), work, audit, menufiles.Images{Directory: "./uploads/products"})
	orders := orderapp.New(orderpg.New(db), work, audit, stock)

	accessRepo := subscriptionpg.NewAccess(db)
	access := subscriptionapp.NewAccess(accessRepo)
	subscriptions := subscriptionapp.New(subscriptionpg.New(db), work, audit)
	organizations := orgapp.New(orgpg.New(db), work, access, audit)
	platform := platformapp.New(platformpg.New(db), work, organizations, subscriptions,
		credentials.Credentials{}, platformdiag.Diagnostics{}, platformdiag.Diagnostics{})

	paymentsRepo := billingpg.NewPayments(db)
	methods := billingapp.NewMethods(billingpg.NewMethods(db), work)
	identityAdminRepo := identitypg.NewAdministration(db)

	return httpapi.NewWithHandlers(httpapi.Handlers{
		Organizations:          orghttp.New(organizations),
		Subscriptions:          subscriptionhttp.New(subscriptions),
		Platform:               platformhttp.New(platform),
		Concierge:              conciergehttp.New(conciergeapp.New(conciergepg.New(db), menu, orders, audit), func() string { return os.Getenv("FUDIA_CONCIERGE_API_KEY") }),
		MenuCatalog:            menuhttp.NewCatalog(menu),
		Payments:               billinghttp.NewPayments(billingapp.NewPayments(paymentsRepo, work, audit, methods, orders)),
		Cash:                   billinghttp.NewCash(billingapp.NewCash(paymentsRepo, work)),
		PaymentMethods:         billinghttp.NewMethods(methods),
		Floor:                  servicehttp.NewFloor(serviceapp.NewFloor(servicepg.NewFloor(db), work, audit)),
		Reservations:           servicehttp.NewReservations(serviceapp.NewReservations(servicepg.NewReservations(db), work, audit)),
		Reporting:              reporthttp.New(reportapp.New(reportpg.New(db), work, identityAdminRepo)),
		Expenses:               billinghttp.NewExpenses(billingapp.NewExpenses(billingpg.NewExpenses(db), methods, audit)),
		Modules:                subscriptionhttp.NewModules(subscriptionapp.NewModules(accessRepo, access, audit)),
		IdentityAdministration: identityhttp.NewAdministration(identityapp.NewAdministration(identityAdminRepo, work, credentials.Credentials{}, access, audit)),
		Identity:               identityhttp.New(identityapp.New(identitypg.New(db), credentials.Credentials{}), os.Getenv("FOODS_COOKIE_SECURE") == "true"),
		Customers:              customerhttp.New(customerapp.New(customerpg.New(db))),
		InventoryStock:         inventoryhttp.NewStock(stock),
		Purchasing:             purchasehttp.New(purchaseapp.New(purchasepg.New(db), work, audit, inventoryapp.NewPurchaseLedger(inventoryRepo))),
		Orders:                 orderhttp.New(orders),
		InventoryUnits:         inventoryhttp.NewUnits(inventoryapp.NewUnits(inventorypg.NewUnits(db))),
	})
}
