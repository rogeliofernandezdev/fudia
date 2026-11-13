from __future__ import annotations

from src.domain.models import TableContext
from src.services.carta import NONE_AVAILABLE, Carta, render_carta
from src.services.start_codes import service_label

CLOSING = "¿Qué deseas pedir?"
# Web chat: points at the shortcut of the page instead of listing the carta.
WEB_CLOSING = "Toca «Carta» para ver los platos o escríbeme lo que deseas pedir."
NONE_AVAILABLE_CLOSING = "Si necesitas ayuda, puedo avisar al personal del local."
# A new diner at the counter is asked for the name the cashier calls out.
COUNTER_NAME_CLOSING = "Para empezar, ¿a nombre de quién va el pedido?"
COUNTER_NOTE = "Pide aquí, recoge en el mostrador y paga en caja."
DELIVERY_NOTE = "Pide aquí para delivery o para recoger en el local."
# WhatsApp cuts text messages at 4096 characters; keep room for the tail.
MAX_REPLY_CHARS = 3500


def _web_welcome(table: TableContext, carta: Carta | None, customer_name: str) -> str:
    """Web chat greeting. The page header already shows the restaurant, the
    venue and the table, and «Carta» opens the dishes with photos, so nothing
    of that is repeated here: the diner asks for the carta when they want it."""
    counter = table.serviceType == "counter"
    hello = f"Hola, {customer_name}. " if customer_name else ""
    greeting = f"Bienvenido a {table.organizationName}."
    if table.serviceType == "delivery":
        greeting = f"{greeting}\n{hello}{DELIVERY_NOTE}"
    elif counter:
        greeting = f"{greeting}\n{hello}{COUNTER_NOTE}"
    if carta is not None and not carta.available:
        return f"{greeting}\n\n{NONE_AVAILABLE}\n\n{NONE_AVAILABLE_CLOSING}"
    closing = COUNTER_NAME_CLOSING if counter and not customer_name else WEB_CLOSING
    return f"{greeting}\n\n{closing}"


def welcome_message(
    table: TableContext,
    carta: Carta | None,
    customer_name: str = "",
    *,
    web: bool = False,
) -> str:
    """Deterministic first reply to a QR start: greeting plus the real carta,
    so the diner sees it without a model call. On the web chat (`web`) the
    greeting is short and the carta is left to the «Carta» button."""
    if web:
        return _web_welcome(table, carta, customer_name)
    # Plain WhatsApp text: no markdown, sober icons only (see
    # docs/03_CONVERSATION_FLOW.md, «Formato de los mensajes»).
    counter = table.serviceType == "counter"
    greeting = (
        f"🍽️ Bienvenido a {table.organizationName}\n"
        f"📍 {service_label(table.name, table.serviceType)}"
    )
    if table.serviceType == "delivery":
        # Enlace de delivery del local: no hay mesa ni mostrador.
        hello = f"Hola, {customer_name}. " if customer_name else ""
        greeting = (
            f"🍽️ Bienvenido a {table.organizationName}\n"
            f"🛵 {table.locationName}\n\n{hello}{DELIVERY_NOTE}"
        )
    if counter:
        hello = f"Hola, {customer_name}. " if customer_name else ""
        greeting = f"{greeting}\n\n{hello}{COUNTER_NOTE}"
    ask_name = counter and not customer_name
    if carta is None:
        # The carta could not be read: the diner can still ask for it.
        return f"{greeting}\n\n{COUNTER_NAME_CLOSING if ask_name else CLOSING}"
    closing = CLOSING if carta.available else NONE_AVAILABLE_CLOSING
    if ask_name and carta.available:
        closing = COUNTER_NAME_CLOSING
    budget = MAX_REPLY_CHARS - len(greeting) - len(closing) - 4
    return f"{greeting}\n\n{render_carta(carta, budget)}\n\n{closing}"
