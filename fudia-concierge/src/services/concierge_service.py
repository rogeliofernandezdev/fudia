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


def extract_qr_token(text: str) -> str | None:
    match = QR_PATTERN.search(text)
    return match.group(1).lower() if match else None


def obviously_out_of_scope(text: str) -> bool:
    value = text.strip()
    if not value:
        return False
    return bool(
        ARITHMETIC_ONLY_PATTERN.fullmatch(value)
        or MATH_TOPIC_PATTERN.search(value)
    )


def _phone_digits(value: str) -> str:
    return "".join(character for character in value if character.isdigit())


def _recipient_matches_configured(recipient: str, configured: str) -> bool:
    if not recipient or not configured:
        return True
    return _phone_digits(recipient) == _phone_digits(configured)


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

    async def handle_message(
        self, phone: str, text: str, recipient_phone: str = ""
    ) -> str:
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
            if not _recipient_matches_configured(
                recipient_phone, table.whatsappPhone
            ):
                return (
                    "Este QR no corresponde al número de WhatsApp que recibió "
                    "el mensaje. Vuelve a abrir WhatsApp desde el QR de tu mesa."
                )
            session.qr_token = token
            session.conversation_id = uuid4().hex
            session.table = table
            session.cart = []
            session.awaiting_confirmation = False
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
        if not _recipient_matches_configured(
            recipient_phone, session.table.whatsappPhone
        ):
            return (
                "Esta conversación pertenece a otro número de WhatsApp del "
                "restaurante. Vuelve a abrirla desde el QR de tu mesa."
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
            return OFF_SCOPE_REPLY

        scope_checker = getattr(self.engine, "is_in_scope", None)
        if callable(scope_checker):
            try:
                if not await scope_checker(session, text):
                    return OFF_SCOPE_REPLY
            except Exception:
                return (
                    "No pude validar tu solicitud en este momento. "
                    "Intenta nuevamente con algo relacionado con tu pedido."
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
