from __future__ import annotations

import asyncio
import logging
import re
import time
from collections.abc import Awaitable, Callable, Coroutine
from typing import Any, Literal, cast
from uuid import uuid4

from src.ai.contracts import ConciergeRouter, ConversationAgent
from src.domain.intents import AgentIntent, ConciergeRoute
from src.domain.models import (
    ChatMessage,
    ConversationSession,
    EffectRecord,
    InboundMessage,
    ProcessingRecord,
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
from src.services.ports import (
    AllowAllRateLimiter,
    ConversationLease,
    ConversationLock,
    ConversationStore,
    FudiaBackend,
    FudiaError,
    LocalConversationLock,
    LockLostError,
    RateLimiter,
    conversation_identity,
)
from src.services.processing import TurnJournal, recovery_reply
from src.services.service_tools import bill_summary
from src.services.tool_registry import ToolRegistry

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
    "Puedo ayudarte únicamente con el pedido de esta mesa: consultar la carta, "
    "agregar o retirar productos, confirmar pedidos, pedir la cuenta o solicitar "
    "atención del personal."
)
TOO_LONG_REPLY = (
    "Tu mensaje es demasiado largo. "
    "Envíame solo lo necesario para gestionar tu pedido."
)
RATE_LIMITED_REPLY = (
    "Has enviado muchos mensajes en poco tiempo. "
    "Continúa con tu pedido dentro de un momento."
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
        max_message_chars: int = 2000,
        lock_heartbeat_seconds: float = 5.0,
        lock_ttl_seconds: float = 30.0,
        clock: Callable[[], float] = time.time,
    ) -> None:
        self.store = store
        self.fudia = fudia
        self.conversation_lock = (
            conversation_lock or LocalConversationLock()
        )
        self.rate_limiter = rate_limiter or AllowAllRateLimiter()
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

            async def execute(
                name: str,
                args: dict[str, Any],
            ) -> dict[str, Any]:
                return await registry.execute_for(
                    intent,
                    name,
                    args,
                )

            return await self.agents[intent].reply(
                session,
                user_message,
                execute,
            )

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

        lease = ConversationLease(
            self.conversation_lock,
            identity,
            await self.conversation_lock.acquire(identity),
        )
        try:
            return await self._guarded(
                lease,
                self._process_locked(message, channel_key, lease),
            )
        finally:
            await self.conversation_lock.release(identity, lease.token)

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
        reply, change = await self._respond(message, token, journal)
        return await self._finish(journal, message.text, reply, change)

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
        # Session, effects and the prepared reply become durable together.
        await journal.checkpoint(with_session=change != "none")
        return record

    async def _respond(
        self,
        message: InboundMessage,
        token: str | None,
        journal: TurnJournal,
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
            # A new QR opens a new context: the previous table's cart,
            # pending confirmation and handoff no longer apply.
            session.qr_token = token
            session.conversation_id = uuid4().hex
            session.context_sequence = message.sequence
            session.table = table
            session.cart = []
            session.awaiting_confirmation = False
            session.pending_order_request_id = None
            session.handoff_pending = False
            reply = (
                f"Hola, estás en {table.organizationName}, mesa {table.name}. "
                "¿Qué deseas pedir?"
            )
            session.messages = [
                ChatMessage(
                    role="user",
                    content="Inicié mi pedido desde el QR de la mesa.",
                ),
                ChatMessage(role="assistant", content=reply),
            ]
            return reply, "session"

        if not session.qr_token or session.table is None:
            return (
                "Para comenzar, escanea el QR de tu mesa y abre WhatsApp "
                "desde ese enlace.",
                "none",
            )
        if session.handoff_pending:
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
        return str(result["reply"]), "transcript"

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
