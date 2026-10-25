package application

import (
	"context"
	"errors"
	"fmt"
	menucontracts "github.com/foods-platform/foods-backend/internal/modules/menu/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"strings"
	"time"
)

type KitchenPage struct {
	DestinationOptions []map[string]string    `json:"destinationOptions"`
	Items              []domain.KitchenTicket `json:"items"`
	Counts             map[string]int         `json:"counts"`
	ChannelOptions     []map[string]string    `json:"channelOptions"`
	ServerTime         time.Time              `json:"serverTime"`
}

func (a *Orders) Kitchen(ctx context.Context, s scope, channel, destination string) (KitchenPage, error) {
	out := KitchenPage{Items: []domain.KitchenTicket{}, Counts: map[string]int{"confirmado": 0, "preparando": 0, "listo": 0}, DestinationOptions: menucontracts.ServiceDestinations[:2], ChannelOptions: domain.ServiceChannels}
	channel = domain.NormalizeChannel(strings.TrimSpace(channel))
	destination = strings.TrimSpace(destination)
	if channel != "" && !domain.ValidOrderChannel(channel) {
		return out, orderFailure(apperror.Invalid, "invalid_channel", "El canal seleccionado no es válido.")
	}
	if destination != "" && destination != "kitchen" && destination != "bar" {
		return out, orderFailure(apperror.Invalid, "invalid_destination", "Selecciona Cocina o Barra.")
	}
	legacy, err := a.repo.LegacyKitchenOrders(ctx, s, channel, destination)
	if err != nil {
		return out, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos cargar las comandas de cocina.")
	}
	for _, item := range legacy {
		items, err := a.repo.Items(ctx, s, item.ID, "")
		if err != nil {
			return out, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos cargar los productos de las comandas.")
		}
		target, err := a.repo.TargetMinutes(ctx, s, item.ID, "")
		if err != nil {
			return out, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos calcular los tiempos de preparación.")
		}
		out.Items = append(out.Items, domain.KitchenTicket{ID: item.ID, OrderID: item.ID, Code: item.Code, Channel: item.Channel, Status: item.Status, CustomerName: item.CustomerName, TableName: item.TableName, Notes: item.Notes, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, TargetMinutes: target, Items: items})
		out.Counts[item.Status]++
	}
	rounds, err := a.repo.KitchenRounds(ctx, s, channel, destination)
	if err != nil {
		return out, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos cargar las rondas de cocina.")
	}
	for _, ticket := range rounds {
		ticket.Code = fmt.Sprintf("%s · R%d", ticket.Code, ticket.RoundNumber)
		ticket.Items, err = a.repo.Items(ctx, s, ticket.OrderID, ticket.ID)
		if err != nil {
			return out, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos cargar los productos de una ronda.")
		}
		ticket.TargetMinutes, err = a.repo.TargetMinutes(ctx, s, ticket.OrderID, ticket.ID)
		if err != nil {
			return out, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos calcular el tiempo de una ronda.")
		}
		out.Items = append(out.Items, ticket)
		out.Counts[ticket.Status]++
	}
	if destination == "bar" {
		out.Items = []domain.KitchenTicket{}
		out.Counts = map[string]int{"confirmado": 0, "preparando": 0, "listo": 0}
	}
	current, err := a.repo.ServiceTickets(ctx, s, channel, destination)
	if err != nil {
		return out, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos cargar la preparación por área.")
	}
	for _, ticket := range current {
		out.Items = append(out.Items, ticket)
		out.Counts[ticket.Status]++
	}
	for i := range out.Items {
		out.Items[i].Channel = domain.NormalizeChannel(out.Items[i].Channel)
	}
	out.ServerTime = time.Now().UTC()
	return out, nil
}
func (a *Orders) allowHistoricalTicket(ctx context.Context, s scope, orderID string) error {
	exists, err := a.repo.HasServiceItems(ctx, s, orderID)
	if err != nil {
		return orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos validar la comanda.")
	}
	if exists {
		return orderFailure(apperror.NotFound, "kitchen_ticket_not_found", "La comanda no existe en esta estación.")
	}
	return nil
}
func (a *Orders) tryStationTicket(ctx context.Context, s scope, id, status string) (bool, error) {
	destination := "kitchen"
	if strings.HasSuffix(id, "~bar") {
		destination = "bar"
		id = strings.TrimSuffix(id, "~bar")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return true, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos actualizar la comanda.")
	}
	defer tx.Rollback(ctx)
	var orderID string
	err = func() error {
		var err error
		orderID, err = a.repo.LockStationOrder(tx.Context(), s, id, destination)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		if destination == "bar" {
			return true, orderFailure(apperror.NotFound, "kitchen_ticket_not_found", "La comanda no existe.")
		}
		return false, nil
	}
	if err != nil {
		return true, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos validar la comanda.")
	}
	var current string
	err = func() error {
		var err error
		current, err = a.repo.StationStatus(tx.Context(), s, orderID, id, destination)
		return err
	}()
	if err != nil {
		return true, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos consultar la preparación.")
	}
	if !domain.ValidKitchenTransition(current, status) {
		return true, orderFailure(apperror.Conflict, "invalid_kitchen_transition", "La comanda cambió de estado. Actualiza antes de continuar.")
	}
	_, err = a.repo.UpdateStationItems(tx.Context(), s, orderID, id, destination, status, current)
	if err == nil {
		_, err = a.repo.UpdateStationRound(tx.Context(), s, id)
	}
	if err == nil {
		err = a.repo.SyncServiceStatus(tx.Context(), s, orderID)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return true, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos guardar la preparación.")
	}
	_ = a.audit.Record(ctx, s, "preparation."+status, "order", orderID)
	return true, nil
}
func (a *Orders) UpdateKitchen(ctx context.Context, s scope, id string, in OrderStatusInput) error {
	in.Status = strings.TrimSpace(in.Status)
	if in.Status != "preparando" && in.Status != "listo" {
		return orderFailure(apperror.Invalid, "invalid_kitchen_status", "Cocina solo puede iniciar preparación o marcar una comanda como lista.")
	}

	if handled, err := a.tryStationTicket(ctx, s, id, in.Status); handled || err != nil {
		return err
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos actualizar la comanda.")
	}
	defer tx.Rollback(ctx)

	var roundCurrent, roundOrderID string
	err = func() error {
		var err error
		roundCurrent, roundOrderID, err = a.repo.LockHistoricalRound(tx.Context(), s, id)
		return err
	}()
	if err == nil {
		if err := a.allowHistoricalTicket(tx.Context(), s, roundOrderID); err != nil {
			return err
		}
		if !domain.ValidKitchenTransition(roundCurrent, in.Status) {
			return orderFailure(apperror.Conflict, "invalid_kitchen_transition", "La ronda cambió de estado. Actualiza la cola antes de continuar.")
		}
		if _, err = a.repo.UpdateHistoricalRound(tx.Context(), s, id, in); err != nil {
			return orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos actualizar la ronda.")
		}
		if err = a.syncKitchenStatus(tx.Context(), s, roundOrderID); err != nil {
			return orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos actualizar el estado general de la comanda.")
		}
		if err = tx.Commit(ctx); err != nil {
			return orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos confirmar el cambio de la ronda.")
		}
		action := "kitchen.round.started"
		if in.Status == "listo" {
			action = "kitchen.round.ready"
		}
		_ = a.audit.Record(ctx, s, action, "order_round", id)
		return nil
	}
	if !errors.Is(err, ErrNotFound) {
		return orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos validar la ronda.")
	}

	var current string
	err = func() error {
		var err error
		current, err = a.repo.LockHistoricalOrder(tx.Context(), s, id)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return orderFailure(apperror.NotFound, "order_not_found", "El pedido o ronda no existe.")
	}
	if err != nil {
		return orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos validar la comanda.")
	}
	if err := a.allowHistoricalTicket(tx.Context(), s, id); err != nil {
		return err
	}
	if !domain.ValidKitchenTransition(current, in.Status) {
		return orderFailure(apperror.Conflict, "invalid_kitchen_transition", "La comanda cambió de estado. Actualiza la cola antes de continuar.")
	}
	tag, err := a.repo.UpdateHistoricalOrder(tx.Context(), s, id, in)
	if err != nil || tag == 0 {
		return orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos actualizar la comanda.")
	}
	if err = tx.Commit(ctx); err != nil {
		return orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos actualizar la comanda.")
	}
	action := "kitchen.started"
	if in.Status == "listo" {
		action = "kitchen.ready"
	}
	_ = a.audit.Record(ctx, s, action, "order", id)
	return nil

}
func (a *Orders) syncKitchenStatus(ctx context.Context, s scope, orderID string) error {
	var hasService bool
	if err := func() error { var err error; hasService, err = a.repo.HasServiceItems(ctx, s, orderID); return err }(); err != nil {
		return err
	}
	if hasService {
		return a.repo.SyncServiceStatus(ctx, s, orderID)
	}
	var confirmed, preparing, ready int
	if err := func() error {
		var err error
		confirmed, preparing, ready, err = a.repo.KitchenRoundCounts(ctx, s, orderID)
		return err
	}(); err != nil {
		return err
	}
	if confirmed+preparing+ready == 0 {
		return nil
	}
	next := "listo"
	if confirmed > 0 {
		next = "confirmado"
	}
	if preparing > 0 {
		next = "preparando"
	}
	_, err := a.repo.SyncHistoricalKitchenStatus(ctx, s, orderID, next)
	return err
}
