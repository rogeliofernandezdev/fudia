package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"sort"
	"strings"
)

func (a *Orders) prepareItems(ctx context.Context, s scope, existingOrderID string, inputs []domain.OrderItemInput) ([]domain.PreparedOrderItem, float64, error) {
	prepared := make([]domain.PreparedOrderItem, 0, len(inputs))
	subtotal := 0.0
	plannedOptionUsage := map[domain.ComboOptionUsageKey]float64{}

	productIDs := make([]string, 0, len(inputs))
	for _, in := range inputs {
		if id := strings.TrimSpace(in.ProductID); id != "" {
			productIDs = append(productIDs, id)
		}
	}
	snapshots := map[string]domain.OrderProductSnapshot{}
	if len(productIDs) > 0 {
		var err error
		if snapshots, err = a.repo.OrderProducts(ctx, s, productIDs); err != nil {
			return nil, 0, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar el producto del pedido.")
		}
	}

	for _, in := range inputs {
		if in.Qty <= 0 {
			return nil, 0, orderFailure(apperror.Invalid, "invalid_order", "Cada línea necesita una cantidad mayor a cero.")
		}
		productIDValue := strings.TrimSpace(in.ProductID)
		if productIDValue == "" {
			return nil, 0, orderFailure(apperror.Invalid, "product_id_required", "Cada línea del pedido debe referenciar un producto.")
		}
		productID := &productIDValue

		snapshot, preloaded := snapshots[productIDValue]
		isCombo := snapshot.IsCombo
		if !preloaded {
			if err := func() error { var err error; isCombo, err = a.repo.ProductIsCombo(ctx, s, productID); return err }(); err != nil {
				return nil, 0, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar el producto del pedido.")
			}
		}

		if !isCombo {
			if len(in.Selections) > 0 {
				return nil, 0, orderFailure(apperror.Invalid, "invalid_combo_selection", "Las opciones enviadas no pertenecen a un menú o combo.")
			}
			if in.UnitPrice < 0 {
				return nil, 0, orderFailure(apperror.Invalid, "invalid_order", "Cada línea necesita un precio válido.")
			}
			productName, productPrice, available := snapshot.Name, snapshot.Price, snapshot.Available
			if !preloaded {
				err := func() error {
					var err error
					productName, productPrice, available, err = a.repo.ProductForOrder(ctx, s, productID)
					return err
				}()
				if errors.Is(err, ErrNotFound) {
					return nil, 0, orderFailure(apperror.NotFound, "product_not_found", "El producto ya no existe.")
				}
				if err != nil {
					return nil, 0, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar el producto.")
				}
			}
			if !available {
				return nil, 0, orderFailure(apperror.Conflict, "product_unavailable", productName+" no está disponible en este momento.")
			}
			// A preloaded product without modifier groups needs no lookup unless the
			// client sent modifiers, which must still be rejected as foreign.
			if preloaded && !snapshot.HasModifiers && len(in.Modifiers) == 0 {
				item := domain.PreparedOrderItem{
					ProductID: productID,
					Name:      productName,
					Qty:       in.Qty,
					UnitPrice: productPrice,
					Note:      strings.TrimSpace(in.Note),
					ItemType:  "product",
					Modifiers: []domain.PreparedOrderModifier{},
				}
				prepared = append(prepared, item)
				subtotal += item.Qty * item.UnitPrice
				continue
			}
			modifiers, modifierSurcharge, modifierErr := a.prepareModifiers(ctx, s, *productID, in.Modifiers)
			if modifierErr != nil {
				return nil, 0, modifierErr
			}
			item := domain.PreparedOrderItem{
				ProductID: productID,
				Name:      productName,
				Qty:       in.Qty,
				UnitPrice: productPrice + modifierSurcharge,
				Note:      strings.TrimSpace(in.Note),
				ItemType:  "product",
				Modifiers: modifiers,
			}
			prepared = append(prepared, item)
			subtotal += item.Qty * item.UnitPrice
			continue
		}

		if len(in.Modifiers) > 0 {
			return nil, 0, orderFailure(apperror.Invalid, "invalid_modifier_selection", "Los modificadores simples no pertenecen a un menú o combo.")
		}

		if existingOrderID != "" && strings.TrimSpace(in.ID) != "" && !in.Reprice {
			var existingName string
			var existingUnitPrice float64
			var existingProductID string
			var existingType string
			err := func() error {
				var err error
				existingName, existingUnitPrice, existingProductID, existingType, err = a.repo.ExistingComboItem(ctx, s, in, existingOrderID)
				return err
			}()
			if err == nil && existingType == "combo" && existingProductID == *productID {
				existingSelections, rowsErr := a.repo.ExistingSelections(ctx, s, strings.TrimSpace(in.ID))
				if rowsErr != nil {
					return nil, 0, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar la configuración existente del menú.")
				}
				existingKeys := map[string]bool{}
				for _, sel := range existingSelections {
					existingKeys[sel.GroupID+":"+sel.ProductID] = true
				}
				requestedKeys := map[string]bool{}
				for _, selection := range in.Selections {
					requestedKeys[strings.TrimSpace(selection.GroupID)+":"+strings.TrimSpace(selection.ProductID)] = true
				}
				sameSelections := len(in.Selections) == len(requestedKeys) && len(existingKeys) == len(requestedKeys)
				if sameSelections {
					for key := range existingKeys {
						if !requestedKeys[key] {
							sameSelections = false
							break
						}
					}
				}
				if sameSelections {
					item := domain.PreparedOrderItem{
						ProductID:  productID,
						Name:       existingName,
						Qty:        in.Qty,
						UnitPrice:  existingUnitPrice,
						Note:       strings.TrimSpace(in.Note),
						ItemType:   "combo",
						Selections: existingSelections,
					}
					prepared = append(prepared, item)
					subtotal += item.Qty * item.UnitPrice
					for _, sel := range existingSelections {
						plannedOptionUsage[domain.ComboOptionUsageKey{ComboProductID: *productID, GroupName: sel.GroupName, ProductID: sel.ProductID}] += item.Qty
					}
					continue
				}
			} else if err != nil && !errors.Is(err, ErrNotFound) {
				return nil, 0, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar el menú existente.")
			}
		}

		var comboName string
		var basePrice float64
		var comboAvailable bool
		err := func() error {
			var err error
			comboName, basePrice, comboAvailable, err = a.repo.ComboForOrder(ctx, s, productID)
			return err
		}()
		if errors.Is(err, ErrNotFound) {
			return nil, 0, orderFailure(apperror.NotFound, "combo_not_found", "El menú o combo ya no existe.")
		}
		if err != nil {
			return nil, 0, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar el menú o combo.")
		}
		if !comboAvailable {
			return nil, 0, orderFailure(apperror.Conflict, "combo_unavailable", comboName+" no está disponible en este momento.")
		}

		groups, groupOrder, err := a.repo.ComboGroups(ctx, s, *productID)
		if err != nil {
			return nil, 0, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar las opciones del menú.")
		}
		selectedByGroup := map[string][]string{}
		seen := map[string]bool{}
		for _, selection := range in.Selections {
			groupID := strings.TrimSpace(selection.GroupID)
			optionID := strings.TrimSpace(selection.ProductID)
			if groupID == "" || optionID == "" {
				return nil, 0, orderFailure(apperror.Invalid, "invalid_combo_selection", "Revisa las opciones seleccionadas del menú.")
			}
			key := groupID + ":" + optionID
			if seen[key] {
				return nil, 0, orderFailure(apperror.Invalid, "invalid_combo_selection", "Una opción del menú está repetida.")
			}
			seen[key] = true
			selectedByGroup[groupID] = append(selectedByGroup[groupID], optionID)
		}

		selections := []domain.PreparedOrderSelection{}
		surchargeTotal := 0.0
		for groupID := range selectedByGroup {
			if _, ok := groups[groupID]; !ok {
				return nil, 0, orderFailure(apperror.Invalid, "invalid_combo_selection", "Una de las opciones no pertenece a este menú.")
			}
		}
		for _, groupID := range groupOrder {
			group := groups[groupID]
			selected := selectedByGroup[groupID]
			minimum := group.Min
			if group.Required && minimum < 1 {
				minimum = 1
			}
			if len(selected) < minimum || len(selected) > group.Max {
				return nil, 0, orderFailure(apperror.Invalid, "invalid_combo_selection", "Completa correctamente el grupo "+group.Name+".")
			}
			for _, optionID := range selected {
				option, ok := group.Options[optionID]
				if !ok {
					return nil, 0, orderFailure(apperror.Invalid, "invalid_combo_selection", "Una de las opciones no pertenece a "+group.Name+".")
				}
				if !option.Available {
					return nil, 0, orderFailure(apperror.Conflict, "combo_option_unavailable", option.Name+" ya no está disponible.")
				}
				selections = append(selections, domain.PreparedOrderSelection{
					GroupID:   group.ID,
					GroupName: group.Name,
					ProductID: option.ProductID,
					Name:      option.Name,
					Surcharge: option.Surcharge,
				})
				surchargeTotal += option.Surcharge
			}
		}

		item := domain.PreparedOrderItem{
			ProductID:  productID,
			Name:       comboName,
			Qty:        in.Qty,
			UnitPrice:  basePrice + surchargeTotal,
			Note:       strings.TrimSpace(in.Note),
			ItemType:   "combo",
			Selections: selections,
		}
		prepared = append(prepared, item)
		subtotal += item.Qty * item.UnitPrice
		for _, sel := range selections {
			plannedOptionUsage[domain.ComboOptionUsageKey{ComboProductID: *productID, GroupName: sel.GroupName, ProductID: sel.ProductID}] += item.Qty
		}
	}

	usageKeys := make([]domain.ComboOptionUsageKey, 0, len(plannedOptionUsage))
	for key := range plannedOptionUsage {
		usageKeys = append(usageKeys, key)
	}
	sort.Slice(usageKeys, func(i, j int) bool {
		if usageKeys[i].ComboProductID != usageKeys[j].ComboProductID {
			return usageKeys[i].ComboProductID < usageKeys[j].ComboProductID
		}
		if usageKeys[i].GroupName != usageKeys[j].GroupName {
			return usageKeys[i].GroupName < usageKeys[j].GroupName
		}
		return usageKeys[i].ProductID < usageKeys[j].ProductID
	})
	for _, key := range usageKeys {
		var existingUsage float64
		if existingOrderID != "" {
			err := func() error {
				var err error
				existingUsage, err = a.repo.ExistingOptionUsage(ctx, s, existingOrderID, key)
				return err
			}()
			if err != nil {
				return nil, 0, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar el cupo existente del menú.")
			}
		}

		var quota *int
		var optionName string
		err := func() error { var err error; quota, optionName, err = a.repo.LockOptionQuota(ctx, s, key); return err }()
		if errors.Is(err, ErrNotFound) {
			if existingOrderID != "" && plannedOptionUsage[key] <= existingUsage {
				continue
			}
			return nil, 0, orderFailure(apperror.Conflict, "combo_option_unavailable", "Una opción del menú ya no está disponible.")
		}
		if err != nil {
			return nil, 0, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar el cupo del menú.")
		}
		if quota == nil {
			continue
		}

		var externalUsage float64
		err = func() error {
			var err error
			externalUsage, err = a.repo.ExternalOptionUsage(ctx, s, key, existingOrderID)
			return err
		}()
		if err != nil {
			return nil, 0, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar el cupo del menú.")
		}

		allowedForOrder := float64(*quota) - externalUsage
		if allowedForOrder < 0 {
			allowedForOrder = 0
		}
		if existingUsage > allowedForOrder {
			allowedForOrder = existingUsage
		}
		if plannedOptionUsage[key] > allowedForOrder {
			return nil, 0, orderFailure(apperror.Conflict, "combo_option_quota_exceeded", optionName+" ya alcanzó el cupo reservado para este menú.")
		}
	}
	return prepared, subtotal, nil
}
func (a *Orders) prepareModifiers(ctx context.Context, s scope, productID string, inputs []domain.OrderItemModifierInput) ([]domain.PreparedOrderModifier, float64, error) {
	groups, groupOrder, err := a.repo.ModifierGroups(ctx, s, productID)
	if err != nil {
		return nil, 0, orderFailure(apperror.Unavailable, "modifiers_unavailable", "No pudimos validar los modificadores del producto.")
	}
	selectedByGroup := map[string][]string{}
	seen := map[string]bool{}
	for _, input := range inputs {
		groupID := strings.TrimSpace(input.GroupID)
		optionID := strings.TrimSpace(input.OptionID)
		if groupID == "" || optionID == "" {
			return nil, 0, orderFailure(apperror.Invalid, "invalid_modifier_selection", "Revisa los modificadores seleccionados.")
		}
		key := groupID + ":" + optionID
		if seen[key] {
			return nil, 0, orderFailure(apperror.Invalid, "invalid_modifier_selection", "Un modificador está repetido.")
		}
		seen[key] = true
		selectedByGroup[groupID] = append(selectedByGroup[groupID], optionID)
	}

	for groupID := range selectedByGroup {
		if _, exists := groups[groupID]; !exists {
			return nil, 0, orderFailure(apperror.Invalid, "invalid_modifier_selection", "Un modificador no pertenece a este producto.")
		}
	}

	prepared := []domain.PreparedOrderModifier{}
	surchargeTotal := 0.0
	for _, groupID := range groupOrder {
		current := groups[groupID]
		selected := selectedByGroup[groupID]
		minimum := current.Min
		if current.Required && minimum < 1 {
			minimum = 1
		}
		if len(selected) < minimum || len(selected) > current.Max {
			return nil, 0, orderFailure(apperror.Invalid, "invalid_modifier_selection", "Completa correctamente el grupo "+current.Name+".")
		}
		for _, optionID := range selected {
			selectedOption, exists := current.Options[optionID]
			if !exists {
				return nil, 0, orderFailure(apperror.Invalid, "invalid_modifier_selection", "Una opción no pertenece a "+current.Name+".")
			}
			prepared = append(prepared, domain.PreparedOrderModifier{
				GroupID: current.ID, GroupName: current.Name,
				OptionID: selectedOption.ID, Name: selectedOption.Name, Surcharge: selectedOption.Surcharge,
			})
			surchargeTotal += selectedOption.Surcharge
		}
	}
	return prepared, surchargeTotal, nil
}
