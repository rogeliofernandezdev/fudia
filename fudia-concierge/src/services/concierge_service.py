from __future__ import annotations

import re
from uuid import uuid4

from src.domain.models import ChatMessage, ConversationSession
from src.graph.app import build_graph
from src.infrastructure.fudia_client import FudiaClient, FudiaError
from src.infrastructure.openai_adapter import AssistantEngine
from src.infrastructure.state_store import ConversationStore
from src.services.tools import ConciergeTools

QR_PATTERN = re.compile(r"(?i)\bFUDIA:([a-f0-9]{32})\b")


def extract_qr_token(text: str) -> str | None:
    match = QR_PATTERN.search(text)
    return match.group(1).lower() if match else None


class ConciergeService:
    def __init__(
        self,
        store: ConversationStore,
        fudia: FudiaClient,
        engine: AssistantEngine,
    ) -> None:
        self.store = store
        self.fudia = fudia
        self.engine = engine
        self.graph = build_graph(self._process_ready_session)

    async def handle_message(self, phone: str, text: str) -> str:
        session = await self.store.get(phone) or ConversationSession(phone=phone)
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
                    "Fudia Concierge no está disponible en este local en este momento. "
                    "Pide ayuda al personal del restaurante."
                )
            session.qr_token = token
            session.conversation_id = uuid4().hex
            session.table = table
            session.cart = []
            session.awaiting_confirmation = False
            session.messages = []
            reply = (
                f"Hola, estás en {table.organizationName}, mesa {table.name}. "
                "¿Qué deseas pedir?"
            )
            session.messages.extend(
                [
                    ChatMessage(
                        role="user",
                        content="Inicié mi pedido desde el QR de la mesa.",
                    ),
                    ChatMessage(role="assistant", content=reply),
                ]
            )
            await self.store.save(session)
            return reply

        if not session.qr_token or session.table is None:
            return (
                "Para comenzar, escanea el QR de tu mesa y abre WhatsApp "
                "desde ese enlace."
            )

        result = await self.graph.ainvoke(
            {"session": session, "user_message": text, "reply": ""}
        )
        return str(result["reply"])

    async def _process_ready_session(
        self,
        session: ConversationSession,
        user_message: str,
    ) -> str:
        tools = ConciergeTools(self.fudia, session, user_message)
        reply = await self.engine.reply(session, user_message, tools.execute)
        session.messages.extend(
            [
                ChatMessage(role="user", content=user_message),
                ChatMessage(role="assistant", content=reply),
            ]
        )
        session.messages = session.messages[-20:]
        await self.store.save(session)
        return reply
