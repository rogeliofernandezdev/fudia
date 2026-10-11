package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/billing/application"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

type Cash struct{ service *application.Cash }

func NewCash(service *application.Cash) *Cash { return &Cash{service} }
func (h *Cash) CreateCashRegister(w http.ResponseWriter, r *http.Request) {
	var in domain.RegisterInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_cash_register", "Revisa los datos de la caja.")
		return
	}
	out, err := h.service.CreateRegister(r.Context(), billingScope(r), in)
	billingResponse(w, 201, out, err)
}
func (h *Cash) UpdateCashRegister(w http.ResponseWriter, r *http.Request) {
	var in domain.RegisterInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_cash_register", "Revisa los datos de la caja.")
		return
	}
	out, err := h.service.UpdateRegister(r.Context(), billingScope(r), r.PathValue("id"), in)
	billingResponse(w, 200, out, err)
}
func (h *Cash) UpdateCashRegisterStatus(w http.ResponseWriter, r *http.Request) {
	var in domain.RegisterStatusInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_cash_register", "Revisa el estado de la caja.")
		return
	}
	err := h.service.SetRegisterActive(r.Context(), billingScope(r), r.PathValue("id"), in)
	billingResponse(w, 204, nil, err)
}
func (h *Cash) OpenCashShift(w http.ResponseWriter, r *http.Request) {
	var in domain.OpenShiftInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_cash_shift", "Selecciona una caja e ingresa un fondo inicial válido.")
		return
	}
	out, err := h.service.Open(r.Context(), billingScope(r), in)
	billingResponse(w, 201, out, err)
}
func (h *Cash) CreateCashMovement(w http.ResponseWriter, r *http.Request) {
	var in domain.CashMovementInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_cash_movement", "Revisa el tipo y monto del movimiento.")
		return
	}
	out, err := h.service.Movement(r.Context(), billingScope(r), r.PathValue("id"), in)
	billingResponse(w, 201, out, err)
}
func (h *Cash) CloseCashShift(w http.ResponseWriter, r *http.Request) {
	var in domain.CloseShiftInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_cash_close", "Revisa los datos del arqueo.")
		return
	}
	out, err := h.service.Close(r.Context(), billingScope(r), r.PathValue("id"), in)
	billingResponse(w, 200, out, err)
}
func (h *Cash) AssignCashShiftUser(w http.ResponseWriter, r *http.Request) {
	var in domain.ShiftUserInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_cash_user", "Selecciona un usuario válido.")
		return
	}
	out, err := h.service.Assign(r.Context(), billingScope(r), r.PathValue("id"), in)
	billingResponse(w, 201, out, err)
}
func (h *Cash) CreateCashOperation(w http.ResponseWriter, r *http.Request) {
	var in domain.CashOperationInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_cash_operation", "Ingresa un monto válido.")
		return
	}
	out, err := h.service.Operation(r.Context(), billingScope(r), r.PathValue("id"), in)
	billingResponse(w, 201, out, err)
}
func (h *Cash) ListCashRegisters(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Registers(r.Context(), billingScope(r), r.URL.Query().Get("q"))
	billingResponse(w, 200, map[string]any{"items": out}, err)
}
func (h *Cash) GetCurrentCashShift(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Current(r.Context(), billingScope(r))
	billingResponse(w, 200, map[string]any{"shift": out}, err)
}
func (h *Cash) ListCashShifts(w http.ResponseWriter, r *http.Request) {
	page, size := shared.PageParams(r)
	q := r.URL.Query()
	out, err := h.service.List(r.Context(), billingScope(r), application.CashFilter{Query: q.Get("q"), Status: q.Get("status"), Page: page, Size: size})
	billingResponse(w, 200, out, err)
}
func (h *Cash) GetCashShift(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Get(r.Context(), billingScope(r), r.PathValue("id"))
	billingResponse(w, 200, out, err)
}
func (h *Cash) GetCashShiftReport(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Report(r.Context(), billingScope(r), r.PathValue("id"))
	if err == nil {
		w.Header().Set("Cache-Control", "no-store")
	}
	billingResponse(w, 200, out, err)
}
func (h *Cash) ListCashUserOptions(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.UserOptions(r.Context(), billingScope(r))
	billingResponse(w, 200, map[string]any{"items": out}, err)
}
func (h *Cash) ListCashShiftUsers(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Users(r.Context(), billingScope(r), r.PathValue("id"))
	billingResponse(w, 200, map[string]any{"items": out}, err)
}
func (h *Cash) UnassignCashShiftUser(w http.ResponseWriter, r *http.Request) {
	err := h.service.Unassign(r.Context(), billingScope(r), r.PathValue("id"), r.PathValue("userId"))
	billingResponse(w, 204, nil, err)
}
