package httpapi

import (
	reportdomain "github.com/foods-platform/foods-backend/internal/modules/reporting/domain"
	"net/http"
)

type dashboardOperations = reportdomain.DashboardOperations
type dashboardPeriod = reportdomain.DashboardPeriod
type dashboardPeriodView = reportdomain.DashboardPeriodView
type dashboardPreviousView = reportdomain.DashboardPreviousView
type dashboardTrendPoint = reportdomain.DashboardTrendPoint
type dashboardTrend = reportdomain.DashboardTrend
type dashboardBreakdown = reportdomain.DashboardBreakdown
type dashboardCategorySale = reportdomain.DashboardCategorySale
type dashboardInsights = reportdomain.DashboardInsights

var dashboardPeriodByKey = reportdomain.DashboardPeriodByKey

type ReportingHandler interface {
	Dashboard(http.ResponseWriter, *http.Request)
}

func (a *API) dashboard(w http.ResponseWriter, r *http.Request) { a.handlers.Reporting.Dashboard(w, r) }
