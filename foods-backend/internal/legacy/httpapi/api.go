// Package httpapi is the transitional adapter for verticals not yet extracted.
// Do not add new domain logic here; see docs/10_BACKEND_REFACTOR.md.
package httpapi

import (
	sharedhttp "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	db       *pgxpool.Pool
	handlers Handlers
}
type scope = requestscope.Scope
type scopeKey = requestscope.Key
type apiError = sharedhttp.ErrorResponse

func NewWithHandlers(db *pgxpool.Pool, handlers Handlers) *API {
	return &API{db: db, handlers: handlers}
}
