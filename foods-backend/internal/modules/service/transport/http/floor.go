package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/service/application"
	"github.com/foods-platform/foods-backend/internal/modules/service/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"net/http"
	"strconv"
)

type Floor struct{ service *application.Floor }

func NewFloor(service *application.Floor) *Floor    { return &Floor{service} }
func floorScope(r *http.Request) requestscope.Scope { s, _ := requestscope.From(r.Context()); return s }
func floorResponse(w http.ResponseWriter, status int, out any, err error) {
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
func floorFilter(r *http.Request, defaultSize int) application.FloorFilter {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if size < 1 {
		size = defaultSize
	}
	return application.FloorFilter{Query: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status"), Page: page, Size: size}
}
func (h *Floor) ListZones(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Zones(r.Context(), floorScope(r), floorFilter(r, 50))
	floorResponse(w, 200, out, err)
}
func (h *Floor) ListTables(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Tables(r.Context(), floorScope(r), floorFilter(r, 10))
	floorResponse(w, 200, out, err)
}
func (h *Floor) GetTableByQR(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.TableQR(r.Context(), r.PathValue("token"))
	floorResponse(w, 200, out, err)
}
func (h *Floor) RegenerateTableQR(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.RegenerateTableQR(r.Context(), floorScope(r), r.PathValue("id"))
	floorResponse(w, 200, out, err)
}
func (h *Floor) CreateTablesBatch(w http.ResponseWriter, r *http.Request) {
	var in domain.TableBatchInput
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.Batch(r.Context(), floorScope(r), in.Items)
	floorResponse(w, 201, out, err)
}
func (h *Floor) CreateZone(w http.ResponseWriter, r *http.Request) {
	var in domain.ZoneInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "El nombre de la zona es obligatorio.")
		return
	}
	out, err := h.service.CreateZone(r.Context(), floorScope(r), in)
	floorResponse(w, 201, out, err)
}
func (h *Floor) UpdateZone(w http.ResponseWriter, r *http.Request) {
	var in domain.ZoneInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "El nombre de la zona es obligatorio.")
		return
	}
	out, err := h.service.UpdateZone(r.Context(), floorScope(r), r.PathValue("id"), in)
	floorResponse(w, 200, out, err)
}
func (h *Floor) DeactivateZone(w http.ResponseWriter, r *http.Request) {
	err := h.service.DeactivateZone(r.Context(), floorScope(r), r.PathValue("id"))
	floorResponse(w, 204, nil, err)
}
func (h *Floor) CreateTable(w http.ResponseWriter, r *http.Request) {
	var in domain.TableInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.CreateTable(r.Context(), floorScope(r), in)
	floorResponse(w, 201, out, err)
}
func (h *Floor) UpdateTable(w http.ResponseWriter, r *http.Request) {
	var in domain.TableInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.UpdateTable(r.Context(), floorScope(r), r.PathValue("id"), in)
	floorResponse(w, 200, out, err)
}
func (h *Floor) DeactivateTable(w http.ResponseWriter, r *http.Request) {
	err := h.service.DeactivateTable(r.Context(), floorScope(r), r.PathValue("id"))
	floorResponse(w, 204, nil, err)
}
