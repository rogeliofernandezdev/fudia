package httpapi

import (
	orderpg "github.com/foods-platform/foods-backend/internal/modules/orders/infrastructure/postgres"
	"net/http"
)

func (a *integrationAPI) serviceKitchenTickets(r *http.Request, channel, destination string) ([]kitchenTicket, error) {
	s := r.Context().Value(scopeKey{}).(scope)
	return orderpg.New(a.db).ServiceTickets(r.Context(), s, channel, destination)
}
