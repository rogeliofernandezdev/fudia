from __future__ import annotations

import asyncio
import logging
import re
import time
from collections.abc import Awaitable, Callable, Coroutine
from decimal import Decimal
from typing import Any, Literal, cast
from uuid import uuid4

from src.ai.contracts import ConciergeRouter, ConversationAgent
from src.ai.tiering import READ_ONLY_TOOLS
from src.domain.intents import AgentIntent, ConciergeRoute
from src.domain.models import (
    ChatMessage,
    ConsentDecision,
    ConversationSession,
    CustomerProfile,
    DeliveryDetails,
    EffectRecord,
    GeoPoint,
    InboundMessage,
    ProcessingRecord,
    ReplyImage,
    TableContext,
    VenueContext,
)
from src.domain.whatsapp import RestaurantOption, RestaurantSelection
from src.graph.app import build_graph
from src.metrics import (
    CUSTOMER_NOTICES,
    DELIVERY_PARTNER_ACTIONS,
    LOCK_LOST,
    MESSAGES_SUPERSEDED,
    PROCESSING_RECOVERIES,
    RATE_LIMITED,
    SCOPE_DECISIONS,
)
from src.observability import fingerprint, log_event
from src.services.carta import Carta, build_carta
from src.services.confirmation import checkout_requested
from src.services.consent import (
    asks_to_withdraw,
    consent_answer,
    consent_question,
    consent_reply,
)
from src.services.contact_name import CUSTOMER_NAME_PROMPT, customer_name_reply, full_customer_name
from src.services.contact_phone import contact_phone_prompt, normalize_contact_phone, phone_text
from src.services.customer_notices import CUSTOMER_WINDOW_SECONDS, notice_text
from src.services.delivery_partners import action_reply
from src.services.delivery_tools import apply_quote, delivery_summary
from src.services.formatting import whatsapp_text, without_reference_quotes
from src.services.menu_tools import unavailable_carta_reply
from src.services.ports import (
    CONVERSATION_LIMIT_CODE,
    STALE_TABLE_ACCOUNT_CODE,
    AllowAllRateLimiter,
    ConversationLease,
    ConversationLock,
    ConversationStore,
    FudiaBackend,
    FudiaError,
    LocalConversationLock,
    LocalTableStartCodes,
    LockLostError,
    RateLimiter,
    TableStartCodes,
    conversation_identity,
    message_ref,
)
from src.services.processing import TurnJournal, recovery_reply
from src.services.restaurants import RestaurantDirectory, option_number, selection_text
from src.services.service_tools import bill_summary
from src.services.start_codes import (
    looks_like_start_message,
    start_code_candidates,
    table_label,
)
from src.services.tool_registry import ToolRegistry
from src.services.tooling import cart_payload
from src.services.turn_cache import TurnScopedFudia
from src.services.usage_meter import TokenMeter, current_meter, metering
from src.services.web_chat import action_reply as web_action_reply
from src.services.welcome import welcome_message

logger = logging.getLogger(__name__)

QR_PATTERN = re.compile(r"(?i)\bFUDIA:([a-f0-9]{32})\b")
ARITHMETIC_ONLY_PATTERN = re.compile(
    r"(?i)^\s*(?:cu[aá]nto\s+(?:es|da)\s+|calcula\s+|resuelve\s+)?"
    r"[0-9\s.,/()+\-*x×÷=]+\??\s*$"
)
MATH_TOPIC_PATTERN = re.compile(r"(?i)\b(fracciones?|ecuaciones?|ra[ií]z cuadrada|derivadas?|integrales?)\b")
OFF_SCOPE_REPLY = (
    "Puedo ayudarte únicamente con el pedido en este restaurante: consultar la "
    "carta, agregar o retirar productos, confirmar pedidos, delivery o recojo, "
    "pedir la cuenta o solicitar atención del personal."
)
SCAN_QR_REPLY = "Para atenderte en tu mesa, vuelve a escanear el QR y envía el mensaje tal como aparece."
EXPIRED_START_CODE_REPLY = (
    "Ese código de mesa ya venció. Vuelve a escanear el QR de tu mesa y envía el mensaje tal como aparece."
)
TOO_LONG_REPLY = "Tu mensaje es demasiado largo. Envíame solo lo necesario para gestionar tu pedido."
RATE_LIMITED_REPLY = (
    "Has enviado muchos mensajes en poco tiempo. Continúa con tu pedido dentro de un momento."
)
# foods-backend did not open a new conversation this month. The diner is
# invited to order in person; caps and plans are never mentioned.
STALE_TABLE_ACCOUNT_REPLY = (
    "Esta mesa tiene una cuenta anterior pendiente. Ya avisamos al personal; "
    "en un momento te atienden y podrás pedir por aquí."
)
CONVERSATION_LIMIT_TABLE_REPLY = (
    "En este momento no podemos tomar tu pedido por WhatsApp. "
    "Por favor, pide directamente al personal del restaurante en tu mesa."
)
CONVERSATION_LIMIT_DELIVERY_REPLY = (
    "En este momento no podemos tomar pedidos por WhatsApp. Te esperamos en el local para atenderte en mesa."
)


def conversation_limit_reply(provider: str, kind: Literal["table", "delivery"]) -> str:
    """Same refusal in the words of the channel the diner is using."""
    reply = CONVERSATION_LIMIT_DELIVERY_REPLY if kind == "delivery" else CONVERSATION_LIMIT_TABLE_REPLY
    return reply.replace("por WhatsApp", "por este chat") if provider == "FUDIA_WEB" else reply


# How a reply relates to the session: untouched, changed, or changed and
# recorded in the transcript the agents read.
SessionChange = Literal["none", "session", "transcript"]


class _LegacyRouter:
    async def route(
        self,
        session: ConversationSession,
        user_message: str,
    ) -> ConciergeRoute:
        return "order"


def conversation_id_for(message_id: str) -> str:
    """Id of the attention opened by this message. Deterministic, so a retry
    of the same message reopens (and counts) the same conversation."""
    return message_ref(f"conversation:{message_id}")[:32]


def profile_name(profile: CustomerProfile | None) -> str:
    """Name a returning diner gave before. The placeholder foods-backend uses
    for a number registered without a name («Cliente WhatsApp 1234») does not
    count: that diner is asked for it."""
    if profile is None or not profile.known:
        return ""
    name = " ".join(profile.customerName.split())
    return "" if name.lower().startswith("cliente whatsapp") else name


def extract_qr_token(text: str) -> str | None:
    match = QR_PATTERN.search(text)
    return match.group(1).lower() if match else None


def obviously_out_of_scope(text: str) -> bool:
    value = text.strip().strip("¿?¡!").strip()
    if not value:
        return False
    return bool(ARITHMETIC_ONLY_PATTERN.fullmatch(value) or MATH_TOPIC_PATTERN.search(value))


def _cart_state(session: ConversationSession) -> list[str]:
    """The round being built, as the agent needs it to resolve «otro igual»,
    «quita la causa» or «sin cebolla el segundo» without guessing."""
    cart = cart_payload(session)
    if not cart["items"]:
        return ["Carrito de esta ronda: vacío."]
    lines = ["Carrito de esta ronda (aún no enviado a cocina):"]
    for item in cart["items"]:
        quantity = item["quantity"]
        if isinstance(quantity, float) and quantity.is_integer():
            quantity = int(quantity)
        details = [f"{selection['groupName']}: {selection['name']}" for selection in item["selections"]] + [
            modifier["name"] for modifier in item["modifiers"]
        ]
        if item["note"]:
            details.append(f"nota: {item['note']}")
        extra = f" ({'; '.join(details)})" if details else ""
        lines.append(f"- {quantity} × {item['name']}{extra}: {item['lineTotal']}")
    lines.append(f"Total del carrito: {cart['total']}")
    return lines


def _round_state(session: ConversationSession) -> list[str]:
    lines = _cart_state(session)
    if session.awaiting_confirmation:
        lines.append(
            "Confirmación pendiente: ya se mostró el resumen; un «sí» del "
            "cliente lo envía y cualquier cambio exige mostrarlo de nuevo."
        )
    lines.append(
        "Pedidos ya enviados en esta atención: sí."
        if session.last_order_id
        else "Pedidos ya enviados en esta atención: ninguno."
    )
    return lines


def _attention_message(session: ConversationSession, user_message: str) -> str:
    """Every turn carries the current state of the attention (table or
    delivery, cart, pending confirmation) so the agent reads each message in
    context instead of in isolation."""
    if session.mode == "counter":
        table = session.table
        place = f" en {table.organizationName} - {table.locationName}" if table else ""
        name = session.delivery.customer_name
        lines = [
            "[Estado de la atención (no lo copies textual)]",
            f"Modo: venta por mostrador{place}. Cada pedido confirmado es "
            "independiente, se recoge en el mostrador y se paga en caja; no "
            "hay cuenta abierta ni delivery.",
            f"Nombre del cliente: {name or 'pendiente (pídelo antes de confirmar)'}.",
            *_round_state(session),
            "",
            "Mensaje del cliente:",
            user_message,
        ]
        return "\n".join(lines)
    if session.mode != "delivery":
        table = session.table
        if table is None:
            return user_message
        lines = [
            "[Estado de la atención (no lo copies textual)]",
            f"Modo: mesa abierta, {table_label(table.name)} en "
            f"{table.organizationName} - {table.locationName}.",
            *_round_state(session),
        ]
        if session.handoff_pending:
            lines.append("El personal del local ya fue avisado y está en camino.")
        lines += ["", "Mensaje del cliente:", user_message]
        return "\n".join(lines)
    summary = delivery_summary(session)
    venue = session.venue
    fulfillment = {
        "delivery": "delivery",
        "pickup": "recojo en el local",
    }.get(str(summary["fulfillment"]), "pendiente de elegir")
    location = "no compartida"
    if summary.get("locationShared"):
        location = (
            f"compartida, a {summary['distanceKm']} km; envío "
            f"{summary['currencySymbol']} {summary['deliveryFee']}"
            if summary.get("covered")
            else (f"fuera de cobertura ({summary['distanceKm']} km; máximo {summary['maxDistanceKm']} km)")
        )
    lines = [
        "[Estado de la atención (no lo copies textual)]",
        "Modo: pedido sin mesa (la atención en mesa terminó) en "
        f"{venue.organizationName} - {venue.locationName}."
        if venue
        else "Modo: pedido sin mesa (la atención en mesa terminó).",
        f"Tipo de entrega: {fulfillment}.",
        f"Nombre del cliente: {summary['customerName'] or 'pendiente'}.",
    ]
    if summary["fulfillment"] == "delivery":
        lines += [
            f"Ubicación: {location}.",
            f"Dirección escrita: {summary.get('address') or 'pendiente'}.",
            f"Referencia: {summary.get('reference') or 'sin referencia'}.",
        ]
    if summary["missing"]:
        lines.append("Falta: " + ", ".join(summary["missing"]) + ".")
    lines += _round_state(session)
    lines += ["", "Mensaje del cliente:", user_message]
    return "\n".join(lines)


def agent_message(session: ConversationSession, user_message: str) -> str:
    state = _attention_message(session, user_message)
    if session.provider == "FUDIA_WEB":
        return (
            "[Canal: chat web del restaurante en el navegador, no la aplicación WhatsApp. "
            "Tiene Carta y Mi pedido; para delivery puede usar Compartir ubicación. "
            "No indiques menús de la aplicación WhatsApp. "
            "En mesa y mostrador no se exige teléfono; para recojo o delivery solicita "
            "un celular de contacto si la herramienta lo requiere. En Perú acepta los 9 dígitos "
            "sin exigir +51; un número extranjero puede incluir su código de país. "
            f"Contacto: {'registrado, no volver a pedirlo' if session.contact_phone else 'pendiente'}.]\n"
            + state
        )
    return state


class ConciergeService:
    """Processing phase of an inbound message.

    `process_inbound` is idempotent per `message_id`: once a reply is prepared
    it returns the persisted record and never runs the model or a tool with
    effects again for that message. Sending the reply is a separate phase
    (`ReplyDispatcher`).
    """

    def __init__(
        self,
        store: ConversationStore,
        fudia: FudiaBackend,
        router: ConciergeRouter | ConversationAgent,
        agents: dict[AgentIntent, ConversationAgent] | None = None,
        conversation_lock: ConversationLock | None = None,
        rate_limiter: RateLimiter | None = None,
        start_codes: TableStartCodes | None = None,
        max_message_chars: int = 2000,
        lock_heartbeat_seconds: float = 5.0,
        lock_ttl_seconds: float = 30.0,
        clock: Callable[[], float] = time.time,
        table_idle_seconds: float = 10_800.0,
        customer_window_seconds: float = CUSTOMER_WINDOW_SECONDS,
        directory: RestaurantDirectory | None = None,
    ) -> None:
        self.directory = directory
        self.store = store
        self.customer_window_seconds = customer_window_seconds
        self.table_idle_seconds = table_idle_seconds
        self.fudia = fudia
        self.conversation_lock = conversation_lock or LocalConversationLock()
        self.rate_limiter = rate_limiter or AllowAllRateLimiter()
        self.start_codes = start_codes or LocalTableStartCodes()
        self.max_message_chars = max_message_chars
        self.lock_heartbeat_seconds = lock_heartbeat_seconds
        self.lock_ttl_seconds = lock_ttl_seconds
        self.clock = clock

        self.router: ConciergeRouter
        self.agents: dict[AgentIntent, ConversationAgent]

        if agents is None:
            legacy_agent = cast(
                ConversationAgent,
                router,
            )
            self.router = _LegacyRouter()
            self.agents = {
                "menu": legacy_agent,
                "order": legacy_agent,
                "service": legacy_agent,
            }
        else:
            self.router = cast(
                ConciergeRouter,
                router,
            )
            self.agents = agents

        processors: dict[
            AgentIntent,
            Callable[
                [ConversationSession, str, Any],
                Awaitable[str],
            ],
        ] = {intent: self._processor(intent) for intent in ("menu", "order", "service")}
        self.graph = build_graph(
            self.router.route,
            processors,
            OFF_SCOPE_REPLY,
        )

    def _processor(
        self,
        intent: AgentIntent,
    ) -> Callable[
        [ConversationSession, str, Any],
        Awaitable[str],
    ]:
        async def process(
            session: ConversationSession,
            user_message: str,
            journal: TurnJournal | None,
        ) -> str:
            # Tools must mutate the very session the journal checkpoints.
            if journal is not None:
                session = journal.session
            # Identical catalog reads within this agent turn reach the backend once.
            registry = ToolRegistry(
                TurnScopedFudia(self.fudia),
                session,
                user_message,
                journal,
            )
            fallback = ""
            action_attempted = False
            web_reply: str | None = None
            cart_changed = False

            async def execute(
                name: str,
                args: dict[str, Any],
            ) -> dict[str, Any]:
                nonlocal fallback, action_attempted, web_reply, cart_changed
                if (
                    session.provider == "FUDIA_WEB" and name == "prepare_confirmation"
                    and cart_changed and not checkout_requested(user_message)
                ):
                    return {
                        "ok": False, "code": "continue_ordering",
                        "message": "Espera la respuesta del cliente antes de preparar la confirmación.",
                    }
                if (
                    session.provider == "FUDIA_WEB" and name == "view_cart"
                    and checkout_requested(user_message)
                ):
                    name = "prepare_confirmation"
                result = await registry.execute_for(
                    intent,
                    name,
                    args,
                )
                if name == "search_menu":
                    fallback = unavailable_carta_reply(result) or fallback
                    if journal is not None and registry.menu.media:
                        journal.record.media = registry.menu.media
                elif name not in READ_ONLY_TOOLS:
                    action_attempted = True
                if result.get("ok") and (name.startswith("add_") or name == "remove_item"):
                    cart_changed = True
                if session.provider == "FUDIA_WEB" and name in {"prepare_confirmation", "confirm_order"}:
                    if result.get("code") != "continue_ordering":
                        web_reply = web_action_reply(name, result, session.currency_symbol)
                return result

            # Tools keep the diner's literal words (explicit confirmation);
            # only the agent also reads the current delivery state.
            reply = await self.agents[intent].reply(
                session,
                agent_message(session, user_message),
                execute,
            )
            if session.provider == "FUDIA_WEB":
                if cart_changed and not checkout_requested(user_message):
                    return "Actualicé tu pedido. ¿Deseas agregar algo más?"
                if web_reply is not None:
                    return web_reply
            if fallback:
                # Keep acknowledgements/errors for mixed requests with actions.
                # A read-only miss always returns the canonical, unabridged carta.
                if not action_attempted:
                    return fallback
                # Never truncate an action acknowledgement/confirmation to fit
                # the carta, or exceed WhatsApp's single-message limit.
                if fallback in reply or len(reply) + len(fallback) + 2 > 4096:
                    return reply
                return f"{reply}\n\n{fallback}"
            return reply

        return process

    async def handle_message(
        self,
        phone: str,
        text: str,
        channel_id: str = "",
    ) -> str:
        """Direct, unqueued processing (local tools and tests)."""
        record = await self.process_inbound(
            InboundMessage(
                message_id=f"direct-{uuid4().hex}",
                phone=phone,
                text=text,
                sender_phone_id=channel_id,
            )
        )
        return record.reply

    async def _inbound_record(self, message: InboundMessage) -> ProcessingRecord | None:
        record = await self.store.get_record(message.message_id)
        if record is None and message.legacy_message_id:
            record = await self.store.get_record(message.legacy_message_id)
        return record

    async def process_inbound(
        self,
        message: InboundMessage,
    ) -> ProcessingRecord:
        if not message.received_at:
            message = message.model_copy(update={"received_at": self.clock()})
        record = await self._inbound_record(message)
        if record is not None and record.prepared:
            return record

        channel_key = message.sender_phone_id.strip() or "default"
        identity = conversation_identity(message.phone, channel_key)
        if message.partner_action is not None:
            return await self._process_partner_action(message, identity, record)
        if message.notice is not None:
            return await self._process_notice(message, channel_key, identity)
        if len(message.text) > self.max_message_chars:
            return await self._prepare_unlocked(message, identity, record, TOO_LONG_REPLY)
        # Only the first attempt counts against the limit: a retry of the
        # same message is not new traffic from the diner.
        if record is None and not await self.rate_limiter.allow(identity):
            RATE_LIMITED.inc()
            return await self._prepare_unlocked(message, identity, record, RATE_LIMITED_REPLY)

        with metering() as meter:
            lease = ConversationLease(
                self.conversation_lock,
                identity,
                await self.conversation_lock.acquire(identity),
            )
            try:
                record = await self._guarded(
                    lease,
                    self._process_locked(message, channel_key, lease),
                )
            finally:
                await self.conversation_lock.release(identity, lease.token)
        await self._report_usage(meter, message, record)
        return record

    async def _process_partner_action(
        self,
        message: InboundMessage,
        identity: str,
        record: ProcessingRecord | None,
    ) -> ProcessingRecord:
        """Botón de una central de Delivery Fudia: sin modelo ni sesión de
        comensal. foods-backend aplica la acción (repetirla no duplica nada) y
        la respuesta se envía como cualquier otra. Un fallo del backend se
        propaga para que la cola reintente el mensaje."""
        action = message.partner_action
        assert action is not None
        verified = message.identity().verifiedPhone
        if not verified:
            return await self._prepare_unlocked(
                message, identity, record, "No pudimos verificar la identidad de la central de delivery."
            )
        result = await self.fudia.apply_delivery_action(action.action, action.token, verified)
        DELIVERY_PARTNER_ACTIONS.labels(action=action.action, outcome=result.outcome).inc()
        log_event(
            logger,
            "concierge.delivery_partner.action",
            message=fingerprint(message.message_id),
            action=action.action,
            outcome=result.outcome,
        )
        return await self._prepare_unlocked(message, identity, record, action_reply(result))

    async def _process_notice(
        self,
        message: InboundMessage,
        channel_key: str,
        identity: str,
    ) -> ProcessingRecord:
        """Aviso de foods-backend: sin modelo ni límite de frecuencia, bajo el
        mismo lock que los mensajes del cliente para no intercalarse con un turno."""
        lease = ConversationLease(
            self.conversation_lock,
            identity,
            await self.conversation_lock.acquire(identity),
        )
        try:
            return await self._guarded(lease, self._notice_locked(message, channel_key, lease))
        finally:
            await self.conversation_lock.release(identity, lease.token)

    async def _notice_locked(
        self,
        message: InboundMessage,
        channel_key: str,
        lease: ConversationLease,
    ) -> ProcessingRecord:
        record = await self._inbound_record(message)
        if record is not None and record.prepared:
            return record
        record = record or ProcessingRecord(message_id=message.message_id)
        record.conversation = lease.identity
        record.attempts += 1
        notice = message.notice
        session = await self.store.get(message.phone, channel_key)
        if (
            notice is None
            or session is None
            or (
                session.provider == "META_CLOUD_API"
                and (
                    not (session.last_customer_message_at or session.last_activity_at)
                    or self.clock() - (session.last_customer_message_at or session.last_activity_at)
                    > self.customer_window_seconds
                )
            )
        ):
            # Fuera de la ventana de 24 h de WhatsApp no se envía texto libre;
            # el personal ya ve el pedido en el panel.
            record.stage = "superseded"
            await self.store.commit(lease, None, record)
            CUSTOMER_NOTICES.labels(result="outside_window").inc()
            log_event(
                logger,
                "concierge.customer_notice.skipped",
                message=fingerprint(message.message_id),
                reason="outside_window",
            )
            return record
        reply = notice_text(notice)
        # El aviso queda en el historial para que el siguiente turno lo conozca.
        session.messages.append(ChatMessage(role="assistant", content=reply))
        session.messages = session.messages[-20:]
        record.reply = reply
        record.stage = "reply_ready"
        # Sin conversation_id: se avisa aunque el cliente ya esté en otro QR.
        record.conversation_id = ""
        await self.store.commit(lease, session, record)
        CUSTOMER_NOTICES.labels(result="prepared").inc()
        return record

    async def _report_usage(
        self,
        meter: TokenMeter,
        message: InboundMessage,
        record: ProcessingRecord,
    ) -> None:
        """Persist the turn's tokens per company in foods-backend. Each attempt
        of a message is its own request: a retried turn really spent again."""
        tiers = meter.tiers()
        if not tiers or not meter.qr_token or not meter.conversation_id:
            return
        request_id = f"{message_ref(message.message_id)[:32]}:{record.attempts}"
        try:
            await self.fudia.report_usage(meter.qr_token, meter.conversation_id, request_id, tiers)
        except Exception as exc:
            log_event(
                logger,
                "concierge.usage.report_failed",
                conversation=meter.conversation_id,
                errorType=type(exc).__name__,
                code=getattr(exc, "code", ""),
            )

    async def _open_conversation(
        self,
        token: str,
        conversation_id: str,
        kind: Literal["table", "delivery"],
        message: InboundMessage | None = None,
    ) -> Literal["open", "limit", "stale_account"]:
        """Count a new attention. foods-backend refuses it when the company
        reached its monthly cap ("limit") or, for a table, when the table has
        a previous account left open without new rounds ("stale_account"; the
        staff is already notified). Any other failure keeps serving the diner
        (the conversation is still counted by its first usage report)."""
        try:
            if self.directory is not None and message is not None:
                await self.fudia.open_conversation(
                    token,
                    conversation_id,
                    kind,
                    origin={
                        "channelKey": message.sender_phone_id,
                        "recipient": message.phone,
                        "identity": message.identity().model_dump(),
                    },
                )
            else:
                await self.fudia.open_conversation(token, conversation_id, kind)
        except FudiaError as exc:
            if exc.code == CONVERSATION_LIMIT_CODE:
                log_event(
                    logger,
                    "concierge.conversation.limit_reached",
                    conversation=conversation_id,
                    kind=kind,
                )
                return "limit"
            if exc.code == STALE_TABLE_ACCOUNT_CODE:
                log_event(
                    logger,
                    "concierge.table.stale_account",
                    conversation=conversation_id,
                )
                return "stale_account"
            log_event(
                logger,
                "concierge.conversation.open_failed",
                conversation=conversation_id,
                kind=kind,
                code=exc.code,
            )
            if self.directory is not None and message is not None:
                return "limit"
        return "open"

    async def _prepare_unlocked(
        self,
        message: InboundMessage,
        identity: str,
        record: ProcessingRecord | None,
        reply: str,
    ) -> ProcessingRecord:
        # Replies that neither read nor change the conversation.
        prepared = record or ProcessingRecord(message_id=message.message_id)
        prepared.conversation = identity
        prepared.attempts += 1
        prepared.reply = reply
        prepared.stage = "reply_ready"
        await self.store.put_record(prepared)
        return prepared

    async def _guarded(
        self,
        lease: ConversationLease,
        work: Coroutine[Any, Any, ProcessingRecord],
    ) -> ProcessingRecord:
        task = asyncio.create_task(work)
        heartbeat = asyncio.create_task(self._keep_lease(lease, task))
        try:
            return await task
        except asyncio.CancelledError:
            current = asyncio.current_task()
            if lease.lost and not (current and current.cancelling()):
                raise LockLostError("La conversación pasó a otro proceso.") from None
            raise
        finally:
            heartbeat.cancel()
            try:
                await heartbeat
            except (asyncio.CancelledError, Exception):
                pass

    async def _keep_lease(
        self,
        lease: ConversationLease,
        task: asyncio.Task[ProcessingRecord],
    ) -> None:
        loop = asyncio.get_running_loop()
        last_renewal = loop.time()
        while True:
            await asyncio.sleep(self.lock_heartbeat_seconds)
            try:
                refreshed = await self.conversation_lock.refresh(
                    lease.identity,
                    lease.token,
                )
            except asyncio.CancelledError:
                raise
            except Exception as exc:
                log_event(
                    logger,
                    "concierge.lock.refresh_failed",
                    conversation=fingerprint(lease.identity),
                    errorType=type(exc).__name__,
                )
                # A transient Redis error is tolerated while the lease is
                # certainly still valid; past that margin it may have expired.
                if loop.time() - last_renewal < self.lock_ttl_seconds * 0.8:
                    continue
                refreshed = False
            if refreshed:
                last_renewal = loop.time()
                continue
            lease.mark_lost()
            LOCK_LOST.inc()
            log_event(
                logger,
                "concierge.lock.lost",
                conversation=fingerprint(lease.identity),
            )
            # Stop the protected work: no more tools, model calls or writes.
            task.cancel()
            return

    async def _process_locked(
        self,
        message: InboundMessage,
        channel_key: str,
        lease: ConversationLease,
    ) -> ProcessingRecord:
        # Re-read under the lock: another worker may have finished it.
        record = await self._inbound_record(message)
        if record is not None and record.prepared:
            return record
        record = record or ProcessingRecord(message_id=message.message_id)
        record.conversation = lease.identity
        record.attempts += 1
        session = await self.store.get(message.phone, channel_key) or ConversationSession(
            phone=message.phone,
            channel_key=channel_key,
        )
        session.provider = message.provider
        session.country_code = message.country_code or session.country_code
        session.customer_identity = message.identity()
        session.last_customer_message_at = max(
            session.last_customer_message_at,
            message.sent_at or message.received_at,
        )
        journal = TurnJournal(self.store, lease, session, record)

        if record.stage in ("effect_pending", "effects_applied"):
            return await self._recover(message, journal)

        token = extract_qr_token(message.text)
        stale_code = False
        if message.provider == "FUDIA_WEB":
            if not message.bound_qr or (token and token != message.bound_qr):
                return await self._finish(
                    journal, message.text,
                    "Este chat corresponde al QR que escaneaste. Abre el QR del otro local para cambiar.",
                    "none",
                )
        elif token is None:
            token, stale_code = await self._token_from_start_code(message.text)
        if message.sent_at and session.context_sent_at and message.sent_at < session.context_sent_at:
            record.stage = "superseded"
            record.conversation_id = session.conversation_id
            await journal.checkpoint(with_session=False)
            return record
        if message.sequence and session.context_sequence and message.sequence < session.context_sequence:
            # Written to a table the diner already left by scanning a newer
            # QR: it must not touch the new table's cart or confirmation.
            record.stage = "superseded"
            record.conversation_id = session.conversation_id
            await journal.checkpoint(with_session=False)
            MESSAGES_SUPERSEDED.labels(phase="processing").inc()
            log_event(
                logger,
                "concierge.message.superseded",
                message=fingerprint(message.message_id),
            )
            return record
        if message.provider == "FUDIA_WEB" and token and await self._web_attention_live(session, token):
            # Opening the same QR again (new scan or reload) keeps a live
            # attention: no new context, no second welcome, nothing to send.
            record.stage = "superseded"
            record.conversation_id = session.conversation_id
            await journal.checkpoint(with_session=False)
            return record

        record.stage = "processing"
        await journal.checkpoint(with_session=False)
        reply, change = await self._respond(message, token, journal, stale_code=stale_code)
        return await self._finish(journal, message.text, reply, change)

    async def _choose_restaurant(
        self,
        message: InboundMessage,
        journal: TurnJournal,
        *,
        force: bool = False,
        only_visited: bool = True,
        offset: int = 0,
    ) -> tuple[str, SessionChange]:
        session = journal.session
        if self.directory is None:
            return SCAN_QR_REPLY, "none"
        if session.awaiting_confirmation or session.pending_order_request_id or session.handoff_pending:
            return (
                "Tienes una confirmación o atención pendiente. Termínala antes de cambiar. "
                "Si solo deseas vaciar el carrito, escribe «descartar carrito».",
                "none",
            )
        try:
            page = await self.directory.restaurants(
                message.sender_phone_id,
                message.identity(),
                only_visited,
                offset,
            )
            if not page.items and only_visited and not force and message.provider == "META_CLOUD_API":
                # Bridge the authenticated Redis memory created before visits
                # existed. Revalidate its QR and real IDs; entering records the
                # first persistent visit. Never infer a tenant from its name.
                memory = await self.store.get_memory(message.phone, session.channel_key)
                if memory is not None:
                    try:
                        known = await self.fudia.resolve_table(memory.qr_token)
                    except FudiaError:
                        known = None
                    if (
                        known is not None
                        and known.conciergeEnabled
                        and known.organizationId
                        and known.locationId
                        and known.countryCode == message.country_code
                    ):
                        return await self._enter_restaurant(
                            message,
                            journal,
                            RestaurantOption(
                                organizationId=known.organizationId,
                                locationId=known.locationId,
                                organizationName=known.organizationName,
                                locationName=known.locationName,
                                visitCount=1,
                            ),
                        )
            if not page.items and only_visited:
                only_visited, offset = False, 0
                page = await self.directory.restaurants(message.sender_phone_id, message.identity(), False, 0)
        except FudiaError:
            return "No pudimos consultar los restaurantes. Inténtalo nuevamente en un momento.", "none"
        if not page.items:
            return (
                "No hay restaurantes disponibles por este canal en este momento. Inténtalo más tarde.",
                "none",
            )
        if only_visited and len(page.items) == 1 and not page.hasMore and not force:
            return await self._enter_restaurant(message, journal, page.items[0])
        session.selection = RestaurantSelection(
            options=page.items,
            only_visited=only_visited,
            offset=offset,
            has_more=page.hasMore,
        )
        session.conversation_id = conversation_id_for(message.message_id)
        session.context_sequence = message.sequence
        session.context_sent_at = message.sent_at
        session.qr_token = None
        session.table = None
        session.venue = None
        session.mode = "delivery"
        session.cart = []
        session.delivery = DeliveryDetails()
        session.awaiting_confirmation = False
        session.awaiting_contact_phone = False
        session.awaiting_customer_name = False
        session.confirmed_delivery = None
        session.pending_order_request_id = None
        session.last_order_id = None
        session.consent_prompt_pending = False
        session.consent_checked = False
        session.messages = []
        return selection_text(session.selection), "session"

    async def _restaurant_choice(
        self,
        message: InboundMessage,
        journal: TurnJournal,
    ) -> tuple[str, SessionChange]:
        selection = journal.session.selection
        if selection is None:
            return await self._choose_restaurant(message, journal)
        text = message.text.strip().casefold()
        if text in {"más", "mas"} and selection.has_more:
            return await self._choose_restaurant(
                message,
                journal,
                force=True,
                only_visited=selection.only_visited,
                offset=selection.offset + 10,
            )
        if text in {"anterior", "anteriores"} and selection.offset:
            return await self._choose_restaurant(
                message,
                journal,
                force=True,
                only_visited=selection.only_visited,
                offset=max(0, selection.offset - 10),
            )
        if text in {"restaurantes", "cambiar restaurante"}:
            return await self._choose_restaurant(message, journal, force=True)
        choice = option_number(message.text, len(selection.options))
        if choice is None:
            return f"Elige un número de la lista.\n\n{selection_text(selection)}", "session"
        return await self._enter_restaurant(message, journal, selection.options[choice])

    async def _enter_restaurant(
        self,
        message: InboundMessage,
        journal: TurnJournal,
        selected: RestaurantOption,
    ) -> tuple[str, SessionChange]:
        if self.directory is None:
            return SCAN_QR_REPLY, "none"
        try:
            context = await self.directory.restaurant_context(message.sender_phone_id, selected.locationId)
            if context.organizationId != selected.organizationId or not context.qrToken:
                raise FudiaError(404, "restaurant_unavailable", "El restaurante ya no está disponible.")
            table = await self.fudia.resolve_table(context.qrToken)
            if (
                not table.conciergeEnabled
                or table.serviceType != "delivery"
                or table.organizationId != context.organizationId
                or table.locationId != context.locationId
            ):
                raise FudiaError(404, "restaurant_unavailable", "El restaurante ya no está disponible.")
        except FudiaError:
            return (
                "Ese restaurante ya no está disponible. Escribe «restaurantes» para consultar la lista.",
                "none",
            )
        reply, change = await self._start_delivery_link(message, context.qrToken, table, journal)
        if change != "none" and selected.visitCount:
            reply = f"Hola de nuevo.\n\n{reply}"
        return reply, change

    async def _token_from_start_code(self, text: str) -> tuple[str | None, bool]:
        """QR token behind a short start code ("Mesa 7 · K7P2").

        Returns `(token, False)` when a live code is found. A start message
        whose code expired returns `(None, True)`; ordinary text that merely
        looks like a code is ignored.
        """
        candidates = start_code_candidates(text)
        for code in candidates:
            token = await self.start_codes.resolve(code)
            if token:
                return token.lower(), False
        return None, bool(candidates) and looks_like_start_message(text)

    async def _finish(
        self,
        journal: TurnJournal,
        user_message: str,
        reply: str,
        change: SessionChange,
        conversation_id: str = "",
    ) -> ProcessingRecord:
        session = journal.session
        record = journal.record
        reply = without_reference_quotes(reply)
        if change == "transcript":
            session.messages.extend(
                [
                    ChatMessage(role="user", content=user_message),
                    ChatMessage(role="assistant", content=reply),
                ]
            )
            session.messages = session.messages[-20:]
        record.reply = reply
        record.stage = "reply_ready"
        record.conversation_id = conversation_id or session.conversation_id
        meter = current_meter()
        if meter is not None:
            meter.bind(session.qr_token, record.conversation_id)
        # Session, effects and the prepared reply become durable together.
        await journal.checkpoint(with_session=change != "none")
        return record

    async def _respond(
        self,
        message: InboundMessage,
        token: str | None,
        journal: TurnJournal,
        *,
        stale_code: bool = False,
    ) -> tuple[str, SessionChange]:
        session = journal.session
        text = message.text

        if token:
            if (
                session.pending_order_request_id and session.order_submission_started is not False
            ) or session.handoff_pending:
                return (
                    "Tienes una operación pendiente. Espera su confirmación antes de abrir otro QR.",
                    "none",
                )
            try:
                table = await self.fudia.resolve_table(token)
            except FudiaError:
                return (
                    "El QR no está activo o ya no corresponde a una mesa. "
                    "Pide ayuda al personal del restaurante.",
                    "none",
                )
            if not table.conciergeEnabled:
                return (
                    "Fudia Concierge no está habilitado para este restaurante "
                    "en este momento. Pide ayuda al personal.",
                    "none",
                )
            session.country_code = table.countryCode or session.country_code
            if (
                self.directory is not None
                and session.mode == "table"
                and session.table is not None
                and session.qr_token
                and session.qr_token != token
            ):
                same_venue = (
                    bool(table.organizationId)
                    and bool(session.table.organizationId)
                    and table.organizationId == session.table.organizationId
                    and table.locationId == session.table.locationId
                )
                if not same_venue and not await self._table_service_ended(session):
                    return (
                        "Tu atención en el restaurante anterior sigue vigente. "
                        "Solicita su cierre al personal antes de cambiar.",
                        "none",
                    )
            if table.serviceType == "delivery":
                return await self._start_delivery_link(message, token, table, journal)
            conversation_id = conversation_id_for(message.message_id)
            # Web chat: the page opens the welcome by itself, so the
            # conversation is counted with the diner's first interaction.
            deferred = message.provider == "FUDIA_WEB"
            if not deferred:
                opened = await self._open_conversation(token, conversation_id, "table", message=message)
                # The current session, if any, stays as it was: a conversation in
                # progress is never cut, and nobody is seated at a stale account.
                if opened == "limit":
                    return CONVERSATION_LIMIT_TABLE_REPLY, "none"
                if opened == "stale_account":
                    return STALE_TABLE_ACCOUNT_REPLY, "none"
            # A new QR opens a new context: the previous table's cart,
            # pending confirmation and handoff no longer apply.
            session.qr_token = token
            session.selection = None
            session.conversation_id = conversation_id
            session.conversation_counted = not deferred
            session.context_sequence = message.sequence
            session.context_sent_at = message.sent_at
            session.table = table
            counter = table.serviceType == "counter"
            # foods-backend decides the order flow from the QR; the session
            # only mirrors it to guide the conversation.
            session.mode = "counter" if counter else "table"
            session.venue = VenueContext(
                organizationId=table.organizationId,
                locationId=table.locationId,
                organizationName=table.organizationName,
                locationName=table.locationName,
            )
            session.delivery = DeliveryDetails(customer_name=session.delivery.customer_name)
            session.last_order_id = None
            session.last_activity_at = self.clock()
            session.cart = []
            session.awaiting_confirmation = False
            session.awaiting_contact_phone = False
            session.awaiting_customer_name = False
            session.confirmed_delivery = None
            session.pending_order_request_id = None
            session.handoff_pending = False
            session.consent_checked = False
            session.consent_prompt_pending = False
            if counter:
                # The cashier calls the order by name: a returning diner is
                # greeted by it, a new one is asked for it.
                profile = await self._customer_profile(token, session.profile_phone)
                session.delivery = DeliveryDetails(customer_name=profile_name(profile))
                session.awaiting_customer_name = not full_customer_name(session.delivery.customer_name)
                session.consent_checked = bool(profile and profile.marketingConsent != "pending")
            try:
                menu = await self.fudia.search_menu(token)
                session.currency_symbol = menu.currencySymbol
                # Web chat: the header already carries the logo and «Carta»
                # opens the dishes, so the welcome sends no images.
                if not deferred:
                    journal.record.media = [
                        ReplyImage(url=url, caption=f"Carta virtual · página {index + 1}")
                        for index, url in enumerate(menu.virtualMenuUrls)
                    ]
                    if menu.restaurantLogoUrl:
                        journal.record.media.insert(
                            0,
                            ReplyImage(url=menu.restaurantLogoUrl, caption=table.organizationName),
                        )
                carta: Carta | None = await build_carta(self.fudia, token, menu)
            except (FudiaError, ValueError):
                # The carta is a courtesy: the table is already open, so the
                # greeting goes out and the diner can still ask for it.
                carta = None
            reply = welcome_message(
                table, carta, customer_name=session.delivery.customer_name, web=deferred
            )
            session.messages = [
                ChatMessage(
                    role="user",
                    content=(
                        "Inicié mi pedido desde el QR del mostrador."
                        if counter
                        else "Inicié mi pedido desde el QR de la mesa."
                    ),
                ),
                ChatMessage(role="assistant", content=reply),
            ]
            return reply, "session"

        if stale_code and (session.mode != "table" or session.table is None):
            # An old start message must not reopen a venue or seat the diner.
            return EXPIRED_START_CODE_REPLY, "none"

        if self.directory is not None and session.provider != "FUDIA_WEB":
            if text.strip().casefold() == "descartar carrito":
                if (
                    session.pending_order_request_id and session.order_submission_started is not False
                ) or session.handoff_pending:
                    return "Tienes una operación pendiente. Espera su confirmación antes de cambiar.", "none"
                if (
                    session.mode == "table"
                    and session.table is not None
                    and not await self._table_service_ended(session)
                ):
                    session.cart = []
                    session.awaiting_confirmation = False
                    session.pending_order_request_id = None
                    return "El carrito está vacío. Tu atención en esta mesa continúa.", "session"
                session.cart = []
                session.awaiting_confirmation = False
                session.pending_order_request_id = None
                return await self._choose_restaurant(message, journal, force=True)
            if session.selection is not None:
                return await self._restaurant_choice(message, journal)
            changing = text.strip().casefold() in {"cambiar restaurante", "otro restaurante", "restaurantes"}
            ended = (
                session.mode == "table"
                and session.table is not None
                and await self._table_service_ended(session)
            )
            if changing and session.mode == "table" and session.table is not None and not ended:
                return (
                    "Tu atención en mesa sigue vigente. "
                    "Solicita su cierre al personal antes de cambiar de restaurante.",
                    "none",
                )
            if changing or ended or not session.qr_token:
                return await self._choose_restaurant(message, journal, force=changing)

        notice = ""
        if not session.qr_token:
            # No live session: the venue of the last QR still takes delivery
            # or pickup orders; without one the diner must scan a QR.
            memory = await self.store.get_memory(message.phone, session.channel_key)
            if memory is None:
                return SCAN_QR_REPLY, "none"
            if (
                await self._open_conversation(
                    memory.qr_token,
                    conversation_id_for(message.message_id),
                    "delivery",
                    message=message,
                )
                != "open"
            ):
                return CONVERSATION_LIMIT_DELIVERY_REPLY, "none"
            session.qr_token = memory.qr_token
            self._open_delivery(session, message, memory.venue, memory.customer_name)
            notice = (
                f"Hola de nuevo. Puedo tomar tu pedido de "
                f"{memory.venue.organizationName} para recojo en el local "
                "o delivery."
            )
        elif session.mode == "table":
            if session.table is None:
                return SCAN_QR_REPLY, "none"
            if await self._table_service_ended(session):
                if (
                    await self._open_conversation(
                        session.qr_token or "",
                        conversation_id_for(message.message_id),
                        "delivery",
                        message=message,
                    )
                    != "open"
                ):
                    return conversation_limit_reply(session.provider, "delivery"), "none"
                venue = session.venue or VenueContext(
                    organizationName=session.table.organizationName,
                    locationName=session.table.locationName,
                )
                self._open_delivery(session, message, venue, session.delivery.customer_name)
                notice = (
                    "Tu atención en mesa ya terminó. Ahora puedo tomar tu "
                    f"pedido de {venue.organizationName} para recojo en el "
                    "local o delivery."
                )
        if not session.conversation_counted:
            # First interaction after a web welcome: only now the attention
            # counts as a conversation. A refusal leaves the session as it was,
            # so the next message asks foods-backend again.
            kind: Literal["table", "delivery"] = "delivery" if session.mode == "delivery" else "table"
            opened = await self._open_conversation(
                session.qr_token or "", session.conversation_id, kind, message=message
            )
            if opened == "limit":
                return conversation_limit_reply(session.provider, kind), "none"
            if opened == "stale_account":
                return STALE_TABLE_ACCOUNT_REPLY, "none"
            session.conversation_counted = True
        session.last_activity_at = self.clock()

        if message.location is not None:
            reply, change = await self._location_reply(session, message.location)
            if notice:
                return f"{notice}\n\n{reply}", "transcript"
            return reply, change

        if session.mode == "table" and session.handoff_pending:
            try:
                handoff = await self.fudia.get_handoff_status(
                    session.qr_token,
                    session.conversation_id,
                )
            except FudiaError:
                return (
                    "Ya solicitaste atención del personal. No pude verificar todavía si fue atendida.",
                    "none",
                )
            if str(handoff.get("status", "")) == "pending":
                return (
                    "El personal del local ya fue avisado. Espera un momento mientras se acercan a tu mesa.",
                    "none",
                )
            session.handoff_pending = False
            await journal.checkpoint()

        if message.web_action is not None and session.provider == "FUDIA_WEB":
            action = message.web_action
            tools = ToolRegistry(TurnScopedFudia(self.fudia), session, message.text, journal)

            def cart_name() -> str:
                product_id = str(action.args.get("productId", ""))
                return next((line.name for line in session.cart if line.product_id == product_id), "")

            # Read before the tool runs: a removed dish is no longer in the cart.
            item = cart_name()
            if action.name == "confirm_order" and (
                not action.confirmation_ref
                or action.confirmation_ref != session.pending_order_request_id
                or not session.awaiting_confirmation
            ):
                result = {"ok": False, "code": "stale_confirmation"}
            else:
                result = await tools.execute_any(action.name, action.args)
            symbol = session.currency_symbol
            if not symbol:
                symbol = (await self.fudia.search_menu(session.qr_token or "")).currencySymbol
            quantity = action.args.get("quantity", 1)
            reply = web_action_reply(
                action.name,
                result,
                symbol,
                dish=item or cart_name(),
                quantity=quantity if isinstance(quantity, (int, float)) else 1,
            )
            # A button pressed after the table attention ended must still say
            # that the chat now takes delivery or pickup orders.
            return (f"{notice}\n\n{reply}" if notice else reply), "transcript"

        if session.consent_prompt_pending:
            # The promotional question is answered once, without the model;
            # anything else continues the order without interrupting it.
            session.consent_prompt_pending = False
            answer = consent_answer(text)
            if answer is not None:
                reply = await self._record_consent(session, journal, answer, text)
                return (f"{notice}\n\n{reply}" if notice else reply), "transcript"
        if session.qr_token and asks_to_withdraw(text):
            reply = await self._record_consent(session, journal, "withdrawn", text)
            return (f"{notice}\n\n{reply}" if notice else reply), "transcript"

        # A phone reply belongs to checkout, even though digits and «+» also
        # match the arithmetic guard. Handle it before scope classification.
        if session.provider in {"WHATSMEOW", "FUDIA_WEB"} and not session.contact_phone:
            contact = normalize_contact_phone(text, session.country_code)
            if contact and (session.awaiting_contact_phone or session.cart):
                session.customer_contact_phone = contact
                session.awaiting_contact_phone = False
                session.awaiting_confirmation = False
                session.pending_order_request_id = None
                session.order_submission_started = False
                reply = "Guardé tu número de contacto."
                if session.cart:
                    tools = ToolRegistry(TurnScopedFudia(self.fudia), session, text, journal)
                    prepared = await tools.execute_any("prepare_confirmation", {})
                    reply += "\n\n" + web_action_reply(
                        "prepare_confirmation", prepared, session.currency_symbol,
                        web=session.provider == "FUDIA_WEB",
                    )
                await journal.local_effect("contact", {"reply": reply})
                return (f"{notice}\n\n{reply}" if notice else reply), "transcript"
            if session.awaiting_contact_phone and phone_text(text):
                return contact_phone_prompt(session.country_code, invalid=True), "transcript"

        if session.awaiting_customer_name:
            name = customer_name_reply(text)
            if name == "":
                return CUSTOMER_NAME_PROMPT, "transcript"
            if name is not None:
                tools = ToolRegistry(TurnScopedFudia(self.fudia), session, text, journal)
                await tools.execute_any("set_delivery_details", {"customerName": name})
                reply = "Guardé tus nombres y apellidos."
                if session.cart:
                    prepared = await tools.execute_any("prepare_confirmation", {})
                    reply += "\n\n" + web_action_reply(
                        "prepare_confirmation", prepared, session.currency_symbol,
                        web=session.provider == "FUDIA_WEB",
                    )
                await journal.local_effect("contact", {"reply": reply})
                return (f"{notice}\n\n{reply}" if notice else reply), "transcript"

        if obviously_out_of_scope(text):
            SCOPE_DECISIONS.labels(decision="out_of_scope").inc()
            if notice:
                return f"{notice}\n\n{OFF_SCOPE_REPLY}", "transcript"
            return OFF_SCOPE_REPLY, "none"

        result = await self.graph.ainvoke(
            {
                "session": session,
                "user_message": text,
                "intent": "order",
                "reply": "",
                "journal": journal,
            }
        )
        reply = whatsapp_text(str(result["reply"]), allow_bold=session.provider == "FUDIA_WEB")
        question = await self._consent_question(session, journal)
        if question:
            reply = f"{reply}\n\n{question}"
        return (f"{notice}\n\n{reply}" if notice else reply), "transcript"

    async def _customer_profile(self, token: str, phone: str) -> CustomerProfile | None:
        """Best effort: without it the diner is simply asked for a name."""
        if not phone:
            return None
        try:
            return await self.fudia.get_customer(token, phone)
        except Exception as exc:
            log_event(
                logger,
                "concierge.customer.lookup_failed",
                errorType=type(exc).__name__,
                code=getattr(exc, "code", ""),
            )
            return None

    async def _consent_question(
        self,
        session: ConversationSession,
        journal: TurnJournal,
    ) -> str:
        """After the first confirmed counter order, ask once per company
        (foods-backend remembers it) for promotional consent. Table and
        delivery conversations keep their current flow; withdrawal works in
        every mode."""
        created = any(
            effect.kind == "create_order" and effect.status == "done" for effect in journal.record.effects
        )
        if not created or session.mode != "counter" or session.consent_checked or not session.qr_token:
            return ""
        session.consent_checked = True
        token = session.qr_token
        profile = await self._customer_profile(token, session.profile_phone)
        if profile is None or profile.marketingConsent != "pending":
            return ""
        venue = session.venue
        question = consent_question(venue.organizationName if venue else "")
        request_id = f"consent-ask-{session.conversation_id}"
        payload = {
            "token": token,
            "phone": session.profile_phone,
            "conversationId": session.conversation_id,
            "decision": "asked",
            "evidence": question,
        }
        await journal.before_effect("marketing_consent", request_id, payload)
        try:
            await self.fudia.record_marketing_consent(
                token,
                session.profile_phone,
                session.conversation_id,
                request_id,
                "asked",
                question,
            )
        except FudiaError as exc:
            # Without a recorded question it is not asked: it would be asked
            # again in another attention.
            await journal.effect_failed(exc.code, 400 <= exc.status_code < 500)
            return ""
        session.consent_prompt_pending = True
        await journal.after_effect({"decision": "asked", "question": question})
        return question

    async def _record_consent(
        self,
        session: ConversationSession,
        journal: TurnJournal,
        decision: Literal["granted", "declined", "withdrawn"],
        evidence: str,
    ) -> str:
        """Store the diner's answer (or withdrawal) with their literal words."""
        if not session.profile_phone:
            return "Necesitamos verificar tu identidad de WhatsApp para actualizar preferencias."
        venue = session.venue
        reply = consent_reply(decision, venue.organizationName if venue else "")
        token = session.qr_token or ""
        request_id = uuid4().hex
        await journal.before_effect(
            "marketing_consent",
            request_id,
            {
                "token": token,
                "phone": session.profile_phone,
                "conversationId": session.conversation_id,
                "decision": decision,
                "evidence": evidence[:500],
            },
        )
        try:
            await self.fudia.record_marketing_consent(
                token,
                session.profile_phone,
                session.conversation_id,
                request_id,
                decision,
                evidence[:500],
            )
        except FudiaError as exc:
            await journal.effect_failed(exc.code, 400 <= exc.status_code < 500)
            if exc.code == "customer_not_found":
                return "Todavía no tienes promociones activas con nosotros."
            return "No pude guardar tu preferencia en este momento. Escríbemela de nuevo en unos minutos."
        await journal.after_effect({"decision": decision, "reply": reply})
        return reply

    async def _table_service_ended(
        self,
        session: ConversationSession,
    ) -> bool:
        """The QR is permanent per table: once the diner's account ended the
        next diner may already be seated, so this conversation must stop
        adding rounds to that table."""
        if session.handoff_pending:
            return False
        if session.last_order_id and session.qr_token:
            try:
                status = await self.fudia.get_order_status(
                    session.qr_token,
                    session.last_order_id,
                    session.conversation_id,
                )
            except FudiaError as exc:
                return exc.code == "order_not_found"
            if not status.active:
                return True
        # With or without an order: hours without a message mean the diner
        # may have left, even leaving the account open. The account itself is
        # protected by foods-backend (a new QR does not join a stale account).
        return bool(
            session.last_activity_at and self.clock() - session.last_activity_at > self.table_idle_seconds
        )

    async def _web_attention_live(self, session: ConversationSession, token: str) -> bool:
        """Web chat: a scan cannot be told apart from reopening the page, so
        the QR opens a new context only when its own attention is over (the
        table account ended or the chat moved on to delivery). Otherwise the
        cart, the confirmation and the counted conversation are kept."""
        if session.qr_token != token:
            return False
        try:
            table = await self.fudia.resolve_table(token)
        except FudiaError:
            return False
        if not table.conciergeEnabled or session.mode != table.serviceType:
            return False
        if session.mode != "table":
            return True
        return session.table is not None and not await self._table_service_ended(session)

    async def _start_delivery_link(
        self,
        message: InboundMessage,
        token: str,
        table: TableContext,
        journal: TurnJournal,
    ) -> tuple[str, SessionChange]:
        """Enlace/QR de delivery del local: abre directamente el pedido de
        delivery o recojo, sin haber pasado antes por una mesa."""
        session = journal.session
        # Web chat: counted with the diner's first interaction (see _respond).
        deferred = message.provider == "FUDIA_WEB"
        if (
            not deferred
            and await self._open_conversation(
                token, conversation_id_for(message.message_id), "delivery", message=message
            )
            != "open"
        ):
            return CONVERSATION_LIMIT_DELIVERY_REPLY, "none"
        venue = VenueContext(
            organizationId=table.organizationId,
            locationId=table.locationId,
            organizationName=table.organizationName,
            locationName=table.locationName,
        )
        profile = await self._customer_profile(token, session.profile_phone)
        session.qr_token = token
        self._open_delivery(session, message, venue, profile_name(profile))
        session.conversation_counted = not deferred
        session.consent_checked = bool(profile and profile.marketingConsent != "pending")
        session.consent_prompt_pending = False
        session.last_activity_at = self.clock()
        try:
            menu = await self.fudia.search_menu(token)
            session.currency_symbol = menu.currencySymbol
            # Web chat: no logo nor carta pages in the welcome, as with a table QR.
            if not deferred:
                journal.record.media = [
                    ReplyImage(url=url, caption=f"Carta virtual · página {index + 1}")
                    for index, url in enumerate(menu.virtualMenuUrls)
                ]
                if menu.restaurantLogoUrl:
                    journal.record.media.insert(
                        0, ReplyImage(url=menu.restaurantLogoUrl, caption=table.organizationName)
                    )
            carta: Carta | None = await build_carta(self.fudia, token, menu)
        except (FudiaError, ValueError):
            carta = None
        reply = welcome_message(
            table, carta, customer_name=session.delivery.customer_name, web=deferred
        )
        session.messages = [
            ChatMessage(
                role="user",
                content="Inicié mi pedido desde el enlace de delivery del local.",
            ),
            ChatMessage(role="assistant", content=reply),
        ]
        return reply, "session"

    def _open_delivery(
        self,
        session: ConversationSession,
        message: InboundMessage,
        venue: VenueContext,
        customer_name: str,
    ) -> None:
        """New context without a table: nothing from the table carries over
        (cart, confirmation, handoff or transcript)."""
        session.conversation_id = conversation_id_for(message.message_id)
        session.conversation_counted = True
        session.context_sequence = message.sequence
        session.context_sent_at = message.sent_at
        session.selection = None
        session.mode = "delivery"
        session.table = None
        session.venue = venue
        session.awaiting_contact_phone = False
        session.awaiting_customer_name = False
        session.confirmed_delivery = None
        session.delivery = DeliveryDetails(customer_name=customer_name)
        session.cart = []
        session.awaiting_confirmation = False
        session.pending_order_request_id = None
        session.last_order_id = None
        session.handoff_pending = False
        session.messages = []
        session.consent_checked = False
        session.consent_prompt_pending = False

    async def _location_reply(
        self,
        session: ConversationSession,
        point: GeoPoint,
    ) -> tuple[str, SessionChange]:
        """A shared pin is quoted by foods-backend without the model."""
        if session.mode == "counter":
            return (
                "Recibí tu ubicación. Este pedido es para recoger en el mostrador del local.",
                "none",
            )
        if session.mode != "delivery" or not session.qr_token:
            return (
                "Recibí tu ubicación. Mientras tu atención en mesa siga "
                "abierta, tu pedido se atiende en la mesa.",
                "none",
            )
        try:
            quote = await self.fudia.quote_delivery(session.qr_token, point.latitude, point.longitude)
        except FudiaError as exc:
            if exc.code == "delivery_unavailable":
                return (
                    "Por ahora este local no tiene delivery. Puedes pedir para recoger en el local.",
                    "transcript",
                )
            return (
                "No pude calcular el costo de envío en este momento. "
                "Vuelve a compartir tu ubicación en unos minutos.",
                "none",
            )
        apply_quote(session, point, quote)
        # A new delivery point may change the fee: confirm again.
        session.awaiting_confirmation = False
        session.pending_order_request_id = None
        symbol = quote.currencySymbol
        distance = quote.distanceKm.quantize(Decimal("0.01"))
        if not quote.covered:
            limit = quote.maxDistanceKm.quantize(Decimal("0.01"))
            return (
                f"Tu ubicación está a {distance} km del local y nuestro "
                f"delivery llega hasta {limit} km. Puedes compartir otra "
                "ubicación o recoger tu pedido en el local.",
                "transcript",
            )
        lines = [
            f"Tu ubicación está a {distance} km del local. El delivery "
            f"cuesta {symbol} {quote.fee.quantize(Decimal('0.01'))}.".replace("  ", " ")
        ]
        if not session.delivery.address:
            lines.append("Escríbeme la dirección exacta y una referencia para el repartidor.")
        if not full_customer_name(session.delivery.customer_name):
            lines.append(CUSTOMER_NAME_PROMPT)
        if not session.cart:
            lines.append("¿Qué te gustaría pedir?")
        return " ".join(lines), "transcript"

    async def _recover(
        self,
        message: InboundMessage,
        journal: TurnJournal,
    ) -> ProcessingRecord:
        """Finish a message that persisted effects before it was interrupted,
        without calling the model again."""
        record = journal.record
        pending = record.pending_effect()
        context = ""
        if pending is not None:
            context = str(pending.payload.get("conversationId", ""))
            await self._replay(pending, journal)
        effect = record.effects[-1] if record.effects else None
        PROCESSING_RECOVERIES.labels(
            effect=effect.kind if effect else "none",
            result=effect.status if effect else "none",
        ).inc()
        log_event(
            logger,
            "concierge.processing.recovered",
            message=fingerprint(message.message_id),
            effect=effect.kind if effect else "none",
            replayed=pending is not None,
            attempt=record.attempts,
        )
        reply = recovery_reply(record, journal.session)
        return await self._finish(
            journal,
            message.text,
            reply,
            "transcript",
            conversation_id=context,
        )

    async def _replay(
        self,
        effect: EffectRecord,
        journal: TurnJournal,
    ) -> None:
        """Repeat a backend operation with its original idempotency key: the
        backend returns what it already did instead of doing it twice."""
        session = journal.session
        payload = effect.payload
        token = str(payload.get("token", ""))
        phone = str(payload.get("phone", ""))
        conversation_id = str(payload.get("conversationId", ""))
        same_context = session.conversation_id == conversation_id
        await journal.lease.verify()
        try:
            if effect.kind == "create_order":
                if payload.get("mode") == "delivery":
                    order = await self.fudia.create_delivery_order(token, dict(payload.get("order", {})))
                else:
                    order = await self.fudia.create_order(
                        token,
                        phone,
                        list(payload.get("lines", [])),
                        conversation_id=conversation_id,
                        request_id=effect.request_id,
                        customer_name=str(payload.get("customerName", "")),
                    )
                if same_context and session.pending_order_request_id == effect.request_id:
                    session.cart = []
                    session.awaiting_confirmation = False
                    session.pending_order_request_id = None
                    session.last_order_id = order.id
                    session.confirmed_delivery = None
                    session.order_submission_started = False
                await journal.after_effect(
                    {
                        "order": {
                            "id": order.id,
                            "code": order.code,
                            "status": order.status,
                            "total": str(order.total),
                        }
                    }
                )
            elif effect.kind == "request_handoff":
                handoff = await self.fudia.request_handoff(
                    token,
                    phone,
                    conversation_id,
                    str(payload.get("reason", "")),
                )
                if same_context:
                    session.handoff_pending = True
                await journal.after_effect({"status": str(handoff.get("status", "pending"))})
            elif effect.kind == "marketing_consent":
                decision = str(payload.get("decision", ""))
                evidence = str(payload.get("evidence", ""))
                await self.fudia.record_marketing_consent(
                    token,
                    phone,
                    conversation_id,
                    effect.request_id,
                    cast(ConsentDecision, decision),
                    evidence,
                )
                result: dict[str, Any] = {"decision": decision}
                if decision == "asked":
                    result["question"] = evidence
                    if same_context:
                        session.consent_prompt_pending = True
                else:
                    venue = session.venue
                    result["reply"] = consent_reply(decision, venue.organizationName if venue else "")
                await journal.after_effect(result)
            elif effect.kind == "request_bill":
                bill = await self.fudia.request_bill(token, phone, conversation_id, effect.request_id)
                await journal.after_effect(bill_summary(bill))
        except FudiaError as exc:
            if 400 <= exc.status_code < 500:
                await journal.effect_failed(exc.code, True)
                return
            # Outcome still unknown: keep it pending and retry the delivery.
            raise
