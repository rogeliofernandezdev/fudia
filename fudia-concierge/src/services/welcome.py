from __future__ import annotations

from src.domain.models import MenuItem, TableContext
from src.services.carta import Carta, combo_heading, combo_lines, money
from src.services.start_codes import table_label

RECOMMENDED_HEADING = "Recomendados de la casa"
UNCATEGORIZED_HEADING = "Otros"
CLOSING = "¿Qué deseas pedir?"
# WhatsApp cuts text messages at 4096 characters; keep room for the tail.
MAX_REPLY_CHARS = 3500


def _sections(carta: Carta) -> list[tuple[str, list[str]]]:
    """Recommended dishes, then each combo with what it includes, then the
    remaining categories in the order the backend sends. A dish offered
    inside a combo is not repeated on its own."""
    symbol = carta.currency_symbol

    def line(item: MenuItem) -> str:
        return f"- {item.name}: {money(symbol, item.price)}"

    recommended = [line(item) for item in carta.items if item.recommended]
    categories: dict[str, list[str]] = {}
    for item in carta.items:
        if item.recommended:
            continue
        heading = (item.categoryName or "").strip() or UNCATEGORIZED_HEADING
        categories.setdefault(f"{heading}:", []).append(line(item))

    sections = [(f"{RECOMMENDED_HEADING}:", recommended)] if recommended else []
    # A combo is one entry: it is shown whole or not at all.
    sections += [
        (combo_heading(view, symbol), ["\n".join(combo_lines(view, symbol))])
        for view in carta.combos
    ]
    return sections + list(categories.items())


def welcome_message(table: TableContext, carta: Carta | None) -> str:
    """Deterministic first reply to a QR start: greeting plus the real carta,
    so the diner sees it without a model call."""
    # Plain WhatsApp text: no markdown, sober icons only (see
    # docs/03_CONVERSATION_FLOW.md, «Formato de los mensajes»).
    greeting = (
        f"🍽️ Bienvenido a {table.organizationName}\n"
        f"📍 {table_label(table.name)}"
    )
    sections = _sections(carta) if carta else []
    entries = sum(len(lines) for _, lines in sections)
    if not entries:
        return f"{greeting}\n\n{CLOSING}"

    blocks: list[str] = []
    # Greeting, closing, separators and room for the "more dishes" note.
    budget = MAX_REPLY_CHARS - len(greeting) - len(CLOSING) - 120
    shown = 0
    full = False
    for heading, lines in sections:
        block = heading
        for entry in lines:
            if len(block) + len(entry) + 3 > budget:
                full = True
                break
            block += f"\n{entry}"
            shown += 1
        if block != heading:
            blocks.append(block)
            budget -= len(block) + 2
        if full:
            break

    hidden = entries - shown
    body = "\n\n".join(blocks)
    if hidden:
        more = "opción más" if hidden == 1 else "opciones más"
        body += (
            f"\n\nHay {hidden} {more}. Escríbeme una categoría o un plato "
            "y te muestro el resto."
        )
    return f"{greeting}\n\n{body}\n\n{CLOSING}"
