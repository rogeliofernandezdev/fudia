// Package httpapi composes module transports and applies route authorization.
package httpapi

import (
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

type API struct {
	handlers Handlers
}
type scope = requestscope.Scope
type scopeKey = requestscope.Key

func NewWithHandlers(handlers Handlers) *API {
	return &API{handlers: handlers}
}
