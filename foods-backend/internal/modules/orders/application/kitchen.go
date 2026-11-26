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
	queues, err := a.repo.KitchenQueues(ctx, s, channel, destination)
	if err != nil {
		return out, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos cargar las comandas de cocina.")
	}
	legacy, rounds := queues.Legacy, queues.Rounds
	refs := make([]domain.TicketRef, 0, len(legacy)+len(rounds))
	for _, item := range legacy {
		refs = append(refs, domain.TicketRef{OrderID: item.ID})
	}
	for _, ticket := range rounds {
		refs = append(refs, domain.TicketRef{OrderID: ticket.OrderID, RoundID: ticket.ID})
	}
	contents, err := a.repo.TicketContents(ctx, s, refs)
	if err != nil {
		return out, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos cargar los productos de las comandas.")
	}
	for i, item := range legacy {
		out.Items = append(out.Items, domain.KitchenTicket{ID: item.ID, OrderID: item.ID, Code: item.Code, Channel: item.Channel, Status: item.Status, CustomerName: item.CustomerName, TableName: item.TableName, Notes: item.Notes, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, TargetMinutes: contents[i].TargetMinutes, Items: contents[i].Items})
		out.Counts[item.Status]++
	}
	for i, ticket := range rounds {
		content := contents[len(legacy)+i]
		ticket.Code = fmt.Sprintf("%s · R%d", ticket.Code, ticket.RoundNumber)
		ticket.Items, ticket.TargetMinutes = content.Items, content.TargetMinutes
		out.Items = append(out.Items, ticket)
		out.Counts[ticket.Status]++
	}
	if destination == "bar" {
		out.Items = []domain.KitchenTicket{}
		out.Counts = map[string]int{"confirmado": 0, "preparando": 0, "listo": 0}
	}
	current := queues.Service
	for _, ticket := range current {
		out.Items = append(out.Items, ticket)
		out.Counts[ticket.Status]++
	}
	for i := range out.Items {
		out.Items[i].Origin = domain.Origin(out.Items[i].Channel)
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

// stationTicket separa la estación del id de comanda: "~bar" es Barra y lo demás Cocina.
func stationTicket(id string) (string, string) {
	if strings.HasSuffix(id, "~bar") {
		return strings.TrimSuffix(id, "~bar"), "bar"
	}
	return id, "kitchen"
}

// lockStation bloquea el pedido de la comanda y devuelve su estado derivado de los
// platos. ErrNotFound indica que la comanda no existe en esa estación.
func (a *Orders) lockStation(ctx context.Context, s scope, id, destination string) (string, string, error) {
	orderID, err := a.repo.LockStationOrder(ctx, s, id, destination)
	if err != nil {
		return "", "", err
	}
	current, err := a.repo.StationStatus(ctx, s, orderID, id, destination)
	if err != nil {
		return "", "", orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos consultar la preparación.")
	}
	return orderID, current, nil
}

// syncStation recalcula la ronda y el estado general del pedido desde sus platos.
func (a *Orders) syncStation(ctx context.Context, s scope, orderID, id string) error {
	if _, err := a.repo.UpdateStationRound(ctx, s, id); err != nil {
		return err
	}
	return a.repo.SyncServiceStatus(ctx, s, orderID)
}

// stationChanged registra el cambio de la comanda y, si quedó lista, avisa al
// personal. Es el mismo efecto para «Marcar listo» y para el último plato listo.
func (a *Orders) stationChanged(ctx context.Context, s scope, orderID, id, destination, status string) {
	_ = a.audit.Record(ctx, s, "preparation."+status, "order", orderID)
	if status == "listo" {
		a.noticeReady(ctx, s, orderID, id, destination)
		return
	}
	// Delivery Fudia ofrece el pedido a las centrales cuando cocina inicia la preparación.
	a.dispatchChanged(ctx, s, orderID)
}

func (a *Orders) tryStationTicket(ctx context.Context, s scope, rawID, status string) (bool, error) {
	id, destination := stationTicket(rawID)
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return true, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos actualizar la comanda.")
	}
	defer tx.Rollback(ctx)
	orderID, current, err := a.lockStation(tx.Context(), s, id, destination)
	if errors.Is(err, ErrNotFound) {
		if destination == "bar" {
			return true, orderFailure(apperror.NotFound, "kitchen_ticket_not_found", "La comanda no existe.")
		}
		return false, nil
	}
	var failure *apperror.Error
	if errors.As(err, &failure) {
		return true, err
	}
	if err != nil {
		return true, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos validar la comanda.")
	}
	// Cada plato avanza un paso a la vez: «Iniciar» empieza los que esperan (aunque
	// otro plato ya esté en marcha) y «Marcar listo» exige que todos hayan empezado.
	confirmed, preparing, err := a.repo.StationPlateCounts(tx.Context(), s, orderID, id, destination)
	if err != nil {
		return true, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos validar la comanda.")
	}
	if status == "listo" && confirmed > 0 {
		return true, orderFailure(apperror.Conflict, "kitchen_plates_not_started", "Hay platos sin iniciar. Inícialos antes de marcar la comanda lista.")
	}
	if (status == "preparando" && confirmed == 0) || (status == "listo" && preparing == 0) {
		return true, orderFailure(apperror.Conflict, "invalid_kitchen_transition", "La comanda cambió de estado. Actualiza antes de continuar.")
	}
	_, err = a.repo.UpdateStationItems(tx.Context(), s, orderID, id, destination, status)
	if err == nil {
		err = a.syncStation(tx.Context(), s, orderID, id)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return true, orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos guardar la preparación.")
	}
	if current == status {
		// Iniciar el resto de una comanda ya en marcha: no hay aviso ni oferta nuevos.
		_ = a.audit.Record(ctx, s, "preparation."+status, "order", orderID)
		return true, nil
	}
	a.stationChanged(ctx, s, orderID, id, destination, status)
	return true, nil
}

// UpdateKitchenItem avanza un plato por sí solo: lo empieza, lo marca listo o lo
// devuelve a preparación mientras la comanda sigue abierta, sin saltar estados.
// El primer plato empezado inicia la comanda y el último listo la termina, con el
// mismo efecto que «Iniciar» y «Marcar listo» sobre toda la comanda.
func (a *Orders) UpdateKitchenItem(ctx context.Context, s scope, rawID, itemID string, in OrderStatusInput) error {
	in.Status = strings.TrimSpace(in.Status)
	itemID = strings.TrimSpace(itemID)
	if in.Status != "listo" && in.Status != "preparando" {
		return orderFailure(apperror.Invalid, "invalid_kitchen_status", "Cocina solo puede empezar un plato, marcarlo listo o devolverlo a preparación.")
	}
	if itemID == "" {
		return orderFailure(apperror.NotFound, "kitchen_item_not_found", "El plato no existe en esta comanda.")
	}
	id, destination := stationTicket(rawID)
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos actualizar el plato.")
	}
	defer tx.Rollback(ctx)
	orderID, before, err := a.lockStation(tx.Context(), s, id, destination)
	if errors.Is(err, ErrNotFound) {
		return orderFailure(apperror.NotFound, "kitchen_ticket_not_found", "La comanda no existe en esta estación.")
	}
	var failure *apperror.Error
	if errors.As(err, &failure) {
		return err
	}
	if err != nil {
		return orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos validar la comanda.")
	}
	// Una comanda completa ya avisó que está lista: deshacer un plato ahí reabriría
	// un aviso entregado, así que solo se corrige mientras la comanda sigue abierta.
	if in.Status == "preparando" && before == "listo" {
		return orderFailure(apperror.Conflict, "kitchen_ticket_ready", "La comanda ya está lista; no se puede devolver un plato a preparación.")
	}
	tag, err := a.repo.UpdateStationItem(tx.Context(), s, orderID, id, destination, itemID, in.Status)
	if err != nil {
		return orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos guardar el plato.")
	}
	if tag == 0 {
		return orderFailure(apperror.Conflict, "kitchen_item_unavailable", "El plato no está en esta comanda o ya cambió de estado. Actualiza antes de continuar.")
	}
	after, err := a.repo.StationStatus(tx.Context(), s, orderID, id, destination)
	if err == nil {
		err = a.syncStation(tx.Context(), s, orderID, id)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return orderFailure(apperror.Unavailable, "kitchen_unavailable", "No pudimos guardar el plato.")
	}
	_ = a.audit.Record(ctx, s, "preparation.item."+in.Status, "order", orderID)
	if after != before && (after == "listo" || after == "preparando") {
		a.stationChanged(ctx, s, orderID, id, destination, after)
	}
	return nil
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
			a.noticeReady(ctx, s, roundOrderID, id, "kitchen")
		} else {
			a.dispatchChanged(ctx, s, roundOrderID)
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
		a.noticeReady(ctx, s, id, id, "kitchen")
	} else {
		a.dispatchChanged(ctx, s, id)
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
