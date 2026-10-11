package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/customers/application"
	"github.com/foods-platform/foods-backend/internal/modules/customers/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

type Handler struct{ service *application.Service }

func New(service *application.Service) *Handler { return &Handler{service} }

var customerChannels = []map[string]string{{"value": "none", "label": "Sin preferencia"}, {"value": "whatsapp", "label": "WhatsApp"}, {"value": "phone", "label": "Llamada"}, {"value": "email", "label": "Correo electrónico"}}
var customerSegments = []map[string]string{{"value": "new", "label": "Nuevo"}, {"value": "recurrent", "label": "Recurrente"}, {"value": "frequent", "label": "Frecuente"}, {"value": "vip", "label": "VIP"}}
var customerTypes = []map[string]string{{"value": "person", "label": "Persona"}, {"value": "company", "label": "Empresa"}}
var customerDocumentTypes = []map[string]string{{"value": "", "label": "Sin documento"}, {"value": "DNI", "label": "DNI"}, {"value": "CE", "label": "Carné de extranjería"}, {"value": "RUC", "label": "RUC"}, {"value": "PASAPORTE", "label": "Pasaporte"}}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	s, ok := shared.Scope(w, r)
	if !ok {
		return
	}
	page, size := shared.PageParams(r)
	out, err := h.service.List(r.Context(), s.OrganizationID, application.Filter{Page: page, PageSize: size, Query: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status"), Segment: r.URL.Query().Get("segment")})
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, map[string]any{"items": out.Items, "total": out.Total, "page": page, "pageSize": size, "channelOptions": customerChannels, "segmentOptions": customerSegments, "customerTypeOptions": customerTypes, "documentTypeOptions": customerDocumentTypes})
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	s, ok := shared.Scope(w, r)
	if !ok {
		return
	}
	c, err := h.service.Get(r.Context(), s.OrganizationID, r.PathValue("id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, c)
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) { h.save(w, r, false) }
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) { h.save(w, r, true) }
func (h *Handler) save(w http.ResponseWriter, r *http.Request, updating bool) {
	s, ok := shared.Scope(w, r)
	if !ok {
		return
	}
	var in domain.Input
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_customer", "El nombre y los datos del cliente no son válidos.")
		return
	}
	id := ""
	status := 201
	if updating {
		id = r.PathValue("id")
		status = 200
	}
	c, err := h.service.Save(r.Context(), s, id, in)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, status, c)
}
func (h *Handler) SetActive(w http.ResponseWriter, r *http.Request) {
	s, ok := shared.Scope(w, r)
	if !ok {
		return
	}
	var in struct {
		Active bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_status", "El estado no es válido.")
		return
	}
	if err := h.service.SetActive(r.Context(), s, r.PathValue("id"), in.Active); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
