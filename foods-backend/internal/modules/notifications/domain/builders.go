package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// Los constructores reciben hechos del evento y devuelven el aviso con su
// texto final; los módulos que publican no redactan mensajes al personal.

// TableOrder avisa a los mozos de un pedido por el QR de una mesa (nuevo o una ronda adicional).
func TableOrder(tableID, tableName, orderID, customer string, products int, requestID string) Input {
	return Input{
		Kind: KindTableOrder, Audience: AudienceService,
		Title:   Join(tableName, "nuevo pedido"),
		Detail:  Join(Products(products), customer),
		OrderID: orderID, TableID: tableID, DedupeKey: dedupe("order", orderID, requestID),
	}
}

// CounterOrder avisa a caja de una venta por mostrador pedida desde su QR.
func CounterOrder(counterName, orderID, code, customer string, products int, total string) Input {
	return Input{
		Kind: KindCounterOrder, Audience: AudienceCashier,
		Title:   strings.TrimSpace("Nuevo pedido en " + counterName),
		Detail:  Join(code, customer, Products(products), total),
		OrderID: orderID, DedupeKey: dedupe("order", orderID, ""),
	}
}

// RemoteOrder avisa a caja de un pedido de delivery o para recoger hecho por WhatsApp.
func RemoteOrder(channel, orderID, code, customer string, products int, total string) Input {
	kind, title := KindDelivery, "Nuevo pedido de delivery"
	if channel != "delivery" {
		kind, title = KindPickup, "Nuevo pedido para recoger"
	}
	return Input{
		Kind: kind, Audience: AudienceCashier, Title: title,
		Detail:  Join(code, customer, Products(products), total),
		OrderID: orderID, DedupeKey: dedupe("order", orderID, ""),
	}
}

// BillRequested avisa a mozos y caja de que una mesa pidió la cuenta.
func BillRequested(tableID, tableName, orderID, remaining, requestID string) Input {
	key, detail := "", ""
	if requestID != "" {
		key = dedupe("bill", orderID, requestID)
	}
	if remaining != "" {
		detail = "Saldo " + remaining
	}
	return Input{
		Kind: KindBill, Audience: AudienceAll,
		Title:   strings.TrimSpace(tableName + " pidió la cuenta"),
		Detail:  detail,
		OrderID: orderID, TableID: tableID, DedupeKey: key,
	}
}

// AssistanceRequested avisa a los mozos de que una mesa pidió atención del personal.
func AssistanceRequested(tableID, tableName, reason, handoffID string) Input {
	return Input{
		Kind: KindAssistance, Audience: AudienceService,
		Title:   strings.TrimSpace(tableName + " solicita atención"),
		Detail:  reason,
		TableID: tableID, DedupeKey: dedupe("handoff", handoffID, ""),
	}
}

// StaleAccount avisa a mozos y caja de que alguien escaneó el QR de una mesa cuya
// cuenta anterior sigue abierta sin pedidos nuevos: el cliente anterior pudo irse
// sin pagar. Un aviso por episodio (staleSince); se resuelve al cobrar, cancelar o
// agregar productos.
func StaleAccount(tableID, tableName, orderID, staleSince string) Input {
	return Input{
		Kind: KindStaleAccount, Audience: AudienceAll,
		Title:   Join(tableName, "cuenta anterior sin cerrar"),
		Detail:  "Un cliente escaneó el QR. Cobra o cancela la cuenta anterior para atenderlo.",
		OrderID: orderID, TableID: tableID, DedupeKey: dedupe("stale", orderID, staleSince),
	}
}

// DeliveryUnassigned avisa a caja que ninguna central de Delivery Fudia aceptó el
// pedido en esta vuelta (round): puede reintentar o usar su delivery propio.
func DeliveryUnassigned(orderID, code, customer string, round int) Input {
	return Input{
		Kind: KindDeliveryUnassigned, Audience: AudienceCashier,
		Title:   "Ninguna central aceptó el delivery",
		Detail:  Join(code, customer, "Reintenta o usa tu delivery propio"),
		OrderID: orderID, DedupeKey: dedupe("delivery-unassigned", orderID, strconv.Itoa(round)),
	}
}

// OrderReady avisa que una comanda está lista para entregar. En mesa la
// entrega el mozo; mostrador, recojo y delivery se avisan también a caja.
func OrderReady(ticketID, station, orderID, tableID, tableName, code, customer string) Input {
	in := Input{Kind: KindOrderReady, OrderID: orderID, TableID: tableID, DedupeKey: dedupe("ready", ticketID, station)}
	label := "Cocina"
	if station == "bar" {
		label = "Barra"
	}
	if tableName != "" {
		in.Audience, in.Title, in.Detail = AudienceService, Join(tableName, "listo para entregar"), Join(label, code)
		return in
	}
	subject := code
	if customer != "" {
		subject = "Pedido de " + customer
	}
	in.Audience, in.Title, in.Detail = AudienceAll, strings.TrimSpace(subject+" listo para entregar"), Join(label, code)
	return in
}

// Money formatea un importe con el símbolo de la moneda del local.
func Money(symbol, amount string) string {
	value, err := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%s %.2f", symbol, value))
}

func dedupe(event, id, extra string) string {
	if id == "" {
		return ""
	}
	if extra == "" {
		return event + ":" + id
	}
	return event + ":" + id + ":" + extra
}
