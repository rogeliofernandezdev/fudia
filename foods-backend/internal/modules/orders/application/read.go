package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"strconv"
)

type Filter struct {
	Query, Channel, Status string
	Page, Size             int
}
type Page struct {
	Items          []domain.Order      `json:"items"`
	Total          int                 `json:"total"`
	Page           int                 `json:"page"`
	PageSize       int                 `json:"pageSize"`
	ChannelCounts  map[string]int      `json:"channelCounts"`
	ChannelOptions []map[string]string `json:"channelOptions"`
	StatusOptions  []map[string]string `json:"statusOptions"`
	CurrencySymbol string              `json:"currencySymbol"`
}

func (a *Orders) List(ctx context.Context, s scope, f Filter) (Page, error) {
	out, err := a.repo.List(ctx, s, f)
	if err != nil {
		return out, orderFailure(apperror.Unavailable, "orders_unavailable", "No pudimos cargar los pedidos.")
	}
	out.ChannelOptions = domain.Channels
	out.StatusOptions = domain.Statuses
	return out, nil
}
func (a *Orders) Floor(ctx context.Context, s scope) ([]domain.FloorTable, error) {
	out, err := a.repo.Floor(ctx, s)
	if err != nil {
		return nil, orderFailure(apperror.Unavailable, "orders_unavailable", "No pudimos cargar el salón.")
	}
	for i := range out {
		t := &out[i]
		if t.Order == nil {
			continue
		}
		paid, err := strconv.ParseFloat(t.PaidText, 64)
		if err != nil {
			return nil, orderFailure(apperror.Unavailable, "orders_unavailable", "No pudimos calcular el estado de cobro del salón.")
		}
		if err = domain.ApplyPaymentSummary(t.Order, paid); err != nil {
			return nil, orderFailure(apperror.Unavailable, "orders_unavailable", "No pudimos calcular el estado de cobro del salón.")
		}
	}
	return out, nil
}
func (a *Orders) Get(ctx context.Context, s scope, id string) (domain.Order, error) {
	var result domain.Order
	o, err := a.repo.Record(ctx, s, id)
	if errors.Is(err, ErrNotFound) {
		return result, orderFailure(apperror.NotFound, "order_not_found", "El pedido no existe.")
	}
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos cargar el pedido.")
	}
	o.ServiceItems, err = a.repo.ServiceItems(ctx, s, o.ID)
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos consultar las entregas.")
	}
	o.Items, err = a.repo.Items(ctx, s, o.ID, "")
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos cargar los productos del pedido.")
	}
	for _, it := range o.Items {
		if q, parseErr := strconv.ParseFloat(it.Qty, 64); parseErr == nil {
			o.ItemCount += int(q)
		}
	}
	if err = a.paymentSummary(ctx, s, &o); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos cargar el estado de cobro del pedido.")
	}
	return o, nil

}
