from __future__ import annotations

from decimal import Decimal, InvalidOperation

from src.domain.models import MenuItem, MenuResponse, TableContext
from src.services.start_codes import table_label

ORDERABLE_STATUSES = {"available", "low"}
RECOMMENDED_HEADING = "⭐ Recomendados de la casa"
UNCATEGORIZED_HEADING = "Otros"
CLOSING = "¿Qué deseas pedir?"
# WhatsApp cuts text messages at 4096 characters; keep room for the tail.
MAX_REPLY_CHARS = 3500


def _price(symbol: str, value: Decimal) -> str:
    try:
        amount = Decimal(value).quantize(Decimal("0.01"))
    except (InvalidOperation, ValueError):
        amount = Decimal("0.00")
    return f"{symbol} {amount}".strip()


def _sections(items: list[MenuItem]) -> list[tuple[str, list[MenuItem]]]:
    """Recommended first, then categories in the order the backend sends."""
    recommended = [item for item in items if item.recommended]
    sections: dict[str, list[MenuItem]] = {}
    for item in items:
        if item.recommended:
            continue
        heading = (item.categoryName or "").strip() or UNCATEGORIZED_HEADING
        sections.setdefault(heading, []).append(item)
    ordered = [(RECOMMENDED_HEADING, recommended)] if recommended else []
    return ordered + list(sections.items())


def welcome_message(table: TableContext, menu: MenuResponse | None) -> str:
    """Deterministic first reply to a QR start: greeting plus the real menu,
    so the diner sees the carta without a model call."""
    greeting = (
        f"¡Bienvenido a {table.organizationName}! 👋\n"
        f"Estás en la {table_label(table.name)}."
    )
    items = [
        item
        for item in (menu.items if menu else [])
        if item.status in ORDERABLE_STATUSES
    ]
    if not items:
        return f"{greeting}\n\n{CLOSING}"

    symbol = menu.currencySymbol if menu else ""
    blocks: list[str] = []
    # Greeting, closing, separators and room for the "more dishes" note.
    budget = MAX_REPLY_CHARS - len(greeting) - len(CLOSING) - 120
    shown = 0
    full = False
    for heading, section in _sections(items):
        block = f"*{heading}*"
        for item in section:
            line = f"\n• {item.name} · {_price(symbol, item.price)}"
            if len(block) + len(line) + 2 > budget:
                full = True
                break
            block += line
            shown += 1
        if "\n" in block:
            blocks.append(block)
            budget -= len(block) + 2
        if full:
            break

    hidden = len(items) - shown
    body = "\n\n".join(blocks)
    if hidden:
        more = "plato más" if hidden == 1 else "platos más"
        body += (
            f"\n\n…y {hidden} {more}. Escríbeme una categoría o un plato "
            "y te muestro el resto."
        )
    return f"{greeting}\n\n{body}\n\n{CLOSING}"
