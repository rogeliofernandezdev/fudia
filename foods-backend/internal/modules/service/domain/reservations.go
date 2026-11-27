package domain

import (
	"regexp"
	"strings"
	"time"
)

type ReservationView struct {
	ID              string  `json:"id"`
	CustomerID      *string `json:"customerId,omitempty"`
	CustomerName    string  `json:"customerName"`
	CustomerPhone   string  `json:"customerPhone"`
	StartsAt        string  `json:"startsAt"`
	Guests          int     `json:"guests"`
	DurationMinutes int     `json:"durationMinutes"`
	// ToleranceMinutes: minutos que se espera al cliente después de StartsAt.
	ToleranceMinutes int     `json:"toleranceMinutes"`
	TableID          *string `json:"tableId,omitempty"`
	TableName        string  `json:"tableName"`
	Status           string  `json:"status"`
	// DepositAmount: garantía de la reserva («0.00» = sin costo, el valor por defecto).
	DepositAmount string `json:"depositAmount"`
	// Expired: el barrido la pasó a «No asistió» al vencer la tolerancia.
	Expired bool `json:"expired"`
	// DepositStatus: none (sin garantía), pending (por cobrar), paid (cobrada,
	// espera al cliente), applied (descontada de su cuenta), retained (quedó en
	// el local) o refunded (devuelta).
	DepositStatus string `json:"depositStatus"`
	// DepositMethodName: medio con que se cobró la garantía.
	DepositMethodName string `json:"depositMethodName"`
	// DepositDocument: serie-correlativo de la boleta de una garantía retenida.
	DepositDocument string `json:"depositDocument"`
	Notes     string `json:"notes"`
	CreatedAt string `json:"createdAt"`
}
type ReservationInput struct {
	CustomerID      *string `json:"customerId,omitempty"`
	CustomerName    string  `json:"customerName"`
	CustomerPhone   string  `json:"customerPhone"`
	StartsAt        string  `json:"startsAt"`
	Guests          int     `json:"guests"`
	DurationMinutes int     `json:"durationMinutes"`
	// ToleranceMinutes nil toma la tolerancia configurada en el local.
	ToleranceMinutes *int    `json:"toleranceMinutes,omitempty"`
	// DepositAmount nil toma la garantía del local al crear y conserva la actual
	// al editar; "0" quita la garantía.
	DepositAmount *string `json:"depositAmount,omitempty"`
	TableID       *string `json:"tableId,omitempty"`
	Notes         string  `json:"notes"`
}

var depositPattern = regexp.MustCompile(`^[0-9]{1,10}([.][0-9]{1,2})?$`)

// ValidDepositAmount acepta un importe no negativo con hasta dos decimales.
func ValidDepositAmount(value string) bool {
	return depositPattern.MatchString(value)
}

// ExpiredReservation es una reserva que el barrido pasó a «No asistió» porque
// venció su tolerancia. LocalTime es la hora reservada en la zona del local.
type ExpiredReservation struct {
	ID, OrganizationID, LocationID string
	CustomerName, TableName        string
	LocalTime                      string
	Guests                         int
	// Deposit: garantía cobrada que queda retenida, ya con su moneda; vacía sin ella.
	Deposit string
}

// Deposit decisions when a reservation with a charged guarantee is cancelled.
const (
	DepositRefund = "refund"
	DepositRetain = "retain"
)

// MaxToleranceMinutes acota la espera: más de 4 horas ya no es una tolerancia.
const MaxToleranceMinutes = 240

// ValidToleranceMinutes valida la tolerancia de una reserva o de un local.
func ValidToleranceMinutes(value int) bool {
	return value >= 0 && value <= MaxToleranceMinutes
}

// ReservationAttended («Asistió») no se guarda: el listado lo proyecta para una
// reserva en mesa (seated) cuyo tiempo reservado ya terminó. Sirve como filtro,
// nunca como destino de un cambio de estado.
const ReservationAttended = "attended"

func ValidReservationStatus(status string) bool {
	return status == "pending" || status == "confirmed" || status == "seated" || status == "cancelled" || status == "no_show"
}
func ValidReservationTransition(current, next string) bool {
	if current == next {
		return true
	}
	switch current {
	case "pending":
		return next == "confirmed" || next == "seated" || next == "cancelled" || next == "no_show"
	case "confirmed":
		return next == "seated" || next == "cancelled" || next == "no_show"
	default:
		return false
	}
}
func NormalizeOptionalID(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
func ParseReservationInput(in *ReservationInput) (time.Time, bool) {
	in.CustomerID = NormalizeOptionalID(in.CustomerID)
	in.TableID = NormalizeOptionalID(in.TableID)
	in.CustomerName = strings.TrimSpace(in.CustomerName)
	in.CustomerPhone = strings.TrimSpace(in.CustomerPhone)
	in.Notes = strings.TrimSpace(in.Notes)
	if in.DepositAmount != nil {
		trimmed := strings.TrimSpace(*in.DepositAmount)
		if trimmed == "" {
			trimmed = "0"
		}
		in.DepositAmount = &trimmed
	}
	if in.DurationMinutes == 0 {
		in.DurationMinutes = 90
	}
	startsAt, err := time.Parse(time.RFC3339, in.StartsAt)
	if err != nil || startsAt.Before(time.Now().Add(-5*time.Minute)) || in.CustomerName == "" || len(in.CustomerName) > 180 || len(in.CustomerPhone) > 40 || in.Guests < 1 || in.Guests > 100 || in.DurationMinutes < 15 || in.DurationMinutes > 360 || len(in.Notes) > 500 || (in.ToleranceMinutes != nil && !ValidToleranceMinutes(*in.ToleranceMinutes)) || (in.DepositAmount != nil && !ValidDepositAmount(*in.DepositAmount)) {
		return time.Time{}, false
	}
	return startsAt, true
}
