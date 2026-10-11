package httpapi

import (
	"net/http"
)

type ReportingHandler interface {
	Dashboard(http.ResponseWriter, *http.Request)
}

func (a *API) dashboard(w http.ResponseWriter, r *http.Request) { a.handlers.Reporting.Dashboard(w, r) }
