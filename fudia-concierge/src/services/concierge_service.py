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
    ConversationSession,
    DeliveryDetails,
    EffectRecord,
    GeoPoint,
    InboundMessage,
    ProcessingRecord,
    VenueContext,
)
from src.graph.app import build_graph
from src.metrics import (
    LOCK_LOST,
    MESSAGES_SUPERSEDED,
    PROCESSING_RECOVERIES,
    RATE_LIMITED,
    SCOPE_DECISIONS,
)
from src.observability import fingerprint, log_event
from src.services.carta import Carta, build_carta
from src.services.delivery_tools import apply_quote, delivery_summary
from src.services.formatting import whatsapp_text
from src.services.menu_tools import unavailable_carta_reply
from src.services.ports import (
    CONVERSATION_LIMIT_CODE,
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
from src.services.service_tools import bill_summary
from src.services.start_codes import (
    looks_like_start_message,
    start_code_candidates,
    table_label,
)
from src.services.tool_registry import ToolRegistry
from src.services.tooling import cart_payload
from src.services.usage_meter import TokenMeter, current_meter, metering
from src.services.welcome import welcome_message

logger = logging.getLogger(__name__)

QR_PATTERN = re.compile(r"(?i)\bFUDIA:([a-f0-9]{32})\b")
ARITHMETIC_ONLY_PATTERN = re.compile(
    r"(?i)^\s*(?:cu[aá]nto\s+(?:es|da)\s+|calcula\s+|resuelve\s+)?"
    r"[0-9\s.,/()+\-*x×÷=]+\??\s*$"
)
MATH_TOPIC_PATTERN = re.compile(
    r"(?i)\b(fracciones?|ecuaciones?|ra[ií]z cuadrada|derivadas?|integrales?)\b"
)
OFF_SCOPE_REPLY = (
    "Puedo ayudarte únicamente con el pedido en este restaurante: consultar la "
    "carta, agregar o retirar productos, confirmar pedidos, delivery o recojo, "
    "pedir la cuenta o solicitar atención del personal."
)
SCAN_QR_REPLY = (
    "Para atenderte en tu mesa, vuelve a escanear el QR y envía el mensaje "
    "tal como aparece."
)
EXPIRED_START_CODE_REPLY = (
    "Ese código de mesa ya venció. Vuelve a escanear el QR de tu mesa y envía "
    "el mensaje tal como aparece."
)
TOO_LONG_REPLY = (
    "Tu mensaje es demasiado largo. "
    "Envíame solo lo necesario para gestionar tu pedido."
)
RATE_LIMITED_REPLY = (
    "Has enviado muchos mensajes en poco tiempo. "
    "Continúa con tu pedido dentro de un momento."
)
# foods-backend did not open a new conversation this month. The diner is
# invited to order in person; caps and plans are never mentioned.
CONVERSATION_LIMIT_TABLE_REPLY = (
    "En este momento no podemos tomar tu pedido por WhatsApp. "
    "Por favor, pide directamente al personal del restaurante en tu mesa."
)
CONVERSATION_LIMIT_DELIVERY_REPLY = (
    "En este momento no podemos tomar pedidos por WhatsApp. "
    "Te esperamos en el local para atenderte en mesa."
)

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


def extract_qr_token(text: str) -> str | None:
    match = QR_PATTERN.search(text)
    return match.group(1).lower() if match else None


def obviously_out_of_scope(text: str) -> bool:
    value = text.strip().strip("¿?¡!").strip()
    if not value:
        return False
    return bool(
        ARITHMETIC_ONLY_PATTERN.fullmatch(value)
        or MATH_TOPIC_PATTERN.search(value)
    )


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
        details = [
            f"{selection['groupName']}: {selection['name']}"
            for selection in item["selections"]
        ] + [modifier["name"] for modifier in item["modifiers"]]
        if item["note"]:
            details.append(f"nota: {item['note']}")
        extra = f" ({'; '.join(details)})" if details else ""
        lines.append(
            f"- {quantity} × {item['name']}{extra}: {item['lineTotal']}"
        )
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


def agent_message(session: ConversationSession, user_message: str) -> str:
    """Every turn carries the current state of the attention (table or
    delivery, cart, pending confirmation) so the agent reads each message in
    context instead of in isolation."""
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
            else (
                f"fuera de cobertura ({summary['distanceKm']} km; máximo "
                f"{summary['maxDistanceKm']} km)"
            )
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
    ) -> None:
        self.store = store
        self.table_idle_seconds = table_idle_seconds
        self.fudia = fudia
        self.conversation_lock = (
            conversation_lock or LocalConversationLock()
        )
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
        ] = {
            intent: self._processor(intent)
            for intent in ("menu", "order", "service")
        }
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
            registry = ToolRegistry(
                self.fudia,
                session,
                user_message,
                journal,
            )
            fallback = ""
            action_attempted = False

            async def execute(
                name: str,
                args: dict[str, Any],
            ) -> dict[str, Any]:
                nonlocal fallback, action_attempted
                result = await registry.execute_for(
                    intent,
                    name,
                    args,
                )
                if name == "search_menu":
                    fallback = unavailable_carta_reply(result) or fallback
                elif name not in READ_ONLY_TOOLS:
                    action_attempted = True
                return result

            # Tools keep the diner's literal words (explicit confirmation);
            # only the agent also reads the current delivery state.
            reply = await self.agents[intent].reply(
                session,
                agent_message(session, user_message),
                execute,
            )
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

    async def process_inbound(
        self,
        message: InboundMessage,
    ) -> ProcessingRecord:
        if not message.received_at:
            message = message.model_copy(
                update={"received_at": self.clock()}
            )
        record = await self.store.get_record(message.message_id)
        if record is not None and record.prepared:
            return record

        channel_key = message.sender_phone_id.strip() or "default"
        identity = conversation_identity(message.phone, channel_key)
        if len(message.text) > self.max_message_chars:
            return await self._prepare_unlocked(
                message, identity, record, TOO_LONG_REPLY
            )
        # Only the first attempt counts against the limit: a retry of the
        # same message is not new traffic from the diner.
        if record is None and not await self.rate_limiter.allow(identity):
            RATE_LIMITED.inc()
            return await self._prepare_unlocked(
                message, identity, record, RATE_LIMITED_REPLY
            )

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
            await self.fudia.report_usage(
                meter.qr_token, meter.conversation_id, request_id, tiers
            )
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
    ) -> bool:
        """Count a new attention. False only when foods-backend refuses new
        conversations this month; any other failure keeps serving the diner
        (the conversation is still counted by its first usage report)."""
        try:
            await self.fudia.open_conversation(token, conversation_id, kind)
        except FudiaError as exc:
            if exc.code == CONVERSATION_LIMIT_CODE:
                log_event(
                    logger,
                    "concierge.conversation.limit_reached",
                    conversation=conversation_id,
                    kind=kind,
                )
                return False
            log_event(
                logger,
                "concierge.conversation.open_failed",
                conversation=conversation_id,
                kind=kind,
                code=exc.code,
            )
        return True

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
                raise LockLostError(
                    "La conversación pasó a otro proceso."
                ) from None
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
        record = await self.store.get_record(message.message_id)
        if record is not None and record.prepared:
            return record
        record = record or ProcessingRecord(message_id=message.message_id)
        record.conversation = lease.identity
        record.attempts += 1
        session = await self.store.get(
            message.phone, channel_key
        ) or ConversationSession(
            phone=message.phone,
            channel_key=channel_key,
        )
        journal = TurnJournal(self.store, lease, session, record)

        if record.stage in ("effect_pending", "effects_applied"):
            return await self._recover(message, journal)

        token = extract_qr_token(message.text)
        stale_code = False
        if token is None:
            token, stale_code = await self._token_from_start_code(message.text)
        if (
            message.sequence
            and session.context_sequence
            and message.sequence < session.context_sequence
        ):
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

        record.stage = "processing"
        await journal.checkpoint(with_session=False)
        reply, change = await self._respond(
            message, token, journal, stale_code=stale_code
        )
        return await self._finish(journal, message.text, reply, change)

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
            conversation_id = conversation_id_for(message.message_id)
            if not await self._open_conversation(
                token, conversation_id, "table"
            ):
                # The current session, if any, stays as it was: a
                # conversation in progress is never cut.
                return CONVERSATION_LIMIT_TABLE_REPLY, "none"
            # A new QR opens a new context: the previous table's cart,
            # pending confirmation and handoff no longer apply.
            session.qr_token = token
            session.conversation_id = conversation_id
            session.context_sequence = message.sequence
            session.table = table
            session.mode = "table"
            session.venue = VenueContext(
                organizationName=table.organizationName,
                locationName=table.locationName,
            )
            session.delivery = DeliveryDetails(
                customer_name=session.delivery.customer_name
            )
            session.last_order_id = None
            session.last_activity_at = self.clock()
            session.cart = []
            session.awaiting_confirmation = False
            session.pending_order_request_id = None
            session.handoff_pending = False
            try:
                carta: Carta | None = await build_carta(
                    self.fudia, token, await self.fudia.search_menu(token)
                )
            except (FudiaError, ValueError):
                # The carta is a courtesy: the table is already open, so the
                # greeting goes out and the diner can still ask for it.
                carta = None
            reply = welcome_message(table, carta)
            session.messages = [
                ChatMessage(
                    role="user",
                    content="Inicié mi pedido desde el QR de la mesa.",
                ),
                ChatMessage(role="assistant", content=reply),
            ]
            return reply, "session"

        if stale_code and (session.mode != "table" or session.table is None):
            # An old start message must not reopen a venue or seat the diner.
            return EXPIRED_START_CODE_REPLY, "none"

        notice = ""
        if not session.qr_token:
            # No live session: the venue of the last QR still takes delivery
            # or pickup orders; without one the diner must scan a QR.
            memory = await self.store.get_memory(
                message.phone, session.channel_key
            )
            if memory is None:
                return SCAN_QR_REPLY, "none"
            if not await self._open_conversation(
                memory.qr_token,
                conversation_id_for(message.message_id),
                "delivery",
            ):
                return CONVERSATION_LIMIT_DELIVERY_REPLY, "none"
            session.qr_token = memory.qr_token
            self._open_delivery(
                session, message, memory.venue, memory.customer_name
            )
            notice = (
                f"Hola de nuevo. Puedo tomar tu pedido de "
                f"{memory.venue.organizationName} para recojo en el local "
                "o delivery."
            )
        elif session.mode == "table":
            if session.table is None:
                return SCAN_QR_REPLY, "none"
            if await self._table_service_ended(session):
                if not await self._open_conversation(
                    session.qr_token or "",
                    conversation_id_for(message.message_id),
                    "delivery",
                ):
                    return CONVERSATION_LIMIT_DELIVERY_REPLY, "none"
                venue = session.venue or VenueContext(
                    organizationName=session.table.organizationName,
                    locationName=session.table.locationName,
                )
                self._open_delivery(
                    session, message, venue, session.delivery.customer_name
                )
                notice = (
                    "Tu atención en mesa ya terminó. Ahora puedo tomar tu "
                    f"pedido de {venue.organizationName} para recojo en el "
                    "local o delivery."
                )
        session.last_activity_at = self.clock()

        if message.location is not None:
            reply, change = await self._location_reply(
                session, message.location
            )
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
                    "Ya solicitaste atención del personal. "
                    "No pude verificar todavía si fue atendida.",
                    "none",
                )
            if str(handoff.get("status", "")) == "pending":
                return (
                    "El personal del local ya fue avisado. "
                    "Espera un momento mientras se acercan a tu mesa.",
                    "none",
                )
            session.handoff_pending = False
            await journal.checkpoint()

        if obviously_out_of_scope(text):
            SCOPE_DECISIONS.labels(
                decision="out_of_scope"
            ).inc()
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
        reply = whatsapp_text(str(result["reply"]))
        return (f"{notice}\n\n{reply}" if notice else reply), "transcript"

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
            return not status.active
        return bool(
            session.last_activity_at
            and self.clock() - session.last_activity_at
            > self.table_idle_seconds
        )

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
        session.context_sequence = message.sequence
        session.mode = "delivery"
        session.table = None
        session.venue = venue
        session.delivery = DeliveryDetails(customer_name=customer_name)
        session.cart = []
        session.awaiting_confirmation = False
        session.pending_order_request_id = None
        session.last_order_id = None
        session.handoff_pending = False
        session.messages = []

    async def _location_reply(
        self,
        session: ConversationSession,
        point: GeoPoint,
    ) -> tuple[str, SessionChange]:
        """A shared pin is quoted by foods-backend without the model."""
        if session.mode != "delivery" or not session.qr_token:
            return (
                "Recibí tu ubicación. Mientras tu atención en mesa siga "
                "abierta, tu pedido se atiende en la mesa.",
                "none",
            )
        try:
            quote = await self.fudia.quote_delivery(
                session.qr_token, point.latitude, point.longitude
            )
        except FudiaError as exc:
            if exc.code == "delivery_unavailable":
                return (
                    "Por ahora este local no tiene delivery. Puedes pedir "
                    "para recoger en el local.",
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
            f"cuesta {symbol} {quote.fee.quantize(Decimal('0.01'))}.".replace(
                "  ", " "
            )
        ]
        if not session.delivery.address:
            lines.append(
                "Escríbeme la dirección exacta y una referencia para el "
                "repartidor."
            )
        if not session.delivery.customer_name:
            lines.append("¿A nombre de quién va el pedido?")
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
                    order = await self.fudia.create_delivery_order(
                        token, dict(payload.get("order", {}))
                    )
                else:
                    order = await self.fudia.create_order(
                        token,
                        phone,
                        list(payload.get("lines", [])),
                        conversation_id=conversation_id,
                        request_id=effect.request_id,
                    )
                if (
                    same_context
                    and session.pending_order_request_id == effect.request_id
                ):
                    session.cart = []
                    session.awaiting_confirmation = False
                    session.pending_order_request_id = None
                    session.last_order_id = order.id
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
                await journal.after_effect(
                    {"status": str(handoff.get("status", "pending"))}
                )
            elif effect.kind == "request_bill":
                bill = await self.fudia.request_bill(
                    token, phone, conversation_id, effect.request_id
                )
                await journal.after_effect(bill_summary(bill))
        except FudiaError as exc:
            if 400 <= exc.status_code < 500:
                await journal.effect_failed(exc.code, True)
                return
            # Outcome still unknown: keep it pending and retry the delivery.
            raise
