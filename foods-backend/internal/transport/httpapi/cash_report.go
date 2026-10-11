package httpapi

import (
	"net/http"
)

func (a *API) getCashShiftReport(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.GetCashShiftReport(w, r)
}
