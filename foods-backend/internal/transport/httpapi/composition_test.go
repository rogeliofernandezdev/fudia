package httpapi

import (
	auditpg "github.com/foods-platform/foods-backend/internal/modules/audit/infrastructure/postgres"
	billingapp "github.com/foods-platform/foods-backend/internal/modules/billing/application"
	billingpg "github.com/foods-platform/foods-backend/internal/modules/billing/infrastructure/postgres"
	billinghttp "github.com/foods-platform/foods-backend/internal/modules/billing/transport/http"
	conciergeapp "github.com/foods-platform/foods-backend/internal/modules/concierge/application"
	conciergepg "github.com/foods-platform/foods-backend/internal/modules/concierge/infrastructure/postgres"
	conciergehttp "github.com/foods-platform/foods-backend/internal/modules/concierge/transport/http"
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
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"os"
)

// The integration suite exercises exactly the adapters composed by cmd/api.
func New(db *pgxpool.Pool) *integrationAPI {
	orders := orderapp.New(orderpg.New(db), database.NewUnitOfWork(db), auditpg.New(db), inventoryapp.NewStock(inventorypg.NewStock(db), database.NewUnitOfWork(db), auditpg.New(db)))
	identityService := identityapp.New(identitypg.New(db), credentials.Credentials{})
	tips := billingapp.NewTips(billingpg.NewPayments(db), database.NewUnitOfWork(db))
	payments := billingapp.NewPayments(billingpg.NewPayments(db), database.NewUnitOfWork(db), auditpg.New(db), billingapp.NewMethods(billingpg.NewMethods(db), database.NewUnitOfWork(db)), orders).WithAuthorization(identityService).WithTips(tips)
	return &integrationAPI{db: db, API: NewWithHandlers(Handlers{
		Tips:                   billinghttp.NewTips(tips),
		Organizations:          orghttp.New(orgapp.New(orgpg.New(db), database.NewUnitOfWork(db), subscriptionapp.NewAccess(subscriptionpg.NewAccess(db)), auditpg.New(db))),
		Subscriptions:          subscriptionhttp.New(subscriptionapp.New(subscriptionpg.New(db), database.NewUnitOfWork(db), auditpg.New(db))),
		Platform:               platformhttp.New(platformapp.New(platformpg.New(db), database.NewUnitOfWork(db), orgapp.New(orgpg.New(db), database.NewUnitOfWork(db), subscriptionapp.NewAccess(subscriptionpg.NewAccess(db)), auditpg.New(db)), subscriptionapp.New(subscriptionpg.New(db), database.NewUnitOfWork(db), auditpg.New(db)), credentials.Credentials{}, platformdiag.Diagnostics{}, platformdiag.Diagnostics{})),
		Concierge:              conciergehttp.New(conciergeapp.New(conciergepg.New(db), menuapp.NewCatalog(menupg.NewCatalog(db), database.NewUnitOfWork(db), auditpg.New(db), menufiles.Images{Directory: "./uploads/products"}), orders, auditpg.New(db)), func() string { return os.Getenv("FUDIA_CONCIERGE_API_KEY") }),
		ConciergeUsage:         conciergehttp.NewUsage(conciergeapp.NewUsage(conciergepg.NewUsage(db), conciergepg.New(db), database.NewUnitOfWork(db), auditpg.New(db))),
		MenuCatalog:            menuhttp.NewCatalog(menuapp.NewCatalog(menupg.NewCatalog(db), database.NewUnitOfWork(db), auditpg.New(db), menufiles.Images{Directory: "./uploads/products"})),
		Payments:               billinghttp.NewPayments(payments),
		Cash:                   billinghttp.NewCash(billingapp.NewCash(billingpg.NewPayments(db), database.NewUnitOfWork(db))),
		PaymentMethods:         billinghttp.NewMethods(billingapp.NewMethods(billingpg.NewMethods(db), database.NewUnitOfWork(db))),
		Floor:                  servicehttp.NewFloor(serviceapp.NewFloor(servicepg.NewFloor(db), database.NewUnitOfWork(db), auditpg.New(db))),
		Reservations:           servicehttp.NewReservations(serviceapp.NewReservations(servicepg.NewReservations(db), database.NewUnitOfWork(db), auditpg.New(db))),
		Reporting:              reporthttp.New(reportapp.New(reportpg.New(db), database.NewUnitOfWork(db), identitypg.NewAdministration(db))),
		Expenses:               billinghttp.NewExpenses(billingapp.NewExpenses(billingpg.NewExpenses(db), billingapp.NewMethods(billingpg.NewMethods(db), database.NewUnitOfWork(db)), auditpg.New(db))),
		Modules:                subscriptionhttp.NewModules(subscriptionapp.NewModules(subscriptionpg.NewAccess(db), subscriptionapp.NewAccess(subscriptionpg.NewAccess(db)), auditpg.New(db))),
		IdentityAdministration: identityhttp.NewAdministration(identityapp.NewAdministration(identitypg.NewAdministration(db), database.NewUnitOfWork(db), credentials.Credentials{}, subscriptionapp.NewAccess(subscriptionpg.NewAccess(db)), auditpg.New(db)).WithCashCollection(payments)),
		Identity:               identityhttp.New(identityService, os.Getenv("FOODS_COOKIE_SECURE") == "true"),
		Customers:              customerhttp.New(customerapp.New(customerpg.New(db))),
		InventoryStock:         inventoryhttp.NewStock(inventoryapp.NewStock(inventorypg.NewStock(db), database.NewUnitOfWork(db), auditpg.New(db))),
		Purchasing:             purchasehttp.New(purchaseapp.New(purchasepg.New(db), database.NewUnitOfWork(db), auditpg.New(db), inventoryapp.NewPurchaseLedger(inventorypg.NewStock(db)))),
		Orders:                 orderhttp.New(orders),
		InventoryUnits:         inventoryhttp.NewUnits(inventoryapp.NewUnits(inventorypg.NewUnits(db))),
	})}
}

// Adapter probe used by the historical-label integration regression.
func (a *integrationAPI) getPaymentByID(r *http.Request, id string) (paymentView, error) {
	s := r.Context().Value(scopeKey{}).(scope)
	return billingpg.NewPayments(a.db).Payment(r.Context(), s, id)
}

type integrationAPI struct {
	*API
	db *pgxpool.Pool
}
