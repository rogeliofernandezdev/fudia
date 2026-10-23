package httpapi

import "net/http"

func (a *API) openConciergeConversation(w http.ResponseWriter, r *http.Request) {
	a.handlers.ConciergeUsage.OpenConciergeConversation(w, r)
}
func (a *API) reportConciergeUsage(w http.ResponseWriter, r *http.Request) {
	a.handlers.ConciergeUsage.ReportConciergeUsage(w, r)
}
func (a *API) getConciergeUsageSettings(w http.ResponseWriter, r *http.Request) {
	a.handlers.ConciergeUsage.GetConciergeUsageSettings(w, r)
}
func (a *API) updateConciergeUsageSettings(w http.ResponseWriter, r *http.Request) {
	a.handlers.ConciergeUsage.UpdateConciergeUsageSettings(w, r)
}
func (a *API) listConciergeUsage(w http.ResponseWriter, r *http.Request) {
	a.handlers.ConciergeUsage.ListConciergeUsage(w, r)
}
func (a *API) getOrganizationConciergeUsage(w http.ResponseWriter, r *http.Request) {
	a.handlers.ConciergeUsage.GetOrganizationConciergeUsage(w, r)
}
func (a *API) updateOrganizationConciergeLimit(w http.ResponseWriter, r *http.Request) {
	a.handlers.ConciergeUsage.UpdateOrganizationConciergeLimit(w, r)
}
