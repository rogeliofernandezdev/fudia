package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"math"
	"sort"
	"strings"
)

func (a *Stock) Transfer(ctx context.Context, s scope, in domain.TransferInput) (map[string]any, error) {
	var result map[string]any
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.ToLocationID = strings.TrimSpace(in.ToLocationID)
	in.Notes = strings.TrimSpace(in.Notes)
	if len(in.IdempotencyKey) > 120 || in.ToLocationID == "" || in.ToLocationID == s.LocationID || len(in.Notes) > 500 || len(in.Items) == 0 {
		return result, stockFailure(apperror.Invalid, "invalid_transfer", "Selecciona otro local y al menos un artículo.")
	}
	seen := map[string]bool{}
	for i := range in.Items {
		in.Items[i].InventoryItemID = strings.TrimSpace(in.Items[i].InventoryItemID)
		in.Items[i].Quantity = math.Round(in.Items[i].Quantity*1000) / 1000
		if in.Items[i].InventoryItemID == "" || in.Items[i].Quantity <= 0 || seen[in.Items[i].InventoryItemID] {
			return result, stockFailure(apperror.Invalid, "invalid_transfer", "Las líneas de transferencia no son válidas.")
		}
		seen[in.Items[i].InventoryItemID] = true
	}
	sort.Slice(in.Items, func(i, j int) bool { return in.Items[i].InventoryItemID < in.Items[j].InventoryItemID })

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, stockFailure(apperror.Unavailable, "transfer_unavailable", "No pudimos iniciar la transferencia.")
	}
	defer tx.Rollback(ctx)
	var destinationID, destination string
	if err = func() error {
		var err error
		destinationID, destination, err = a.repo.TransferDestination(tx.Context(), s, in)
		return err
	}(); errors.Is(err, ErrNotFound) {
		return result, stockFailure(apperror.NotFound, "destination_not_found", "El local destino no existe o está inactivo.")
	} else if err != nil {
		return result, stockFailure(apperror.Unavailable, "transfer_unavailable", "No pudimos validar el destino.")
	}
	// PostgreSQL accepts several spellings of the same UUID, while balances come
	// back keyed by the canonical one: use it from here on.
	in.ToLocationID = destinationID
	if in.ToLocationID == s.LocationID {
		return result, stockFailure(apperror.Invalid, "invalid_transfer", "Selecciona otro local y al menos un artículo.")
	}
	if in.IdempotencyKey != "" {
		var existingID, existingCode, existingTo string
		err = func() error {
			var err error
			existingID, existingCode, existingTo, err = a.repo.ExistingTransfer(tx.Context(), s, in)
			return err
		}()
		if err == nil {
			return map[string]any{"id": existingID, "code": existingCode, "toLocationId": existingTo, "idempotent": true}, nil
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return result, stockFailure(apperror.Unavailable, "transfer_unavailable", "No pudimos validar la transferencia previa.")
		}
	}
	var transferID, code string
	if err = func() error {
		var err error
		transferID, code, err = a.repo.CreateTransfer(tx.Context(), s, in)
		return err
	}(); err != nil {
		return result, stockFailure(apperror.Unavailable, "transfer_unavailable", "No pudimos crear la transferencia.")
	}

	for _, line := range in.Items {
		var productID *string
		var name string
		if err = func() error {
			var err error
			productID, name, err = a.repo.TransferItem(tx.Context(), s, line)
			return err
		}(); errors.Is(err, ErrNotFound) {
			return result, stockFailure(apperror.NotFound, "inventory_item_not_found", "Un artículo ya no existe.")
		} else if err != nil {
			return result, stockFailure(apperror.Unavailable, "transfer_unavailable", "No pudimos validar los artículos.")
		}
		if _, err = a.repo.EnsureTransferBalances(tx.Context(), s, line, in); err != nil {
			return result, stockFailure(apperror.Unavailable, "transfer_unavailable", "No pudimos preparar los saldos.")
		}
		balances, err := a.repo.LockTransferBalances(tx.Context(), s, line.InventoryItemID, in.ToLocationID)
		if err != nil {
			return result, stockFailure(apperror.Unavailable, "transfer_unavailable", "No pudimos bloquear los saldos.")
		}
		src, srcFound := balances[s.LocationID]
		dst, dstFound := balances[in.ToLocationID]
		if !srcFound || !dstFound {
			return result, stockFailure(apperror.Unavailable, "transfer_unavailable", "No pudimos bloquear los saldos.")
		}
		if line.Quantity > src.Quantity+0.000001 {
			return result, stockFailure(apperror.Conflict, "insufficient_stock", "No hay stock suficiente de "+name+" para transferir.")
		}
		srcAfter := math.Round((src.Quantity-line.Quantity)*1000) / 1000
		dstAfter := math.Round((dst.Quantity+line.Quantity)*1000) / 1000
		if dstAfter >= domain.MaxStockQuantity {
			return result, stockLimitFailure()
		}
		dstAvg := src.Cost
		if dstAfter > 0 {
			dstAvg = math.Round(((dst.Quantity*dst.Cost)+(line.Quantity*src.Cost))/dstAfter*10000) / 10000
		}
		srcAvg := src.Cost
		if _, err = a.repo.TransferSourceBalance(tx.Context(), s, line, srcAfter, srcAvg); err != nil {
			return result, stockFailure(apperror.Unavailable, "transfer_unavailable", "No pudimos descontar el origen.")
		}
		if _, err = a.repo.TransferDestinationBalance(tx.Context(), s, line, in, dstAfter, dstAvg); err != nil {
			return result, stockFailure(apperror.Unavailable, "transfer_unavailable", "No pudimos ingresar el destino.")
		}
		if _, err = a.repo.CreateTransferItem(tx.Context(), s, transferID, line, src); err != nil {
			return result, stockFailure(apperror.Unavailable, "transfer_unavailable", "No pudimos guardar el detalle.")
		}
		if err = a.movement(tx.Context(), s, productID, line.InventoryItemID, "transfer_out", -line.Quantity, srcAfter, src.Cost, "inventory_transfer", transferID, code+" → "+destination); err != nil {
			return result, stockFailure(apperror.Unavailable, "transfer_unavailable", "No pudimos registrar el Kárdex de origen.")
		}
		valueDelta := math.Round(line.Quantity*src.Cost*10000) / 10000
		balanceValue := math.Round(dstAfter*dstAvg*10000) / 10000
		if _, err = a.repo.TransferIncomingMovement(tx.Context(), s, in, productID, line, dstAfter, transferID, code, src, valueDelta, balanceValue); err != nil {
			return result, stockFailure(apperror.Unavailable, "transfer_unavailable", "No pudimos registrar el Kárdex de destino.")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return result, stockFailure(apperror.Unavailable, "transfer_unavailable", "No pudimos confirmar la transferencia.")
	}
	_ = a.audit.Record(ctx, s, "inventory.transfer_created", "inventory_transfer", transferID)
	return map[string]any{"id": transferID, "code": code, "toLocationId": in.ToLocationID, "toLocationName": destination}, nil

}

type TransfersPage struct {
	Items    []domain.TransferSummary `json:"items"`
	Total    int                      `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"pageSize"`
}

func (a *Stock) Transfers(ctx context.Context, s scope, f StockFilter) (TransfersPage, error) {
	out, err := a.repo.Transfers(ctx, s, f)
	if err != nil {
		return out, stockFailure(apperror.Unavailable, "transfers_unavailable", "No pudimos cargar las transferencias.")
	}
	return out, nil
}
