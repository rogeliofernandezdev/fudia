package httpapi

import (
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

// The integration suite exercises exactly the adapters composed by cmd/api.
func New(db *pgxpool.Pool) *API {
	return NewWithHandlers(db, Handlers{
		Identity:       identityhttp.New(identityapp.New(identitypg.New(db), credentials.Credentials{}), os.Getenv("FOODS_COOKIE_SECURE") == "true"),
		Customers:      customerhttp.New(customerapp.New(customerpg.New(db))),
		InventoryUnits: inventoryhttp.NewUnits(inventoryapp.NewUnits(inventorypg.NewUnits(db))),
	})
}
