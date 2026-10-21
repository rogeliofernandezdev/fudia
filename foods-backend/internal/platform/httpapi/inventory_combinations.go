package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

type inventoryCombinationView struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Unit      string `json:"unit"`
	UnitName  string `json:"unitName"`
	IsDefault bool   `json:"isDefault"`
}
type inventoryCombinationInput struct {
	Unit      string `json:"unit"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault"`
}
type inventoryPresentationInput struct {
	PresentationType     string  `json:"presentationType"`
	UnitsPerPresentation float64 `json:"unitsPerPresentation"`
	IsDefault            bool    `json:"isDefault"`
}

func (a *API) listInventoryCombinations(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	unit := strings.TrimSpace(r.URL.Query().Get("unit"))
	rows, err := a.db.Query(r.Context(), `
 SELECT t.code,t.name,u.code,u.name,
  CASE WHEN t.code='unit' THEN NOT EXISTS(SELECT 1 FROM inventory_unit_combinations d WHERE d.organization_id=u.organization_id AND d.unit=u.code AND d.is_default AND d.presentation_type<>'unit') ELSE c.is_default END
 FROM inventory_units u JOIN inventory_presentation_types t ON t.organization_id=u.organization_id
 LEFT JOIN inventory_unit_combinations c ON c.organization_id=u.organization_id AND c.unit=u.code AND c.presentation_type=t.code
 WHERE u.organization_id=$1 AND ($2='' OR u.code=$2) AND (t.code='unit' OR c.id IS NOT NULL)
 ORDER BY u.sort_order,u.name,t.sort_order,t.name`, s.OrganizationID, unit)
	if err != nil {
		fail(w, 503, "inventory_combinations_unavailable", "No pudimos cargar las combinaciones.")
		return
	}
	defer rows.Close()
	items := []inventoryCombinationView{}
	for rows.Next() {
		var x inventoryCombinationView
		if err = rows.Scan(&x.Code, &x.Name, &x.Unit, &x.UnitName, &x.IsDefault); err != nil {
			break
		}
		items = append(items, x)
	}
	if err != nil || rows.Err() != nil {
		fail(w, 503, "inventory_combinations_unavailable", "No pudimos cargar las combinaciones.")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (a *API) createInventoryCombination(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in inventoryCombinationInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_inventory_combination", "Revisa los datos de la combinación.")
		return
	}
	normalized, invalid := normalizeInventoryUnit(inventoryUnitInput{Code: in.Code, Name: in.Name})
	if invalid != "" {
		fail(w, 400, "invalid_inventory_combination", invalid)
		return
	}
	in.Unit = strings.TrimSpace(in.Unit)
	in.Code = normalized.Code
	in.Name = normalized.Name
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "inventory_combinations_unavailable", "No pudimos guardar la combinación.")
		return
	}
	defer tx.Rollback(r.Context())
	// Serialize catalog/default changes on the base-unit row.
	var unitName string
	err = tx.QueryRow(r.Context(), `SELECT name FROM inventory_units WHERE organization_id=$1 AND code=$2 FOR UPDATE`, s.OrganizationID, in.Unit).Scan(&unitName)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 400, "invalid_inventory_unit", "Selecciona una unidad base de tu empresa.")
		return
	}
	if err != nil {
		fail(w, 503, "inventory_combinations_unavailable", "No pudimos validar la unidad.")
		return
	}
	if _, err = tx.Exec(r.Context(), `INSERT INTO inventory_presentation_types(organization_id,code,name) VALUES($1,$2,$3) ON CONFLICT(organization_id,code) DO NOTHING`, s.OrganizationID, in.Code, in.Name); err != nil {
		fail(w, 503, "inventory_combinations_unavailable", "No pudimos guardar la presentación.")
		return
	}
	var name string
	if err = tx.QueryRow(r.Context(), `SELECT name FROM inventory_presentation_types WHERE organization_id=$1 AND code=$2`, s.OrganizationID, in.Code).Scan(&name); err != nil {
		fail(w, 503, "inventory_combinations_unavailable", "No pudimos validar la presentación.")
		return
	}
	if name != in.Name {
		fail(w, 409, "inventory_presentation_conflict", "Ese código ya corresponde a otra presentación.")
		return
	}
	if in.IsDefault {
		if _, err = tx.Exec(r.Context(), `UPDATE inventory_unit_combinations SET is_default=false WHERE organization_id=$1 AND unit=$2 AND is_default`, s.OrganizationID, in.Unit); err != nil {
			fail(w, 503, "inventory_combinations_unavailable", "No pudimos guardar la preferencia.")
			return
		}
	}
	var id string
	err = tx.QueryRow(r.Context(), `INSERT INTO inventory_unit_combinations(organization_id,unit,presentation_type,is_default) VALUES($1,$2,$3,$4) ON CONFLICT(organization_id,unit,presentation_type) DO UPDATE SET is_default=EXCLUDED.is_default RETURNING id`, s.OrganizationID, in.Unit, in.Code, in.IsDefault).Scan(&id)
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata) VALUES($1,$2,$3,'inventory_combination.saved','inventory_combination',$4,jsonb_build_object('unit',$5::text,'type',$6::text,'default',$7::boolean))`, s.OrganizationID, s.LocationID, s.UserID, id, in.Unit, in.Code, in.IsDefault)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		fail(w, 503, "inventory_combinations_unavailable", "No pudimos guardar la combinación.")
		return
	}
	writeJSON(w, 201, inventoryCombinationView{Code: in.Code, Name: name, Unit: in.Unit, UnitName: unitName, IsDefault: in.IsDefault})
}

func normalizeInventoryPresentation(in inventoryPresentationInput) (inventoryPresentationInput, string) {
	in.PresentationType = strings.ToLower(strings.TrimSpace(in.PresentationType))
	if !inventoryUnitCode.MatchString(in.PresentationType) {
		return in, "Selecciona una presentación válida."
	}
	if math.IsNaN(in.UnitsPerPresentation) || math.IsInf(in.UnitsPerPresentation, 0) || in.UnitsPerPresentation <= 0 || in.UnitsPerPresentation >= 1e11 {
		return in, "La conversión debe ser mayor que cero y menor que 100000000000."
	}
	if math.Abs(in.UnitsPerPresentation*1000-math.Round(in.UnitsPerPresentation*1000)) > 0.000001 {
		return in, "Usa como máximo tres decimales en la conversión."
	}
	if in.PresentationType == "unit" && in.UnitsPerPresentation != 1 {
		return in, "La unidad base equivale a una unidad."
	}
	return in, ""
}

func validateInventoryCombination(ctx context.Context, tx pgx.Tx, org, item, typeCode string) error {
	var allowed bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(
 SELECT 1 FROM inventory_items i JOIN inventory_presentation_types t ON t.organization_id=i.organization_id AND t.code=$3
 WHERE i.organization_id=$1 AND i.id=$2 AND (t.code='unit' OR EXISTS(SELECT 1 FROM inventory_unit_combinations c WHERE c.organization_id=i.organization_id AND c.unit=i.unit AND c.presentation_type=t.code)))`, org, item, typeCode).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return &inventoryCombinationError{}
	}
	return nil
}

type inventoryCombinationError struct{}

func (*inventoryCombinationError) Error() string {
	return "La presentación no está configurada para la unidad base de este artículo."
}

func setDefaultInventoryPresentation(ctx context.Context, tx pgx.Tx, org, item, id string) error {
	if _, err := tx.Exec(ctx, `SELECT id FROM inventory_items WHERE organization_id=$1 AND id=$2 FOR UPDATE`, org, item); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE inventory_presentations SET is_default=false WHERE organization_id=$1 AND inventory_item_id=$2 AND is_default`, org, item); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE inventory_presentations SET is_default=true,updated_at=now() WHERE organization_id=$1 AND inventory_item_id=$2 AND id=$3 AND active`, org, item, id)
	if err == nil && tag.RowsAffected() != 1 {
		return fmt.Errorf("presentation not found")
	}
	return err
}

func (a *API) savePurchasePresentation(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in inventoryPresentationInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_inventory_presentation", "Revisa la presentación.")
		return
	}
	in, invalid := normalizeInventoryPresentation(in)
	if invalid != "" {
		fail(w, 400, "invalid_inventory_presentation", invalid)
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "inventory_presentations_unavailable", "No pudimos guardar la presentación.")
		return
	}
	defer tx.Rollback(r.Context())
	var unit string
	err = tx.QueryRow(r.Context(), `SELECT unit FROM inventory_items WHERE organization_id=$1 AND id=$2 AND active FOR UPDATE`, s.OrganizationID, r.PathValue("id")).Scan(&unit)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "inventory_item_not_found", "El artículo no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "inventory_presentations_unavailable", "No pudimos validar el artículo.")
		return
	}
	id, err := ensureInventoryPresentation(r.Context(), tx, s.OrganizationID, r.PathValue("id"), in.PresentationType, in.UnitsPerPresentation)
	var invalidCombo *inventoryCombinationError
	if errors.As(err, &invalidCombo) {
		fail(w, 400, "invalid_inventory_combination", invalidCombo.Error())
		return
	}
	if err == nil && in.IsDefault {
		err = setDefaultInventoryPresentation(r.Context(), tx, s.OrganizationID, r.PathValue("id"), id)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata) VALUES($1,$2,$3,'inventory_presentation.saved','inventory_presentation',$4,jsonb_build_object('itemId',$5::text,'factor',$6::numeric,'default',$7::boolean))`, s.OrganizationID, s.LocationID, s.UserID, id, r.PathValue("id"), in.UnitsPerPresentation, in.IsDefault)
	}
	var out inventoryPresentationOption
	if err == nil {
		err = tx.QueryRow(r.Context(), `SELECT p.id,p.presentation_type,p.units_per_presentation::text,t.name,p.is_default FROM inventory_presentations p JOIN inventory_presentation_types t ON t.organization_id=p.organization_id AND t.code=p.presentation_type WHERE p.organization_id=$1 AND p.id=$2`, s.OrganizationID, id).Scan(&out.ID, &out.PresentationType, &out.UnitsPerPresentation, &out.Name, &out.IsDefault)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		fail(w, 503, "inventory_presentations_unavailable", "No pudimos guardar la presentación.")
		return
	}
	writeJSON(w, 201, out)
}
