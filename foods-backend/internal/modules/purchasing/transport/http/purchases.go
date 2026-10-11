package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/purchasing/application"
	"github.com/foods-platform/foods-backend/internal/modules/purchasing/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"net/http"
)

type Handler struct{ service *application.Purchasing }

func New(service *application.Purchasing) *Handler { return &Handler{service} }
func scopeFor(r *http.Request) requestscope.Scope  { s, _ := requestscope.From(r.Context()); return s }
func respond(w http.ResponseWriter, status int, out any, err error) {
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	if status == 204 {
		w.WriteHeader(status)
		return
	}
	shared.WriteJSON(w, status, out)
}
func filterFor(r *http.Request) application.Filter {
	page, size := shared.PageParams(r)
	q := r.URL.Query()
	return application.Filter{Page: page, Size: size, Query: q.Get("q"), Status: q.Get("status"), From: q.Get("from"), To: q.Get("to")}
}
func (h *Handler) ListSuppliers(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Suppliers(r.Context(), scopeFor(r), filterFor(r))
	respond(w, 200, out, err)
}
func (h *Handler) ListPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Orders(r.Context(), scopeFor(r), filterFor(r))
	respond(w, 200, out, err)
}
func (h *Handler) ListPurchaseReceipts(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Receipts(r.Context(), scopeFor(r), filterFor(r))
	respond(w, 200, out, err)
}
func (h *Handler) GetPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Order(r.Context(), scopeFor(r), r.PathValue("id"))
	respond(w, 200, out, err)
}
func (h *Handler) GetPurchaseReceipt(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Receipt(r.Context(), scopeFor(r), r.PathValue("id"))
	respond(w, 200, out, err)
}
func (h *Handler) CreateSupplier(w http.ResponseWriter, r *http.Request) {
	var in domain.SupplierInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_supplier", "Revisa los datos del proveedor.")
		return
	}
	out, err := h.service.SaveSupplier(r.Context(), scopeFor(r), r.PathValue("id"), in, false)
	respond(w, 201, out, err)
}
func (h *Handler) UpdateSupplier(w http.ResponseWriter, r *http.Request) {
	var in domain.SupplierInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_supplier", "Revisa los datos del proveedor.")
		return
	}
	out, err := h.service.SaveSupplier(r.Context(), scopeFor(r), r.PathValue("id"), in, true)
	respond(w, 200, out, err)
}
func (h *Handler) CreatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	var in domain.PurchaseOrderInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_purchase", "Revisa los datos de la orden.")
		return
	}
	out, err := h.service.SaveOrder(r.Context(), scopeFor(r), r.PathValue("id"), in, false)
	respond(w, 201, out, err)
}
func (h *Handler) UpdatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	var in domain.PurchaseOrderInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_purchase", "Revisa los datos de la orden.")
		return
	}
	out, err := h.service.SaveOrder(r.Context(), scopeFor(r), r.PathValue("id"), in, true)
	respond(w, 200, out, err)
}
func (h *Handler) ReceivePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	var in domain.PurchaseReceiptInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_purchase_receipt", "Revisa las cantidades recibidas.")
		return
	}
	out, err := h.service.Receive(r.Context(), scopeFor(r), r.PathValue("id"), in)
	status := 201
	if out["idempotent"] == true {
		status = 200
	}
	respond(w, status, out, err)
}
func (h *Handler) CreatePurchaseReturn(w http.ResponseWriter, r *http.Request) {
	var in domain.PurchaseReturnInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_purchase_return", "Revisa la devolución.")
		return
	}
	out, err := h.service.Return(r.Context(), scopeFor(r), r.PathValue("id"), in)
	status := 201
	if out["idempotent"] == true {
		status = 200
	}
	respond(w, status, out, err)
}
func (h *Handler) UpdateSupplierStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Active bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_status", "El estado no es válido.")
		return
	}
	err := h.service.SetSupplierActive(r.Context(), scopeFor(r), r.PathValue("id"), in.Active)
	respond(w, 204, nil, err)
}
func (h *Handler) UpdatePurchaseOrderStatus(w http.ResponseWriter, r *http.Request) {
	var in application.OrderStatusInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_status", "El estado solicitado no es válido.")
		return
	}
	err := h.service.SetOrderStatus(r.Context(), scopeFor(r), r.PathValue("id"), in)
	respond(w, 204, nil, err)
}
func (h *Handler) ApprovePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	err := h.service.Approve(r.Context(), scopeFor(r), r.PathValue("id"))
	respond(w, 204, nil, err)
}
