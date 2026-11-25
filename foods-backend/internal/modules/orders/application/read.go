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
	Items    []domain.Order `json:"items"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
	// ChannelCounts respeta búsqueda y estado, pero abarca todos los canales antes de paginar.
	ChannelCounts  map[string]int      `json:"channelCounts"`
	ChannelOptions []map[string]string `json:"channelOptions"`
	// ManualChannelOptions son los tipos que se registran sin mesa (Mostrador, Recojo y Delivery).
	ManualChannelOptions []map[string]string `json:"manualChannelOptions"`
	// StatusOptions lista estados operativos con pedidos para la búsqueda y canal, más el elegido.
	StatusOptions  []map[string]string `json:"statusOptions"`
	CurrencySymbol string              `json:"currencySymbol"`
	// PresentStatuses lo llena el repositorio; no se serializa.
	PresentStatuses map[string]bool `json:"-"`
}

func (a *Orders) List(ctx context.Context, s scope, f Filter) (Page, error) {
	f.Channel = domain.NormalizeChannel(f.Channel)
	out, err := a.repo.List(ctx, s, f)
	if err != nil {
		return out, orderFailure(apperror.Unavailable, "orders_unavailable", "No pudimos cargar los pedidos.")
	}
	out.ChannelOptions = domain.ServiceChannels
	out.ManualChannelOptions = domain.ManualChannels
	for i := range out.Items {
		out.Items[i].OperationalStatus = domain.OperationalOrderStatus(out.Items[i].Status)
	}
	out.StatusOptions = PresentStatusOptions(out.PresentStatuses, f.Status)
	return out, nil
}

// PresentStatusOptions conserva el orden del catálogo y deja solo los estados presentes; el estado
// elegido se mantiene aunque ya no tenga pedidos, para que el filtro nunca muestre un valor ausente.
func PresentStatusOptions(present map[string]bool, selected string) []map[string]string {
	options := []map[string]string{}
	operational := map[string]bool{}
	for status, exists := range present {
		if exists {
			operational[domain.OperationalOrderStatus(status)] = true
		}
	}
	selected = domain.OperationalOrderStatus(selected)
	for _, option := range domain.Statuses {
		if option["value"] == "en_camino" {
			continue
		}
		if operational[option["value"]] || option["value"] == selected {
			options = append(options, option)
		}
	}
	return options
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
	o, paid, err := a.repo.Detail(ctx, s, id)
	if errors.Is(err, ErrNotFound) {
		return result, orderFailure(apperror.NotFound, "order_not_found", "El pedido no existe.")
	}
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos cargar el pedido.")
	}
	for _, it := range o.Items {
		if q, parseErr := strconv.ParseFloat(it.Qty, 64); parseErr == nil {
			o.ItemCount += int(q)
		}
	}
	if err = domain.ApplyPaymentSummary(&o, paid); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos cargar el estado de cobro del pedido.")
	}
	return o, nil
}
