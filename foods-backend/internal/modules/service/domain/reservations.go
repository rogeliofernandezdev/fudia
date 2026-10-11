package domain

import (
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
	TableID         *string `json:"tableId,omitempty"`
	TableName       string  `json:"tableName"`
	Status          string  `json:"status"`
	Notes           string  `json:"notes"`
	CreatedAt       string  `json:"createdAt"`
}
type ReservationInput struct {
	CustomerID      *string `json:"customerId,omitempty"`
	CustomerName    string  `json:"customerName"`
	CustomerPhone   string  `json:"customerPhone"`
	StartsAt        string  `json:"startsAt"`
	Guests          int     `json:"guests"`
	DurationMinutes int     `json:"durationMinutes"`
	TableID         *string `json:"tableId,omitempty"`
	Notes           string  `json:"notes"`
}

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
	if in.DurationMinutes == 0 {
		in.DurationMinutes = 90
	}
	startsAt, err := time.Parse(time.RFC3339, in.StartsAt)
	if err != nil || startsAt.Before(time.Now().Add(-5*time.Minute)) || in.CustomerName == "" || len(in.CustomerName) > 180 || len(in.CustomerPhone) > 40 || in.Guests < 1 || in.Guests > 100 || in.DurationMinutes < 15 || in.DurationMinutes > 360 || len(in.Notes) > 500 {
		return time.Time{}, false
	}
	return startsAt, true
}
