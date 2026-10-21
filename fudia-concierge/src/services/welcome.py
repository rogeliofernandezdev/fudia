from __future__ import annotations

from src.domain.models import TableContext
from src.services.carta import Carta, render_carta
from src.services.start_codes import table_label

CLOSING = "¿Qué deseas pedir?"
NONE_AVAILABLE_CLOSING = "Si necesitas ayuda, puedo avisar al personal del local."
# WhatsApp cuts text messages at 4096 characters; keep room for the tail.
MAX_REPLY_CHARS = 3500


def welcome_message(table: TableContext, carta: Carta | None) -> str:
    """Deterministic first reply to a QR start: greeting plus the real carta,
    so the diner sees it without a model call."""
    # Plain WhatsApp text: no markdown, sober icons only (see
    # docs/03_CONVERSATION_FLOW.md, «Formato de los mensajes»).
    greeting = (
        f"🍽️ Bienvenido a {table.organizationName}\n"
        f"📍 {table_label(table.name)}"
    )
    if carta is None:
        # The carta could not be read: the diner can still ask for it.
        return f"{greeting}\n\n{CLOSING}"
    closing = CLOSING if carta.available else NONE_AVAILABLE_CLOSING
    budget = MAX_REPLY_CHARS - len(greeting) - len(closing) - 4
    return f"{greeting}\n\n{render_carta(carta, budget)}\n\n{closing}"
