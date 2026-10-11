package httpapi

import (
	servicedomain "github.com/foods-platform/foods-backend/internal/modules/service/domain"
	"net/http"
)

type reservationView = servicedomain.ReservationView
type reservationInput = servicedomain.ReservationInput

var validReservationStatus = servicedomain.ValidReservationStatus
var validReservationTransition = servicedomain.ValidReservationTransition
var normalizeOptionalID = servicedomain.NormalizeOptionalID
var parseReservationInput = servicedomain.ParseReservationInput

type ReservationsHandler interface {
	ListReservations(http.ResponseWriter, *http.Request)
	CreateReservation(http.ResponseWriter, *http.Request)
	UpdateReservation(http.ResponseWriter, *http.Request)
	UpdateReservationStatus(http.ResponseWriter, *http.Request)
}

func (a *API) listReservations(w http.ResponseWriter, r *http.Request) {
	a.handlers.Reservations.ListReservations(w, r)
}
func (a *API) createReservation(w http.ResponseWriter, r *http.Request) {
	a.handlers.Reservations.CreateReservation(w, r)
}
func (a *API) updateReservation(w http.ResponseWriter, r *http.Request) {
	a.handlers.Reservations.UpdateReservation(w, r)
}
func (a *API) updateReservationStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.Reservations.UpdateReservationStatus(w, r)
}
