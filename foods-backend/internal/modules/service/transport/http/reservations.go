package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/service/application"
	"github.com/foods-platform/foods-backend/internal/modules/service/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

type Reservations struct{ service *application.Reservations }

func NewReservations(service *application.Reservations) *Reservations { return &Reservations{service} }
func (h *Reservations) ListReservations(w http.ResponseWriter, r *http.Request) {
	page, size := shared.PageParams(r)
	out, err := h.service.List(r.Context(), floorScope(r), application.FloorFilter{Query: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status"), Page: page, Size: size})
	floorResponse(w, 200, out, err)
}
func (h *Reservations) UpdateReservationStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_reservation_status", "El estado de reserva no es válido.")
		return
	}
	err := h.service.SetStatus(r.Context(), floorScope(r), r.PathValue("id"), in.Status)
	floorResponse(w, 204, nil, err)
}
func (h *Reservations) CreateReservation(w http.ResponseWriter, r *http.Request) {
	var in domain.ReservationInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_reservation", "Revisa los datos de la reserva.")
		return
	}
	out, err := h.service.Create(r.Context(), floorScope(r), in)
	floorResponse(w, 201, out, err)
}
func (h *Reservations) UpdateReservation(w http.ResponseWriter, r *http.Request) {
	var in domain.ReservationInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_reservation", "Revisa los datos de la reserva.")
		return
	}
	out, err := h.service.Update(r.Context(), floorScope(r), r.PathValue("id"), in)
	floorResponse(w, 200, out, err)
}
