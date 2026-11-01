package http

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/application"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

type Handler struct {
	documents *application.Documents
	settings  *application.Settings
	workerKey func() string
}

func New(documents *application.Documents, settings *application.Settings, workerKey func() string) *Handler {
	return &Handler{documents, settings, workerKey}
}

func sessionScope(r *http.Request) requestscope.Scope {
	s, _ := requestscope.From(r.Context())
	return s
}
func respond(w http.ResponseWriter, status int, out any, err error) {
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, status, out)
}

func (h *Handler) Settings(w http.ResponseWriter, r *http.Request) {
	out, err := h.settings.View(r.Context(), sessionScope(r))
	respond(w, 200, out, err)
}
func (h *Handler) SaveProfile(w http.ResponseWriter, r *http.Request) {
	var in application.ProfileSettingsInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.settings.SaveProfile(r.Context(), sessionScope(r), r.PathValue("id"), in)
	respond(w, 200, out, err)
}
func (h *Handler) SaveLocation(w http.ResponseWriter, r *http.Request) {
	var in application.LocationSettingsInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.settings.SaveLocation(r.Context(), sessionScope(r), r.PathValue("id"), in)
	respond(w, 200, out, err)
}
func (h *Handler) SaveCredentials(w http.ResponseWriter, r *http.Request) {
	var in application.CredentialsInput
	// A .pfx is a few KB; base64 adds a third.
	r.Body = http.MaxBytesReader(w, r.Body, 512<<10)
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.settings.SaveCredentials(r.Context(), sessionScope(r), r.PathValue("id"), in)
	respond(w, 200, out, err)
}
func (h *Handler) File(w http.ResponseWriter, r *http.Request) {
	file, err := h.documents.File(r.Context(), sessionScope(r), r.PathValue("id"), r.PathValue("kind"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+file.Name+`"`)
	w.Header().Set("Cache-Control", "private, no-store")
	_, _ = w.Write(file.Content)
}
func (h *Handler) CheckoutOptions(w http.ResponseWriter, r *http.Request) {
	out, err := h.documents.CheckoutOptions(r.Context(), sessionScope(r))
	respond(w, 200, out, err)
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, size := shared.PageParams(r)
	query := r.URL.Query()
	out, err := h.documents.List(r.Context(), sessionScope(r), application.DocumentFilter{Status: strings.TrimSpace(query.Get("status")), Query: query.Get("q"), Page: page, Size: size})
	respond(w, 200, out, err)
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	out, err := h.documents.Get(r.Context(), sessionScope(r), r.PathValue("id"))
	respond(w, 200, out, err)
}
func (h *Handler) Retry(w http.ResponseWriter, r *http.Request) {
	out, err := h.documents.Retry(r.Context(), sessionScope(r), r.PathValue("id"))
	respond(w, 200, out, err)
}

// WorkerAuth protects the internal endpoints called by Cloud Tasks and Cloud
// Scheduler. They must also stay private in Cloud Run (OIDC invoker).
func (h *Handler) WorkerAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := strings.TrimSpace(h.workerKey())
		provided := strings.TrimSpace(r.Header.Get("X-Fudia-Worker-Key"))
		if expected == "" {
			shared.Fail(w, 503, "einvoicing_worker_not_configured", "El envío de comprobantes no está configurado.")
			return
		}
		if len(expected) != len(provided) || subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
			shared.Fail(w, 401, "einvoicing_worker_unauthorized", "Credencial del worker inválida.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (h *Handler) ProcessDocument(w http.ResponseWriter, r *http.Request) {
	if err := h.documents.Process(r.Context(), r.PathValue("id")); err != nil {
		shared.Fail(w, 503, "einvoicing_process_failed", "No pudimos enviar el comprobante; se reintentará.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) Sweep(w http.ResponseWriter, r *http.Request) {
	processed, err := h.documents.Sweep(r.Context(), 50)
	if err != nil {
		shared.Fail(w, 503, "einvoicing_sweep_failed", "No pudimos revisar los comprobantes pendientes.")
		return
	}
	shared.WriteJSON(w, 200, map[string]int{"processed": processed})
}
