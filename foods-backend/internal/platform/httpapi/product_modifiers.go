package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type modifierOptionInput struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Surcharge string `json:"surcharge"`
}

type modifierGroupInput struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	Required      bool                  `json:"required"`
	MinSelections int                   `json:"minSelections"`
	MaxSelections int                   `json:"maxSelections"`
	Options       []modifierOptionInput `json:"options"`
}

type modifierConfigInput struct {
	Groups []modifierGroupInput `json:"groups"`
}

type modifierOptionView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Surcharge string `json:"surcharge"`
}

type modifierGroupView struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	Required      bool                 `json:"required"`
	MinSelections int                  `json:"minSelections"`
	MaxSelections int                  `json:"maxSelections"`
	Options       []modifierOptionView `json:"options"`
}

type modifierConfigView struct {
	ProductID string              `json:"productId"`
	Groups    []modifierGroupView `json:"groups"`
}

type modifierQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadProductModifiers(
	ctx context.Context,
	q modifierQuerier,
	organizationID string,
	productID string,
) (modifierConfigView, error) {
	out := modifierConfigView{ProductID: productID, Groups: []modifierGroupView{}}
	rows, err := q.Query(ctx, `
		SELECT g.id::text,g.name,g.required,g.min_selections,g.max_selections,
		       o.id::text,o.name,o.surcharge::text
		FROM product_modifier_groups g
		LEFT JOIN product_modifier_options o
		  ON o.group_id=g.id AND o.organization_id=g.organization_id
		WHERE g.organization_id=$1 AND g.product_id=$2
		ORDER BY g.sort_order,g.id,o.sort_order,o.id
	`, organizationID, productID)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	index := map[string]int{}
	for rows.Next() {
		var groupID, groupName string
		var required bool
		var minSelections, maxSelections int
		var optionID, optionName, surcharge *string
		if err := rows.Scan(
			&groupID, &groupName, &required, &minSelections, &maxSelections,
			&optionID, &optionName, &surcharge,
		); err != nil {
			return out, err
		}
		position, exists := index[groupID]
		if !exists {
			position = len(out.Groups)
			index[groupID] = position
			out.Groups = append(out.Groups, modifierGroupView{
				ID: groupID, Name: groupName, Required: required,
				MinSelections: minSelections, MaxSelections: maxSelections,
				Options: []modifierOptionView{},
			})
		}
		if optionID != nil && optionName != nil && surcharge != nil {
			out.Groups[position].Options = append(out.Groups[position].Options, modifierOptionView{
				ID: *optionID, Name: *optionName, Surcharge: *surcharge,
			})
		}
	}
	return out, rows.Err()
}

func validateModifierConfig(in modifierConfigInput) string {
	groupNames := map[string]bool{}
	for _, group := range in.Groups {
		name := strings.TrimSpace(group.Name)
		if name == "" || len([]rune(name)) > 80 {
			return "Cada grupo necesita un nombre de hasta 80 caracteres."
		}
		key := strings.ToLower(name)
		if groupNames[key] {
			return "Los grupos de modificadores necesitan nombres distintos."
		}
		groupNames[key] = true
		if group.MinSelections < 0 || group.MaxSelections < 1 ||
			group.MinSelections > group.MaxSelections {
			return "Revisa los mínimos y máximos de cada grupo."
		}
		if group.Required && group.MinSelections < 1 {
			return "Un grupo obligatorio debe exigir al menos una selección."
		}
		if group.MaxSelections > len(group.Options) {
			return "El máximo de selecciones no puede superar las opciones disponibles."
		}

		optionNames := map[string]bool{}
		for _, option := range group.Options {
			optionName := strings.TrimSpace(option.Name)
			if optionName == "" || len([]rune(optionName)) > 80 {
				return "Cada opción necesita un nombre de hasta 80 caracteres."
			}
			optionKey := strings.ToLower(optionName)
			if optionNames[optionKey] {
				return "Las opciones de un mismo grupo necesitan nombres distintos."
			}
			optionNames[optionKey] = true
			value := strings.TrimSpace(option.Surcharge)
			if value == "" {
				value = "0"
			}
			surcharge, err := strconv.ParseFloat(value, 64)
			if err != nil || surcharge < 0 {
				return "Cada recargo debe ser un importe válido mayor o igual a cero."
			}
		}
	}
	return ""
}

func (a *API) productSupportsModifiers(
	ctx context.Context,
	q interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	organizationID string,
	productID string,
) (bool, error) {
	var exists, combo bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS(
		         SELECT 1 FROM products
		         WHERE id=$1 AND organization_id=$2
		       ),
		       EXISTS(
		         SELECT 1 FROM menu_combos
		         WHERE product_id=$1 AND organization_id=$2
		       )
	`, productID, organizationID).Scan(&exists, &combo)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, pgx.ErrNoRows
	}
	return !combo, nil
}

func (a *API) getProductModifiers(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	productID := r.PathValue("id")
	supported, err := a.productSupportsModifiers(r.Context(), a.db, s.OrganizationID, productID)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "product_not_found", "El producto no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "modifiers_unavailable", "No pudimos validar el producto.")
		return
	}
	if !supported {
		fail(w, 409, "modifiers_not_supported_for_combo", "Los combos administran sus opciones desde Menús y combos.")
		return
	}
	out, err := loadProductModifiers(r.Context(), a.db, s.OrganizationID, productID)
	if err != nil {
		fail(w, 503, "modifiers_unavailable", "No pudimos cargar los modificadores.")
		return
	}
	writeJSON(w, 200, out)
}

func (a *API) saveProductModifiers(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	productID := r.PathValue("id")
	var in modifierConfigInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_modifiers", "Revisa la configuración de modificadores.")
		return
	}
	if message := validateModifierConfig(in); message != "" {
		fail(w, 400, "invalid_modifiers", message)
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "modifiers_unavailable", "No pudimos iniciar la actualización.")
		return
	}
	defer tx.Rollback(r.Context())

	supported, err := a.productSupportsModifiers(r.Context(), tx, s.OrganizationID, productID)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "product_not_found", "El producto no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "modifiers_unavailable", "No pudimos validar el producto.")
		return
	}
	if !supported {
		fail(w, 409, "modifiers_not_supported_for_combo", "Los combos administran sus opciones desde Menús y combos.")
		return
	}

	// Serialize configuration writes for this product. Under READ COMMITTED, a
	// waiter sees the previous save before it computes the differential update.
	var lockedProductID string
	if err = tx.QueryRow(r.Context(), `
		SELECT id::text FROM products
		WHERE id=$1 AND organization_id=$2
		FOR UPDATE
	`, productID, s.OrganizationID).Scan(&lockedProductID); err != nil {
		fail(w, 503, "modifiers_unavailable", "No pudimos bloquear el producto para actualizar sus modificadores.")
		return
	}

	current, err := loadProductModifiers(r.Context(), tx, s.OrganizationID, productID)
	if err != nil {
		fail(w, 503, "modifiers_unavailable", "No pudimos cargar la configuración actual.")
		return
	}
	currentGroups := make(map[string]modifierGroupView, len(current.Groups))
	groupsByName := make(map[string]string, len(current.Groups))
	for _, group := range current.Groups {
		currentGroups[group.ID] = group
		groupsByName[strings.ToLower(strings.TrimSpace(group.Name))] = group.ID
	}

	groupIDs := make([]string, len(in.Groups))
	optionIDs := make([][]string, len(in.Groups))
	keptGroups := map[string]bool{}
	for groupIndex, group := range in.Groups {
		groupID := strings.TrimSpace(group.ID)
		if groupID == "" {
			// Backward-compatible matching for clients that do not yet echo IDs.
			groupID = groupsByName[strings.ToLower(strings.TrimSpace(group.Name))]
		}
		if groupID != "" {
			if _, exists := currentGroups[groupID]; !exists || keptGroups[groupID] {
				fail(w, 400, "invalid_modifiers", "Un grupo no pertenece a este producto o está repetido.")
				return
			}
			keptGroups[groupID] = true
		}
		groupIDs[groupIndex] = groupID
		optionIDs[groupIndex] = make([]string, len(group.Options))
	}

	// Remove groups omitted from the submitted configuration before renames so
	// their unique names cannot block a retained group's attribute update.
	for groupID := range currentGroups {
		if keptGroups[groupID] {
			continue
		}
		if _, err = tx.Exec(r.Context(), `
			DELETE FROM product_modifier_groups
			WHERE id=$1 AND organization_id=$2 AND product_id=$3
		`, groupID, s.OrganizationID, productID); err != nil {
			fail(w, 503, "modifiers_unavailable", "No pudimos eliminar los grupos retirados.")
			return
		}
	}

	for groupIndex, group := range in.Groups {
		groupID := groupIDs[groupIndex]
		if groupID == "" {
			err = tx.QueryRow(r.Context(), `
				INSERT INTO product_modifier_groups(
				  organization_id,product_id,name,required,min_selections,max_selections,sort_order
				) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id::text
			`, s.OrganizationID, productID, strings.TrimSpace(group.Name),
				group.Required, group.MinSelections, group.MaxSelections, groupIndex).Scan(&groupID)
		} else {
			_, err = tx.Exec(r.Context(), `
				UPDATE product_modifier_groups
				SET name=$3,required=$4,min_selections=$5,max_selections=$6,sort_order=$7,updated_at=now()
				WHERE id=$1 AND organization_id=$2 AND product_id=$8
			`, groupID, s.OrganizationID, strings.TrimSpace(group.Name),
				group.Required, group.MinSelections, group.MaxSelections, groupIndex, productID)
		}
		if err != nil {
			fail(w, 409, "modifier_group_conflict", "No pudimos guardar un grupo de modificadores.")
			return
		}

		currentGroup, existed := currentGroups[groupID]
		optionsByID := map[string]modifierOptionView{}
		optionsByName := map[string]string{}
		if existed {
			for _, option := range currentGroup.Options {
				optionsByID[option.ID] = option
				optionsByName[strings.ToLower(strings.TrimSpace(option.Name))] = option.ID
			}
		}
		keptOptions := map[string]bool{}
		for optionIndex, option := range group.Options {
			optionID := strings.TrimSpace(option.ID)
			if optionID == "" {
				optionID = optionsByName[strings.ToLower(strings.TrimSpace(option.Name))]
			}
			if optionID != "" {
				if _, exists := optionsByID[optionID]; !exists || keptOptions[optionID] {
					fail(w, 400, "invalid_modifiers", "Una opción no pertenece al grupo o está repetida.")
					return
				}
				keptOptions[optionID] = true
			}
			optionIDs[groupIndex][optionIndex] = optionID
		}
		for optionID := range optionsByID {
			if keptOptions[optionID] {
				continue
			}
			if _, err = tx.Exec(r.Context(), `
				DELETE FROM product_modifier_options
				WHERE id=$1 AND organization_id=$2 AND group_id=$3
			`, optionID, s.OrganizationID, groupID); err != nil {
				fail(w, 503, "modifiers_unavailable", "No pudimos eliminar las opciones retiradas.")
				return
			}
		}
		for optionIndex, option := range group.Options {
			surcharge := strings.TrimSpace(option.Surcharge)
			if surcharge == "" {
				surcharge = "0"
			}
			optionID := optionIDs[groupIndex][optionIndex]
			if optionID == "" {
				_, err = tx.Exec(r.Context(), `
					INSERT INTO product_modifier_options(
					  organization_id,group_id,name,surcharge,sort_order
					) VALUES($1,$2,$3,$4,$5)
				`, s.OrganizationID, groupID, strings.TrimSpace(option.Name), surcharge, optionIndex)
			} else {
				_, err = tx.Exec(r.Context(), `
					UPDATE product_modifier_options
					SET name=$4,surcharge=$5,sort_order=$6,updated_at=now()
					WHERE id=$1 AND organization_id=$2 AND group_id=$3
				`, optionID, s.OrganizationID, groupID, strings.TrimSpace(option.Name), surcharge, optionIndex)
			}
			if err != nil {
				fail(w, 409, "modifier_option_conflict", "No pudimos guardar una opción de modificador.")
				return
			}
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "modifiers_unavailable", "No pudimos confirmar los modificadores.")
		return
	}

	a.audit(r, "product.modifiers_saved", "product", productID)
	out, err := loadProductModifiers(r.Context(), a.db, s.OrganizationID, productID)
	if err != nil {
		fail(w, 503, "modifiers_unavailable", "Los modificadores se guardaron, pero no pudimos recargarlos.")
		return
	}
	writeJSON(w, 200, out)
}
