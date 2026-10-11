package httpapi

import (
	billingdomain "github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"net/http"
)

type cashReportSale = billingdomain.CashReportSale
type cashReportPayment = billingdomain.CashReportPayment
type cashReportCount = billingdomain.CashReportCount
type cashReportMethod = billingdomain.CashReportMethod
type cashShiftReport = billingdomain.CashShiftReport

func (a *API) getCashShiftReport(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.GetCashShiftReport(w, r)
}
