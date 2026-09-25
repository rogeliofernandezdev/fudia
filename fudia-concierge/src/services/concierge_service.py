from __future__ import annotations

import asyncio
import re
from collections.abc import Awaitable, Callable
from typing import Any, cast
from uuid import uuid4

from src.ai.contracts import ConciergeRouter, ConversationAgent
from src.domain.intents import AgentIntent, ConciergeRoute
from src.domain.models import ChatMessage, ConversationSession
from src.graph.app import build_graph
from src.infrastructure.concurrency import (
    ConversationLock,
    MemoryConversationLock,
    conversation_identity,
)
from src.infrastructure.fudia_client import FudiaClient, FudiaError
from src.infrastructure.rate_limit import AllowAllRateLimiter, RateLimiter
from src.infrastructure.state_store import ConversationStore
from src.metrics import RATE_LIMITED, SCOPE_DECISIONS
from src.services.tool_registry import ToolRegistry

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


def _phone_digits(value: str) -> str:
    return "".join(
        character
        for character in value
        if character.isdigit()
    )



class ConciergeService:
    def __init__(
        self,
        store: ConversationStore,
        fudia: FudiaClient,
        router: ConciergeRouter | ConversationAgent,
        agents: dict[AgentIntent, ConversationAgent] | None = None,
        conversation_lock: ConversationLock | None = None,
        rate_limiter: RateLimiter | None = None,
        max_message_chars: int = 2000,
        lock_heartbeat_seconds: float = 5.0,
    ) -> None:
        self.store = store
        self.fudia = fudia
        self.conversation_lock = (
            conversation_lock or MemoryConversationLock()
        )
        self.rate_limiter = rate_limiter or AllowAllRateLimiter()
        self.max_message_chars = max_message_chars
        self.lock_heartbeat_seconds = lock_heartbeat_seconds

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
                [ConversationSession, str],
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
        [ConversationSession, str],
        Awaitable[str],
    ]:
        async def process(
            session: ConversationSession,
            user_message: str,
        ) -> str:
            registry = ToolRegistry(
                self.fudia,
                session,
                user_message,
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

            reply = await self.agents[intent].reply(
                session,
                user_message,
                execute,
            )
            session.messages.extend(
                [
                    ChatMessage(
                        role="user",
                        content=user_message,
                    ),
                    ChatMessage(
                        role="assistant",
                        content=reply,
                    ),
                ]
            )
            session.messages = session.messages[-20:]
            await self.store.save(session)
            return reply

        return process

    async def handle_message(
        self,
        phone: str,
        text: str,
        recipient_phone: str = "",
        channel_id: str = "",
    ) -> str:
        if len(text) > self.max_message_chars:
            return (
                "Tu mensaje es demasiado largo. "
                "Envíame solo lo necesario para gestionar tu pedido."
            )

        channel_key = (
            channel_id.strip()
            or _phone_digits(recipient_phone)
            or "default"
        )
        identity = conversation_identity(phone, channel_key)
        if not await self.rate_limiter.allow(identity):
            RATE_LIMITED.inc()
            return (
                "Has enviado muchos mensajes en poco tiempo. "
                "Continúa con tu pedido dentro de un momento."
            )

        lock_token = await self.conversation_lock.acquire(identity)
        heartbeat = asyncio.create_task(
            self._keep_lock_alive(
                identity,
                lock_token,
            )
        )
        try:
            return await self._handle_locked_message(
                phone,
                text,
                recipient_phone,
                channel_key,
            )
        finally:
            heartbeat.cancel()
            try:
                await heartbeat
            except asyncio.CancelledError:
                pass
            await self.conversation_lock.release(
                identity,
                lock_token,
            )

    async def _keep_lock_alive(
        self,
        identity: str,
        token: str,
    ) -> None:
        while True:
            await asyncio.sleep(
                self.lock_heartbeat_seconds
            )
            refreshed = (
                await self.conversation_lock.refresh(
                    identity,
                    token,
                )
            )
            if not refreshed:
                return

    async def _handle_locked_message(
        self,
        phone: str,
        text: str,
        recipient_phone: str,
        channel_key: str,
    ) -> str:
        session = (
            await self.store.get(phone, channel_key)
            or ConversationSession(
                phone=phone,
                channel_key=channel_key,
            )
        )
        token = extract_qr_token(text)

        if token:
            try:
                table = await self.fudia.resolve_table(token)
            except FudiaError:
                return (
                    "El QR no está activo o ya no corresponde a una mesa. "
                    "Pide ayuda al personal del restaurante."
                )
            if not table.conciergeEnabled:
                return (
                    "Fudia Concierge no está habilitado para este restaurante "
                    "en este momento. Pide ayuda al personal."
                )
            session.channel_key = channel_key
            session.qr_token = token
            session.conversation_id = uuid4().hex
            session.table = table
            session.cart = []
            session.awaiting_confirmation = False
            session.pending_order_request_id = None
            session.handoff_pending = False
            session.messages = []
            reply = (
                f"Hola, estás en {table.organizationName}, mesa {table.name}. "
                "¿Qué deseas pedir?"
            )
            session.messages.extend(
                [
                    ChatMessage(
                        role="user",
                        content=(
                            "Inicié mi pedido desde el QR de la mesa."
                        ),
                    ),
                    ChatMessage(
                        role="assistant",
                        content=reply,
                    ),
                ]
            )
            await self.store.save(session)
            return reply

        if not session.qr_token or session.table is None:
            return (
                "Para comenzar, escanea el QR de tu mesa y abre WhatsApp "
                "desde ese enlace."
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
                    "No pude verificar todavía si fue atendida."
                )
            if str(handoff.get("status", "")) == "pending":
                return (
                    "El personal del local ya fue avisado. "
                    "Espera un momento mientras se acercan a tu mesa."
                )
            session.handoff_pending = False
            await self.store.save(session)

        if obviously_out_of_scope(text):
            SCOPE_DECISIONS.labels(
                decision="out_of_scope"
            ).inc()
            return OFF_SCOPE_REPLY

        result = await self.graph.ainvoke(
            {
                "session": session,
                "user_message": text,
                "intent": "order",
                "reply": "",
            }
        )
        return str(result["reply"])
