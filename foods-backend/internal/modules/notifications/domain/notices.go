// Package domain define los avisos al personal: qué se avisa, a quién y con qué texto.
package domain

import (
	"fmt"
	"strings"
)

const (
	KindTableOrder   = "table_order"
	KindCounterOrder = "counter_order"
	KindDelivery     = "delivery_order"
	KindPickup       = "pickup_order"
	KindOrderReady   = "order_ready"
	KindBill         = "bill_requested"
	KindAssistance   = "assistance_requested"
	KindStaleAccount = "stale_account"
	// KindDeliveryUnassigned: ninguna central de Delivery Fudia aceptó el pedido.
	KindDeliveryUnassigned = "delivery_unassigned"
)

// Audiencias: el servicio de salón (mozos), la caja (cajeros) o ambos.
const (
	AudienceService = "service"
	AudienceCashier = "cashier"
	AudienceAll     = "all"
)

// Notice es un aviso tal como lo ve un usuario en la campana.
type Notice struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	OrderID   string `json:"orderId,omitempty"`
	TableID   string `json:"tableId,omitempty"`
	CreatedAt string `json:"createdAt"`
	Read      bool   `json:"read"`
}

// Feed es la campana: los avisos recientes y cuántos siguen sin leer.
type Feed struct {
	Items  []Notice `json:"items"`
	Unread int      `json:"unread"`
}

// Input es un aviso por publicar. DedupeKey identifica el evento de origen
// para que un reintento no lo duplique.
type Input struct {
	Kind, Audience, Title, Detail string
	OrderID, TableID, DedupeKey   string
}

// Audiences devuelve las audiencias visibles para un usuario según sus permisos.
func Audiences(service, cashier bool) []string {
	out := []string{}
	if service {
		out = append(out, AudienceService)
	}
	if cashier {
		out = append(out, AudienceCashier)
	}
	if len(out) > 0 {
		out = append(out, AudienceAll)
	}
	return out
}

// Normalize recorta los textos a sus límites y rechaza tipos o audiencias desconocidos.
func Normalize(in Input) (Input, bool) {
	switch in.Kind {
	case KindTableOrder, KindCounterOrder, KindDelivery, KindPickup, KindOrderReady, KindBill, KindAssistance, KindStaleAccount, KindDeliveryUnassigned:
	default:
		return in, false
	}
	if in.Audience != AudienceService && in.Audience != AudienceCashier && in.Audience != AudienceAll {
		return in, false
	}
	in.Title = clip(in.Title, 120)
	in.Detail = clip(in.Detail, 240)
	in.DedupeKey = clip(in.DedupeKey, 200)
	in.OrderID = strings.TrimSpace(in.OrderID)
	in.TableID = strings.TrimSpace(in.TableID)
	return in, in.Title != ""
}

// Products describe la cantidad de productos de un pedido.
func Products(count int) string {
	if count == 1 {
		return "1 producto"
	}
	return fmt.Sprintf("%d productos", count)
}

// Join une las partes no vacías del detalle con « · ».
func Join(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, " · ")
}

func clip(value string, limit int) string {
	value = strings.TrimSpace(value)
	if runes := []rune(value); len(runes) > limit {
		value = strings.TrimSpace(string(runes[:limit]))
	}
	return value
}
