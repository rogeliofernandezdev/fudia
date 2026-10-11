package httpapi

import (
	"net/http"
)

type FloorHandler interface {
	ListZones(http.ResponseWriter, *http.Request)
	ListTables(http.ResponseWriter, *http.Request)
	GetTableByQR(http.ResponseWriter, *http.Request)
	RegenerateTableQR(http.ResponseWriter, *http.Request)
	CreateTablesBatch(http.ResponseWriter, *http.Request)
	CreateZone(http.ResponseWriter, *http.Request)
	UpdateZone(http.ResponseWriter, *http.Request)
	DeactivateZone(http.ResponseWriter, *http.Request)
	CreateTable(http.ResponseWriter, *http.Request)
	UpdateTable(http.ResponseWriter, *http.Request)
	DeactivateTable(http.ResponseWriter, *http.Request)
}

func (a *API) listZones(w http.ResponseWriter, r *http.Request)  { a.handlers.Floor.ListZones(w, r) }
func (a *API) listTables(w http.ResponseWriter, r *http.Request) { a.handlers.Floor.ListTables(w, r) }
func (a *API) getTableByQR(w http.ResponseWriter, r *http.Request) {
	a.handlers.Floor.GetTableByQR(w, r)
}
func (a *API) regenerateTableQR(w http.ResponseWriter, r *http.Request) {
	a.handlers.Floor.RegenerateTableQR(w, r)
}
func (a *API) createTablesBatch(w http.ResponseWriter, r *http.Request) {
	a.handlers.Floor.CreateTablesBatch(w, r)
}
func (a *API) createZone(w http.ResponseWriter, r *http.Request) { a.handlers.Floor.CreateZone(w, r) }
func (a *API) updateZone(w http.ResponseWriter, r *http.Request) { a.handlers.Floor.UpdateZone(w, r) }
func (a *API) deactivateZone(w http.ResponseWriter, r *http.Request) {
	a.handlers.Floor.DeactivateZone(w, r)
}
func (a *API) createTable(w http.ResponseWriter, r *http.Request) { a.handlers.Floor.CreateTable(w, r) }
func (a *API) updateTable(w http.ResponseWriter, r *http.Request) { a.handlers.Floor.UpdateTable(w, r) }
func (a *API) deactivateTable(w http.ResponseWriter, r *http.Request) {
	a.handlers.Floor.DeactivateTable(w, r)
}
