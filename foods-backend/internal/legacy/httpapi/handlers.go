package httpapi

import "net/http"

// Handlers injects the extracted verticals. Only cmd/api composes production adapters.
type Handlers struct {
	Identity       IdentityHandler
	Customers      CustomerHandler
	InventoryUnits UnitHandler
}
type IdentityHandler interface {
	Login(http.ResponseWriter, *http.Request)
	Logout(http.ResponseWriter, *http.Request)
	Auth(http.Handler) http.Handler
	RequirePermission(string, http.Handler) http.Handler
}
type CustomerHandler interface {
	List(http.ResponseWriter, *http.Request)
	Get(http.ResponseWriter, *http.Request)
	Create(http.ResponseWriter, *http.Request)
	Update(http.ResponseWriter, *http.Request)
	SetActive(http.ResponseWriter, *http.Request)
}
type UnitHandler interface {
	List(http.ResponseWriter, *http.Request)
	Create(http.ResponseWriter, *http.Request)
}

func (a *API) login(w http.ResponseWriter, r *http.Request)  { a.handlers.Identity.Login(w, r) }
func (a *API) logout(w http.ResponseWriter, r *http.Request) { a.handlers.Identity.Logout(w, r) }
func (a *API) auth(next http.Handler) http.Handler           { return a.handlers.Identity.Auth(next) }
func (a *API) requirePermission(permission string, next http.Handler) http.Handler {
	return a.handlers.Identity.RequirePermission(permission, next)
}
func (a *API) listCustomers(w http.ResponseWriter, r *http.Request) { a.handlers.Customers.List(w, r) }
func (a *API) getCustomer(w http.ResponseWriter, r *http.Request)   { a.handlers.Customers.Get(w, r) }
func (a *API) createCustomer(w http.ResponseWriter, r *http.Request) {
	a.handlers.Customers.Create(w, r)
}
func (a *API) updateCustomer(w http.ResponseWriter, r *http.Request) {
	a.handlers.Customers.Update(w, r)
}
func (a *API) updateCustomerStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.Customers.SetActive(w, r)
}
func (a *API) listInventoryUnits(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryUnits.List(w, r)
}
func (a *API) createInventoryUnit(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryUnits.Create(w, r)
}
