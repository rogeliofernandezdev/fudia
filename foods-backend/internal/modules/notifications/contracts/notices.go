// Package contracts publica cómo otros módulos avisan al personal sin depender
// de la implementación de Notifications.
package contracts

import (
	"context"

	"github.com/foods-platform/foods-backend/internal/modules/notifications/domain"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

type Notice = domain.Input
type CustomerNotice = domain.CustomerInput

// Publisher registra un aviso para el personal del local del alcance. Es de
// mejor esfuerzo: quien publica lo hace después de confirmar su transacción y
// un fallo no revierte la operación de negocio.
type Publisher interface {
	Publish(context.Context, requestscope.Scope, Notice) error
	// PublishCustomer deja un aviso para el cliente de WhatsApp en la bandeja
	// de salida que Concierge reclama y envía.
	PublishCustomer(context.Context, requestscope.Scope, CustomerNotice) error
}

var (
	TableOrder          = domain.TableOrder
	CounterOrder        = domain.CounterOrder
	RemoteOrder         = domain.RemoteOrder
	BillRequested       = domain.BillRequested
	AssistanceRequested = domain.AssistanceRequested
	StaleAccount        = domain.StaleAccount
	OrderReady          = domain.OrderReady
	DeliveryUnassigned  = domain.DeliveryUnassigned
	Money               = domain.Money
	// OrderReadyForCustomer devuelve false si el pedido no avisa al cliente.
	OrderReadyForCustomer = domain.OrderReadyForCustomer
	// OrderSentForCustomer devuelve false si falta el WhatsApp de la ronda.
	OrderSentForCustomer = domain.OrderSentForCustomer
	// OrderOnTheWayForCustomer devuelve false si el pedido no es delivery de WhatsApp.
	OrderOnTheWayForCustomer = domain.OrderOnTheWayForCustomer
)
