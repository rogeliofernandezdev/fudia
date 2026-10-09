package httpapi

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type availabilityHistoryItem struct {
	ID                      string  `json:"id"`
	CreatedAt               string  `json:"createdAt"`
	BusinessDate            *string `json:"businessDate"`
	UserID                  *string `json:"userId"`
	UserName                string  `json:"userName"`
	Reason                  *string `json:"reason"`
	PreviousPortionQuantity *int    `json:"previousPortionQuantity"`
	PortionQuantity         *int    `json:"portionQuantity"`
	SoldQuantity            *int    `json:"soldQuantity"`
	PreviousManualStatus    *string `json:"previousManualStatus"`
	ManualStatus            *string `json:"manualStatus"`
}

func (a *API) listProductAvailabilityHistory(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	tx, err := a.db.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		fail(w, 503, "availability_history_unavailable", "No pudimos consultar el historial.")
		return
	}
	defer tx.Rollback(r.Context())
	productID := r.PathValue("productId")
	var exists bool
	if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND organization_id=$2)`, productID, s.OrganizationID).Scan(&exists); err != nil {
		fail(w, 503, "availability_history_unavailable", "No pudimos validar el producto.")
		return
	}
	if !exists {
		fail(w, 404, "product_not_found", "El producto no existe.")
		return
	}
	const filter = ` FROM audit_log h WHERE h.organization_id=$1 AND h.location_id=$2 AND h.entity_id=$3 AND h.entity_type='product' AND h.action='product.availability_updated'`
	var total int
	if err = tx.QueryRow(r.Context(), `SELECT count(*)`+filter, s.OrganizationID, s.LocationID, productID).Scan(&total); err != nil {
		fail(w, 503, "availability_history_unavailable", "No pudimos consultar el historial.")
		return
	}
	rows, err := tx.Query(r.Context(), `SELECT h.id,h.created_at,h.metadata->>'businessDate',h.user_id,
	 COALESCE(h.metadata->>'actorName',(SELECT full_name FROM users WHERE id=h.user_id),'Usuario no disponible'),h.reason,
	 (h.metadata->>'previousPortionQuantity')::integer,(h.metadata->>'portionQuantity')::integer,
	 (h.metadata->>'soldQuantity')::integer,h.metadata->>'previousManualStatus',h.metadata->>'manualStatus'`+
		filter+` ORDER BY h.created_at DESC,h.id DESC LIMIT $4 OFFSET $5`, s.OrganizationID, s.LocationID, productID, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "availability_history_unavailable", "No pudimos cargar los cambios.")
		return
	}
	defer rows.Close()
	items := []availabilityHistoryItem{}
	for rows.Next() {
		var item availabilityHistoryItem
		var createdAt time.Time
		if err = rows.Scan(&item.ID, &createdAt, &item.BusinessDate, &item.UserID, &item.UserName, &item.Reason,
			&item.PreviousPortionQuantity, &item.PortionQuantity, &item.SoldQuantity, &item.PreviousManualStatus, &item.ManualStatus); err != nil {
			fail(w, 503, "availability_history_unavailable", "No pudimos leer los cambios.")
			return
		}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		fail(w, 503, "availability_history_unavailable", "No pudimos leer los cambios.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "availability_history_unavailable", "No pudimos confirmar la lectura del historial.")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}
