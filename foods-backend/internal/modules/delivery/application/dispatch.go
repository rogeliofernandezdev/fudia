package application

import (
	"context"
	"errors"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	noticecontracts "github.com/foods-platform/foods-backend/internal/modules/notifications/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

// OrderFacts es lo que el despacho necesita saber del pedido y de su local.
type OrderFacts struct {
	ID, Code, Channel, Status, CustomerName string
	Completed                               bool
	HasPayments                             bool
	// Mode es el modo de delivery del local (domain.ModeOwn si no se configuró).
	Mode string
}

// DispatchState es el despacho bloqueado dentro de la transacción.
type DispatchState struct {
	ID, OrderID, Status string
	Round               int
	ReadyNotified       bool
}

// OfferState es la oferta a la que pertenece el token de un botón, con su
// despacho bloqueado y el alcance del pedido.
type OfferState struct {
	OrganizationID, LocationID           string
	OfferID, DispatchID, OrderID         string
	PartnerID, PartnerPhone              string
	OfferStatus, DispatchStatus          string
	OrderStatus                          string
	Expired                              bool
	CurrentRound, Assigned, Eligible     bool
	HasPayments, Completed               bool
	OrderCode, OrganizationName, LocName string
}

// OrderRef identifica un pedido que el barrido debe revisar.
type OrderRef struct{ OrganizationID, LocationID, OrderID string }

// DispatchRepository persiste despachos, ofertas y la bandeja de mensajes.
type DispatchRepository interface {
	OrderFacts(ctx context.Context, s scope, orderID string) (OrderFacts, error)
	LockDispatch(ctx context.Context, s scope, orderID string) (DispatchState, bool, error)
	// CreateDispatch inserta el despacho; created=false si otro proceso ya lo creó.
	CreateDispatch(ctx context.Context, s scope, orderID, status string) (DispatchState, bool, error)
	SetDispatchStatus(ctx context.Context, s scope, dispatchID, status string) error
	StartRound(ctx context.Context, s scope, dispatchID string, round int) error
	HasActivePartners(ctx context.Context, s scope) (bool, error)
	// NextPartner elige la central activa de mayor prioridad aún no ofrecida en la vuelta.
	NextPartner(ctx context.Context, s scope, dispatchID string, round int) (partnerID string, priority int, ok bool, err error)
	// CreateOffer registra la oferta y su mensaje en la bandeja de salida.
	CreateOffer(ctx context.Context, s scope, dispatchID, partnerID string, round, priority int) error
	ExpireOverdueOffers(ctx context.Context, s scope, dispatchID string) error
	HasPendingOffer(ctx context.Context, s scope, dispatchID string) (bool, error)
	WithdrawPendingOffers(ctx context.Context, s scope, dispatchID string) error
	RespondOffer(ctx context.Context, s scope, offerID, status string) error
	Accept(ctx context.Context, s scope, dispatchID, offerID string) error
	AcceptedOffer(ctx context.Context, s scope, dispatchID string) (string, bool, error)
	EnqueueMessage(ctx context.Context, s scope, dispatchID, offerID, kind string) error
	MarkReadyNotified(ctx context.Context, s scope, dispatchID string) error
	MarkPickedUp(ctx context.Context, s scope, dispatchID string) error
	FeeState(ctx context.Context, s scope, orderID string) (FeeState, bool, error)
	MarkFeeReleased(ctx context.Context, s scope, dispatchID string, released bool) error
	Dispatch(ctx context.Context, s scope, orderID string) (domain.Dispatch, error)
	// Integración con Concierge.
	DueOrders(ctx context.Context, limit int) ([]OrderRef, error)
	ClaimMessages(ctx context.Context, limit int) ([]domain.PartnerMessage, error)
	MarkMessageDispatched(ctx context.Context, id string) error
	LockOfferByToken(ctx context.Context, token string) (OfferState, error)
}

// OrderHandoff conecta el despacho con Orders (la composición lo implementa
// con sus casos de uso): «en camino» al recoger y el envío que pasa a la
// central al aceptar (y vuelve si el local usa su delivery propio).
type OrderHandoff interface {
	MarkOnTheWay(ctx context.Context, s scope, orderID string) error
	ReleaseDeliveryFee(ctx context.Context, s scope, orderID string) error
	RestoreDeliveryFee(ctx context.Context, s scope, orderID string, fee float64) error
}

// FeeState indica dónde está el costo de envío de un despacho.
type FeeState struct {
	DispatchID, Status string
	PartnerFee         float64
	Released           bool
}

// Dispatcher ofrece los pedidos de delivery a las centrales y avanza la cascada.
type Dispatcher struct {
	repo    DispatchRepository
	work    unitofwork.Manager
	notices noticecontracts.Publisher
	handoff OrderHandoff
}

func NewDispatcher(repo DispatchRepository, work unitofwork.Manager) *Dispatcher {
	return &Dispatcher{repo: repo, work: work}
}

// WithNotices avisa a caja cuando ninguna central acepta.
func (a *Dispatcher) WithNotices(notices noticecontracts.Publisher) *Dispatcher {
	a.notices = notices
	return a
}

// WithHandoff conecta el paso a «en camino» de Orders.
func (a *Dispatcher) WithHandoff(handoff OrderHandoff) *Dispatcher {
	a.handoff = handoff
	return a
}

// ClaimLimit acota cuántos mensajes reclama Concierge por consulta.
const ClaimLimit = 50

// SweepLimit acota cuántos pedidos revisa un barrido.
const SweepLimit = 100

var errDispatchNotFound = failure(apperror.NotFound, "delivery_dispatch_not_found", "Este pedido no tiene despacho a centrales.")

// OrderChanged implementa contracts.OrderDispatch: Orders lo llama después de
// confirmar, cancelar o terminar de preparar un pedido. Es de mejor esfuerzo; el
// barrido periódico corrige lo que falle aquí.
func (a *Dispatcher) OrderChanged(ctx context.Context, s scope, orderID string) {
	if a == nil || s.OrganizationID == "" || s.LocationID == "" || !uuidPattern.MatchString(orderID) {
		return
	}
	_ = a.syncOrder(ctx, s, orderID)
}

// Sweep revisa los pedidos con algo pendiente: despachos por crear, ofertas
// vencidas, «listo para recoger» por avisar y pedidos cancelados.
func (a *Dispatcher) Sweep(ctx context.Context) error {
	refs, err := a.repo.DueOrders(ctx, SweepLimit)
	if err != nil {
		return err
	}
	for _, ref := range refs {
		_ = a.syncOrder(ctx, scope{OrganizationID: ref.OrganizationID, LocationID: ref.LocationID}, ref.OrderID)
	}
	return nil
}

func (a *Dispatcher) syncOrder(ctx context.Context, s scope, orderID string) error {
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	notice, err := a.sync(tx.Context(), s, orderID)
	if err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	a.publish(ctx, s, notice)
	return nil
}

// settleFee deja el costo de envío donde corresponde dentro de la transacción: con
// la central si aceptó o recogió, y de vuelta en el pedido si el local dejó de
// usar centrales. Un fallo revierte el cambio de despacho y de dinero juntos.
func (a *Dispatcher) settleFee(ctx context.Context, s scope, orderID string) error {
	if a.handoff == nil {
		return errors.New("order handoff is not configured")
	}
	state, found, err := a.repo.FeeState(ctx, s, orderID)
	if err != nil || !found {
		return err
	}
	withPartner := state.Status == domain.DispatchAccepted || state.Status == domain.DispatchPickedUp
	switch {
	case withPartner && !state.Released && state.PartnerFee > 0:
		if err := a.handoff.ReleaseDeliveryFee(ctx, s, orderID); err != nil {
			return err
		}
		return a.repo.MarkFeeReleased(ctx, s, state.DispatchID, true)
	case !withPartner && state.Released:
		if state.Status == domain.DispatchCancelled {
			return nil
		}
		if err := a.handoff.RestoreDeliveryFee(ctx, s, orderID, state.PartnerFee); err != nil {
			return err
		}
		return a.repo.MarkFeeReleased(ctx, s, state.DispatchID, false)
	}
	return nil
}

func (a *Dispatcher) publish(ctx context.Context, s scope, notice *noticecontracts.Notice) {
	if notice != nil && a.notices != nil {
		_ = a.notices.Publish(ctx, s, *notice)
	}
}

// sync aplica, dentro de la transacción, lo que corresponde al estado actual
// del pedido. Devuelve el aviso a caja si la cascada se agotó.
func (a *Dispatcher) sync(ctx context.Context, s scope, orderID string) (*noticecontracts.Notice, error) {
	facts, err := a.repo.OrderFacts(ctx, s, orderID)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d, found, err := a.repo.LockDispatch(ctx, s, orderID)
	if err != nil {
		return nil, err
	}
	if !found {
		if facts.Mode != domain.ModeFudia || !domain.OrderDispatchable(facts.Channel, facts.Status, facts.Completed) {
			return nil, nil
		}
		var created bool
		if d, created, err = a.repo.CreateDispatch(ctx, s, orderID, domain.DispatchSearching); err != nil || !created {
			return nil, err
		}
		return a.offerNext(ctx, s, facts, d)
	}
	if !domain.ActiveDispatch(d.Status) {
		return nil, nil
	}
	if facts.Status == "cancelado" {
		return nil, a.stop(ctx, s, d, domain.DispatchCancelled)
	}
	// El local resolvió el pedido sin la central (lo despachó o lo cobró y cerró).
	if d.Status != domain.DispatchAccepted && (facts.Completed || facts.Status == "en_camino" || facts.Status == "entregado") {
		return nil, a.stop(ctx, s, d, domain.DispatchOwn)
	}
	switch d.Status {
	case domain.DispatchSearching:
		if facts.HasPayments {
			return nil, a.stop(ctx, s, d, domain.DispatchOwn)
		}
		if err = a.repo.ExpireOverdueOffers(ctx, s, d.ID); err != nil {
			return nil, err
		}
		pending, err := a.repo.HasPendingOffer(ctx, s, d.ID)
		if err != nil || pending {
			return nil, err
		}
		return a.offerNext(ctx, s, facts, d)
	case domain.DispatchAccepted:
		if err := a.settleFee(ctx, s, orderID); err != nil {
			return nil, err
		}
		return nil, a.notifyReady(ctx, s, d, facts.Status)
	}
	return nil, nil
}

// offerNext ofrece el pedido a la siguiente central de la vuelta o, si no
// queda ninguna, lo deja sin asignar y prepara el aviso a caja.
func (a *Dispatcher) offerNext(ctx context.Context, s scope, facts OrderFacts, d DispatchState) (*noticecontracts.Notice, error) {
	partnerID, priority, ok, err := a.repo.NextPartner(ctx, s, d.ID, d.Round)
	if err != nil {
		return nil, err
	}
	if ok {
		if err = a.repo.CreateOffer(ctx, s, d.ID, partnerID, d.Round, priority); err != nil {
			return nil, err
		}
		return nil, a.repo.SetDispatchStatus(ctx, s, d.ID, domain.DispatchSearching)
	}
	if err = a.repo.SetDispatchStatus(ctx, s, d.ID, domain.DispatchUnassigned); err != nil {
		return nil, err
	}
	notice := noticecontracts.DeliveryUnassigned(facts.ID, facts.Code, facts.CustomerName, d.Round)
	return &notice, nil
}

// notifyReady envía «listo para recoger» a la central que aceptó, una sola vez.
func (a *Dispatcher) notifyReady(ctx context.Context, s scope, d DispatchState, orderStatus string) error {
	if d.ReadyNotified || orderStatus != "listo" {
		return nil
	}
	offerID, ok, err := a.repo.AcceptedOffer(ctx, s, d.ID)
	if err != nil || !ok {
		return err
	}
	if err = a.repo.EnqueueMessage(ctx, s, d.ID, offerID, domain.MessageReady); err != nil {
		return err
	}
	return a.repo.MarkReadyNotified(ctx, s, d.ID)
}

// stop cierra el despacho: retira las ofertas pendientes y, si una central ya
// había aceptado, le avisa que no recoja el pedido.
func (a *Dispatcher) stop(ctx context.Context, s scope, d DispatchState, status string) error {
	if err := a.repo.WithdrawPendingOffers(ctx, s, d.ID); err != nil {
		return err
	}
	if d.Status == domain.DispatchAccepted {
		offerID, ok, err := a.repo.AcceptedOffer(ctx, s, d.ID)
		if err != nil {
			return err
		}
		if ok {
			if err = a.repo.EnqueueMessage(ctx, s, d.ID, offerID, domain.MessageCancelled); err != nil {
				return err
			}
		}
	}
	if err := a.repo.SetDispatchStatus(ctx, s, d.ID, status); err != nil {
		return err
	}
	return a.settleFee(ctx, s, d.OrderID)
}

// Dispatch devuelve el despacho del pedido con el historial de ofertas.
func (a *Dispatcher) Dispatch(ctx context.Context, s scope, orderID string) (domain.Dispatch, error) {
	if !uuidPattern.MatchString(orderID) {
		return domain.Dispatch{}, errDispatchNotFound
	}
	out, err := a.repo.Dispatch(ctx, s, orderID)
	if errors.Is(err, ErrNotFound) {
		return out, errDispatchNotFound
	}
	if err != nil {
		return out, unavailable("delivery_dispatch_unavailable", "No pudimos cargar el despacho del pedido.", err)
	}
	return out, nil
}

// Retry vuelve a ofrecer el pedido desde la 1.ª central (nueva vuelta). Sirve
// cuando ninguna aceptó o cuando el local había elegido su delivery propio.
func (a *Dispatcher) Retry(ctx context.Context, s scope, orderID string) (domain.Dispatch, error) {
	if !uuidPattern.MatchString(orderID) {
		return domain.Dispatch{}, errDispatchNotFound
	}
	fail := func(err error) (domain.Dispatch, error) {
		return domain.Dispatch{}, unavailable("delivery_dispatch_unavailable", "No pudimos volver a ofrecer el pedido.", err)
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return fail(err)
	}
	defer tx.Rollback(ctx)
	tctx := tx.Context()
	facts, err := a.repo.OrderFacts(tctx, s, orderID)
	if errors.Is(err, ErrNotFound) {
		return domain.Dispatch{}, failure(apperror.NotFound, "order_not_found", "El pedido no existe.")
	}
	if err != nil {
		return fail(err)
	}
	if !domain.OrderDispatchable(facts.Channel, facts.Status, facts.Completed) {
		return domain.Dispatch{}, failure(apperror.Conflict, "delivery_not_dispatchable", "Solo se ofrecen a una central los pedidos de delivery confirmados que aún no salen.")
	}
	if facts.HasPayments {
		return domain.Dispatch{}, failure(apperror.Conflict, "delivery_payment_conflict", "Este pedido ya tiene pagos registrados. Coordina el reparto con el local.")
	}
	active, err := a.repo.HasActivePartners(tctx, s)
	if err != nil {
		return fail(err)
	}
	if !active {
		return domain.Dispatch{}, failure(apperror.Conflict, "delivery_partners_missing", "No hay centrales activas con cobertura para el distrito del local.")
	}
	d, found, err := a.repo.LockDispatch(tctx, s, orderID)
	if err != nil {
		return fail(err)
	}
	if !found {
		var created bool
		if d, created, err = a.repo.CreateDispatch(tctx, s, orderID, domain.DispatchSearching); err != nil {
			return fail(err)
		}
		if !created {
			return domain.Dispatch{}, failure(apperror.Conflict, "delivery_dispatch_busy", "El pedido se está ofreciendo en este momento. Vuelve a intentarlo.")
		}
	} else {
		if !domain.Retryable(d.Status) {
			return domain.Dispatch{}, failure(apperror.Conflict, "delivery_dispatch_in_progress", "El pedido ya está ofrecido o aceptado por una central.")
		}
		d.Round++
		if err = a.repo.StartRound(tctx, s, d.ID, d.Round); err != nil {
			return fail(err)
		}
	}
	notice, err := a.offerNext(tctx, s, facts, d)
	if err != nil {
		return fail(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fail(err)
	}
	a.publish(ctx, s, notice)
	return a.Dispatch(ctx, s, orderID)
}

// UseOwn deja el pedido para el delivery propio del local: retira las ofertas
// y avisa a la central que ya había aceptado.
func (a *Dispatcher) UseOwn(ctx context.Context, s scope, orderID string) (domain.Dispatch, error) {
	if !uuidPattern.MatchString(orderID) {
		return domain.Dispatch{}, errDispatchNotFound
	}
	fail := func(err error) (domain.Dispatch, error) {
		return domain.Dispatch{}, unavailable("delivery_dispatch_unavailable", "No pudimos cambiar a delivery propio.", err)
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return fail(err)
	}
	defer tx.Rollback(ctx)
	tctx := tx.Context()
	facts, err := a.repo.OrderFacts(tctx, s, orderID)
	if errors.Is(err, ErrNotFound) {
		return domain.Dispatch{}, failure(apperror.NotFound, "order_not_found", "El pedido no existe.")
	}
	if err != nil {
		return fail(err)
	}
	if facts.Channel != "delivery" {
		return domain.Dispatch{}, failure(apperror.Conflict, "delivery_not_dispatchable", "Solo los pedidos de delivery usan centrales.")
	}
	if facts.Completed || facts.Status == "cancelado" {
		return domain.Dispatch{}, failure(apperror.Conflict, "order_closed", "El pedido ya está cerrado.")
	}
	d, found, err := a.repo.LockDispatch(tctx, s, orderID)
	if err != nil {
		return fail(err)
	}
	switch {
	case !found:
		// Evita que el barrido lo ofrezca después a una central.
		if _, _, err = a.repo.CreateDispatch(tctx, s, orderID, domain.DispatchOwn); err != nil {
			return fail(err)
		}
	case d.Status == domain.DispatchOwn:
	case d.Status == domain.DispatchPickedUp:
		return domain.Dispatch{}, failure(apperror.Conflict, "delivery_already_picked_up", "La central ya recogió este pedido.")
	case d.Status == domain.DispatchCancelled:
		return domain.Dispatch{}, failure(apperror.Conflict, "order_closed", "El pedido está cancelado.")
	case d.Status == domain.DispatchAccepted && facts.HasPayments:
		return domain.Dispatch{}, failure(apperror.Conflict, "delivery_payment_conflict", "Devuelve primero los pagos registrados antes de cambiar a delivery propio.")
	default:
		if err = a.stop(tctx, s, d, domain.DispatchOwn); err != nil {
			return fail(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return fail(err)
	}
	return a.Dispatch(ctx, s, orderID)
}

// Claim entrega a Concierge los mensajes pendientes para las centrales. Antes
// avanza la cascada (ofertas vencidas, pedidos listos o cancelados).
func (a *Dispatcher) Claim(ctx context.Context, limit int) ([]domain.PartnerMessage, error) {
	if limit < 1 || limit > ClaimLimit {
		limit = 20
	}
	_ = a.Sweep(ctx)
	out, err := a.repo.ClaimMessages(ctx, limit)
	if err != nil {
		return nil, unavailable("delivery_messages_unavailable", "No pudimos cargar los mensajes para las centrales.", err)
	}
	return out, nil
}

// MarkDispatched confirma que Concierge envió el mensaje; desde ese momento
// corren los 3 minutos de la oferta.
func (a *Dispatcher) MarkDispatched(ctx context.Context, id string) error {
	if !uuidPattern.MatchString(id) {
		return failure(apperror.NotFound, "delivery_message_not_found", "El mensaje no existe.")
	}
	err := a.repo.MarkMessageDispatched(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return failure(apperror.NotFound, "delivery_message_not_found", "El mensaje no existe.")
	}
	if err != nil {
		return unavailable("delivery_messages_unavailable", "No pudimos confirmar el mensaje.", err)
	}
	return nil
}

// Act aplica el botón que pulsó la central (Aceptar, Rechazar o Recoger).
func (a *Dispatcher) Act(ctx context.Context, in domain.ActionInput) (domain.ActionResult, error) {
	if !domain.ValidAction(in.Action) {
		return domain.ActionResult{}, failure(apperror.Invalid, "invalid_delivery_action", "La acción no es válida.")
	}
	if !uuidPattern.MatchString(in.Token) {
		return domain.ActionResult{Outcome: domain.OutcomeNotFound}, nil
	}
	fail := func(err error) (domain.ActionResult, error) {
		return domain.ActionResult{}, unavailable("delivery_action_unavailable", "No pudimos registrar la respuesta de la central.", err)
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return fail(err)
	}
	defer tx.Rollback(ctx)
	tctx := tx.Context()
	st, err := a.repo.LockOfferByToken(tctx, in.Token)
	if errors.Is(err, ErrNotFound) {
		return domain.ActionResult{Outcome: domain.OutcomeNotFound}, nil
	}
	if err != nil {
		return fail(err)
	}
	// Solo el WhatsApp de la central ofertada puede responder.
	if phone := domain.NormalizePhone(in.Phone); phone == "" || phone != st.PartnerPhone {
		return domain.ActionResult{Outcome: domain.OutcomeNotFound}, nil
	}
	s := scope{OrganizationID: st.OrganizationID, LocationID: st.LocationID}
	result := domain.ActionResult{OrderCode: st.OrderCode, OrganizationName: st.OrganizationName, LocationName: st.LocName}
	var notice *noticecontracts.Notice
	pickup := false
	switch in.Action {
	case domain.ActionAccept:
		result.Outcome, notice, err = a.accept(tctx, s, st)
	case domain.ActionReject:
		result.Outcome, notice, err = a.reject(tctx, s, st)
	case domain.ActionPickup:
		result.Outcome, pickup, err = a.checkPickup(tctx, s, st)
	}
	if err != nil {
		return fail(err)
	}
	if pickup {
		if a.handoff == nil {
			return fail(errors.New("order handoff is not configured"))
		}
		if err = a.handoff.MarkOnTheWay(tctx, s, st.OrderID); err != nil {
			return domain.ActionResult{}, err
		}
		if err = a.repo.MarkPickedUp(tctx, s, st.DispatchID); err != nil {
			return fail(err)
		}
		result.Outcome = domain.OutcomePickedUp
	}
	if err = tx.Commit(ctx); err != nil {
		return fail(err)
	}
	a.publish(ctx, s, notice)
	return result, nil
}

func (a *Dispatcher) accept(ctx context.Context, s scope, st OfferState) (string, *noticecontracts.Notice, error) {
	switch {
	case st.DispatchStatus == domain.DispatchCancelled:
		return domain.OutcomeCancelled, nil, nil
	case !st.CurrentRound:
		return domain.OutcomeTaken, nil, nil
	case st.OfferStatus == domain.OfferAccepted && st.Assigned && (st.DispatchStatus == domain.DispatchAccepted || st.DispatchStatus == domain.DispatchPickedUp):
		return domain.OutcomeAlreadyAccepted, nil, nil
	case st.OfferStatus == domain.OfferExpired:
		return domain.OutcomeExpired, nil, nil
	case st.OfferStatus != domain.OfferPending || st.DispatchStatus != domain.DispatchSearching:
		return domain.OutcomeTaken, nil, nil
	}
	if !st.Eligible || st.Completed || st.OrderStatus == "cancelado" {
		if err := a.repo.RespondOffer(ctx, s, st.OfferID, domain.OfferWithdrawn); err != nil {
			return "", nil, err
		}
		notice, err := a.advance(ctx, s, st)
		return domain.OutcomeTaken, notice, err
	}
	if st.HasPayments {
		d := DispatchState{ID: st.DispatchID, OrderID: st.OrderID, Status: st.DispatchStatus}
		return domain.OutcomePaymentConflict, nil, a.stop(ctx, s, d, domain.DispatchOwn)
	}
	if st.Expired {
		if err := a.repo.RespondOffer(ctx, s, st.OfferID, domain.OfferExpired); err != nil {
			return "", nil, err
		}
		notice, err := a.advance(ctx, s, st)
		return domain.OutcomeExpired, notice, err
	}
	if err := a.repo.Accept(ctx, s, st.DispatchID, st.OfferID); err != nil {
		return "", nil, err
	}
	if err := a.settleFee(ctx, s, st.OrderID); err != nil {
		return "", nil, err
	}
	// Si cocina ya terminó, la central recibe de inmediato «listo para recoger».
	d := DispatchState{ID: st.DispatchID, OrderID: st.OrderID, Status: domain.DispatchAccepted}
	return domain.OutcomeAccepted, nil, a.notifyReady(ctx, s, d, st.OrderStatus)
}

func (a *Dispatcher) reject(ctx context.Context, s scope, st OfferState) (string, *noticecontracts.Notice, error) {
	switch {
	case st.DispatchStatus == domain.DispatchCancelled:
		return domain.OutcomeCancelled, nil, nil
	case !st.CurrentRound:
		return domain.OutcomeTaken, nil, nil
	case st.OfferStatus == domain.OfferAccepted && st.Assigned && (st.DispatchStatus == domain.DispatchAccepted || st.DispatchStatus == domain.DispatchPickedUp):
		return domain.OutcomeAlreadyAccepted, nil, nil
	case st.OfferStatus != domain.OfferPending || st.DispatchStatus != domain.DispatchSearching:
		return domain.OutcomeTaken, nil, nil
	}
	if err := a.repo.RespondOffer(ctx, s, st.OfferID, domain.OfferRejected); err != nil {
		return "", nil, err
	}
	notice, err := a.advance(ctx, s, st)
	return domain.OutcomeRejected, notice, err
}

// advance pasa a la siguiente central después de un rechazo o un vencimiento.
func (a *Dispatcher) advance(ctx context.Context, s scope, st OfferState) (*noticecontracts.Notice, error) {
	facts, err := a.repo.OrderFacts(ctx, s, st.OrderID)
	if err != nil {
		return nil, err
	}
	d, found, err := a.repo.LockDispatch(ctx, s, st.OrderID)
	if err != nil || !found {
		return nil, err
	}
	if facts.Status == "cancelado" {
		return nil, a.stop(ctx, s, d, domain.DispatchCancelled)
	}
	if facts.HasPayments || !domain.OrderDispatchable(facts.Channel, facts.Status, facts.Completed) {
		return nil, a.stop(ctx, s, d, domain.DispatchOwn)
	}
	pending, err := a.repo.HasPendingOffer(ctx, s, d.ID)
	if err != nil || pending {
		return nil, err
	}
	return a.offerNext(ctx, s, facts, d)
}

// checkPickup valida «Recoger»; true indica que hay que pasar el pedido a «en camino».
func (a *Dispatcher) checkPickup(ctx context.Context, s scope, st OfferState) (string, bool, error) {
	switch {
	case st.DispatchStatus == domain.DispatchCancelled:
		return domain.OutcomeCancelled, false, nil
	case st.OfferStatus != domain.OfferAccepted:
		return domain.OutcomeTaken, false, nil
	case !st.CurrentRound || !st.Assigned:
		return domain.OutcomeTaken, false, nil
	case st.DispatchStatus == domain.DispatchPickedUp:
		return domain.OutcomePickedUp, false, nil
	case st.DispatchStatus != domain.DispatchAccepted:
		return domain.OutcomeTaken, false, nil
	}
	switch st.OrderStatus {
	case "listo":
		return "", true, nil
	case "en_camino", "entregado":
		// El local ya lo despachó: se registra la recogida sin repetir la transición.
		return domain.OutcomePickedUp, false, a.repo.MarkPickedUp(ctx, s, st.DispatchID)
	}
	return domain.OutcomeNotReady, false, nil
}
