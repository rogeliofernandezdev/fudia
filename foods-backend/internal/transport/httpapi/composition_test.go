package httpapi

import (
	"context"
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
	deliveryapp "github.com/foods-platform/foods-backend/internal/modules/delivery/application"
	deliverypg "github.com/foods-platform/foods-backend/internal/modules/delivery/infrastructure/postgres"
	deliveryhttp "github.com/foods-platform/foods-backend/internal/modules/delivery/transport/http"
	einvoicingapp "github.com/foods-platform/foods-backend/internal/modules/einvoicing/application"
	einvoicingpg "github.com/foods-platform/foods-backend/internal/modules/einvoicing/infrastructure/postgres"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/jurisdictions/pe"
	einvoicingports "github.com/foods-platform/foods-backend/internal/modules/einvoicing/ports"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/providers/fake"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/providers/sunat"
	einvoicinghttp "github.com/foods-platform/foods-backend/internal/modules/einvoicing/transport/http"
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
	noticeapp "github.com/foods-platform/foods-backend/internal/modules/notifications/application"
	noticepg "github.com/foods-platform/foods-backend/internal/modules/notifications/infrastructure/postgres"
	noticehttp "github.com/foods-platform/foods-backend/internal/modules/notifications/transport/http"
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
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/secretbox"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"os"
)

// syncDispatcher sends the document before the payment response returns, so
// integration tests observe the final status deterministically.
type syncDispatcher struct{ documents *einvoicingapp.Documents }

func (d syncDispatcher) Dispatch(id string) { _ = d.documents.Process(context.Background(), id) }

// testOrderHandoff replica el adaptador de cmd/api que pasa a «en camino» el
// pedido recogido por una central.
type testOrderHandoff struct{ orders *orderapp.Orders }

func (h testOrderHandoff) MarkOnTheWay(ctx context.Context, s requestscope.Scope, orderID string) error {
	return h.orders.MarkDeliveryOnTheWayInTransaction(ctx, s, orderID)
}

func (h testOrderHandoff) MarkDelivered(ctx context.Context, s requestscope.Scope, orderID string) error {
	return h.orders.MarkDeliveryDeliveredInTransaction(ctx, s, orderID)
}

func (h testOrderHandoff) ReleaseDeliveryFee(ctx context.Context, s requestscope.Scope, orderID string) error {
	return h.orders.ReleaseDeliveryFeeInTransaction(ctx, s, orderID)
}

func (h testOrderHandoff) RestoreDeliveryFee(ctx context.Context, s requestscope.Scope, orderID string, fee float64) error {
	return h.orders.RestoreDeliveryFeeInTransaction(ctx, s, orderID, fee)
}

// The integration suite exercises exactly the adapters composed by cmd/api.
func New(db *pgxpool.Pool) *integrationAPI {
	identityService := identityapp.New(identitypg.New(db), credentials.Credentials{})
	notices := noticeapp.New(noticepg.New(db), identityService)
	orders := orderapp.New(orderpg.New(db), database.NewUnitOfWork(db), auditpg.New(db), inventoryapp.NewStock(inventorypg.NewStock(db), database.NewUnitOfWork(db), auditpg.New(db))).WithNotices(notices)
	dispatcher := deliveryapp.NewDispatcher(deliverypg.New(db), database.NewUnitOfWork(db)).WithNotices(notices).WithHandoff(testOrderHandoff{orders}).WithPanel(deliverypg.New(db))
	orders.WithDispatch(dispatcher)
	tips := billingapp.NewTips(billingpg.NewPayments(db), database.NewUnitOfWork(db))
	access := subscriptionapp.NewAccess(subscriptionpg.NewAccess(db))
	taxRepository := orgpg.NewTaxes(db)
	taxes := orgapp.NewTaxes(taxRepository)
	registry := einvoicingports.NewRegistry().WithJurisdiction(pe.New()).WithProvider(fake.New()).WithProvider(sunat.New().WithEndpoints(os.Getenv("FUDIA_TEST_SUNAT_URL"), os.Getenv("FUDIA_TEST_SUNAT_URL")))
	keyring, _ := secretbox.New(1, map[int][]byte{1: []byte("integration-test-master-key-32b!")})
	fiscal := einvoicingapp.NewDocuments(einvoicingpg.New(db), auditpg.New(db), orders, taxes, access, registry)
	fiscal.WithDispatcher(syncDispatcher{fiscal}).WithCipher(keyring)
	fiscalSettings := einvoicingapp.NewSettings(einvoicingpg.New(db), database.NewUnitOfWork(db), auditpg.New(db), access, registry).WithCipher(keyring)
	payments := billingapp.NewPayments(billingpg.NewPayments(db), database.NewUnitOfWork(db), auditpg.New(db), billingapp.NewMethods(billingpg.NewMethods(db), database.NewUnitOfWork(db)), orders).WithAuthorization(identityService).WithTips(tips).WithFiscal(fiscal)
	platform := platformapp.New(platformpg.New(db), database.NewUnitOfWork(db), orgapp.New(orgpg.New(db), database.NewUnitOfWork(db), subscriptionapp.NewAccess(subscriptionpg.NewAccess(db)), auditpg.New(db)), subscriptionapp.New(subscriptionpg.New(db), database.NewUnitOfWork(db), auditpg.New(db)), credentials.Credentials{}, platformdiag.Diagnostics{}, platformdiag.Diagnostics{}).WithTips(tips)
	concierge := conciergeapp.New(conciergepg.New(db), menuapp.NewCatalog(menupg.NewCatalog(db), database.NewUnitOfWork(db), auditpg.New(db), menufiles.Images{Directory: "./uploads/products"}), orders, auditpg.New(db)).WithCustomers(customerapp.New(customerpg.New(db))).WithNotices(notices).WithDirectory(conciergepg.New(db), platform, database.NewUnitOfWork(db)).WithDeliveryTracking(dispatcher)
	return &integrationAPI{db: db, fiscal: fiscal, dispatcher: dispatcher, API: NewWithHandlers(Handlers{
		Delivery:               deliveryhttp.New(deliveryapp.NewPartners(deliverypg.New(db), database.NewUnitOfWork(db)), dispatcher),
		Tips:                   billinghttp.NewTips(tips),
		Notices:                noticehttp.New(notices),
		RestaurantProfile:      orghttp.New(orgapp.New(orgpg.New(db), database.NewUnitOfWork(db), subscriptionapp.NewAccess(subscriptionpg.NewAccess(db)), auditpg.New(db))).WithProfiles(orgapp.NewRestaurantProfiles(orgpg.NewRestaurantProfiles(db), menuapp.NewCatalog(menupg.NewCatalog(db), database.NewUnitOfWork(db), auditpg.New(db), menufiles.Images{Directory: "./uploads/products"}).RestaurantMedia())),
		SpecialTaxes:           orghttp.NewSpecialTaxes(orgapp.NewSpecialTaxes(taxRepository, auditpg.New(db))),
		Organizations:          orghttp.New(orgapp.New(orgpg.New(db), database.NewUnitOfWork(db), subscriptionapp.NewAccess(subscriptionpg.NewAccess(db)), auditpg.New(db)).WithTips(tips)),
		EInvoicing:             einvoicinghttp.New(fiscal, fiscalSettings, func() string { return os.Getenv("FUDIA_EINVOICING_WORKER_KEY") }),
		Subscriptions:          subscriptionhttp.New(subscriptionapp.New(subscriptionpg.New(db), database.NewUnitOfWork(db), auditpg.New(db))),
		Platform:               platformhttp.New(platform),
		Concierge:              conciergehttp.New(concierge, func() string { return os.Getenv("FUDIA_CONCIERGE_API_KEY") }),
		ConciergeUsage:         conciergehttp.NewUsage(conciergeapp.NewUsage(conciergepg.NewUsage(db), conciergepg.New(db), database.NewUnitOfWork(db), auditpg.New(db)).WithTableGuard(concierge).WithOrigins(conciergepg.NewUsage(db), platform)),
		MenuCatalog:            menuhttp.NewCatalog(menuapp.NewCatalog(menupg.NewCatalog(db), database.NewUnitOfWork(db), auditpg.New(db), menufiles.Images{Directory: "./uploads/products"})),
		Payments:               billinghttp.NewPayments(payments),
		Cash:                   billinghttp.NewCash(billingapp.NewCash(billingpg.NewPayments(db), database.NewUnitOfWork(db)).WithTips(tips)),
		PaymentMethods:         billinghttp.NewMethods(billingapp.NewMethods(billingpg.NewMethods(db), database.NewUnitOfWork(db))),
		Floor:                  servicehttp.NewFloor(serviceapp.NewFloor(servicepg.NewFloor(db), database.NewUnitOfWork(db), auditpg.New(db))),
		Reservations:           servicehttp.NewReservations(serviceapp.NewReservations(servicepg.NewReservations(db), database.NewUnitOfWork(db), auditpg.New(db))),
		Reporting:              reporthttp.New(reportapp.New(reportpg.New(db), database.NewUnitOfWork(db), identitypg.NewAdministration(db))),
		Expenses:               billinghttp.NewExpenses(billingapp.NewExpenses(billingpg.NewExpenses(db), billingapp.NewMethods(billingpg.NewMethods(db), database.NewUnitOfWork(db)), auditpg.New(db))),
		Modules:                subscriptionhttp.NewModules(subscriptionapp.NewModules(subscriptionpg.NewAccess(db), subscriptionapp.NewAccess(subscriptionpg.NewAccess(db)), auditpg.New(db))),
		IdentityAdministration: identityhttp.NewAdministration(identityapp.NewAdministration(identitypg.NewAdministration(db), database.NewUnitOfWork(db), credentials.Credentials{}, subscriptionapp.NewAccess(subscriptionpg.NewAccess(db)), auditpg.New(db)).WithCashCollection(payments)),
		Identity:               identityhttp.New(identityService, os.Getenv("FOODS_COOKIE_SECURE") == "true"),
		Customers:              customerhttp.New(customerapp.New(customerpg.New(db))),
		InventoryStock:         inventoryhttp.NewStock(inventoryapp.NewStock(inventorypg.NewStock(db), database.NewUnitOfWork(db), auditpg.New(db)).WithProductNames(menuapp.NewCatalog(menupg.NewCatalog(db), database.NewUnitOfWork(db), auditpg.New(db), nil))),
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
	db     *pgxpool.Pool
	fiscal *einvoicingapp.Documents
	// dispatcher permite a las pruebas forzar el barrido de Delivery Fudia.
	dispatcher *deliveryapp.Dispatcher
}
