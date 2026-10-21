from __future__ import annotations

import asyncio
import unicodedata
from dataclasses import dataclass, field
from decimal import Decimal, InvalidOperation

from src.domain.models import ComboGroup, MenuItem, MenuResponse
from src.services.ports import FudiaBackend, FudiaError

ORDERABLE_STATUSES = {"available", "low"}
RECOMMENDED_HEADING = "👨‍🍳 Recomendados de la casa"
UNCATEGORIZED_HEADING = "Otros"
EXTRAS_HEADING = "➕ Extras"
COMBO_ICON = "🍱"
NONE_AVAILABLE = "Hoy no tenemos platos disponibles."

# Course of a category inside a menu: entradas, segundos, bebidas, postres.
STARTER, SOUP, MAIN, DRINK, DESSERT = range(5)

# One food icon and course per category heading, by the words in the category
# name the restaurant chose; the first match wins and anything else is a main
# course with 🍴.
CATEGORY_ICONS: tuple[tuple[tuple[str, ...], str, int], ...] = (
    (("menu", "combo", "promo"), COMBO_ICON, MAIN),
    (("entrada", "piqueo", "ensalada"), "🥗", STARTER),
    (("sopa", "caldo", "crema"), "🍲", SOUP),
    (("marin", "pescado", "marisco", "cevich"), "🐟", MAIN),
    (("pollo", "brasa"), "🍗", MAIN),
    (("parrilla", "carne", "anticucho"), "🥩", MAIN),
    (("pasta", "tallarin"), "🍝", MAIN),
    (("pizza",), "🍕", MAIN),
    (("sandwich", "sanguch", "hamburguesa"), "🥪", MAIN),
    (("desayuno",), "🍳", MAIN),
    (("postre", "dulce", "torta"), "🍰", DESSERT),
    (("cafe", "infusion"), "☕", DRINK),
    (("cerveza", "trago", "coctel", "vino", "bar"), "🍺", DRINK),
    (("bebida", "jugo", "refresco", "gaseosa"), "🥤", DRINK),
    (("segundo", "fondo", "plato", "criollo"), "🍛", MAIN),
)


def _category_kind(name: str) -> tuple[str, int]:
    plain = unicodedata.normalize("NFKD", name.lower()).encode("ascii", "ignore").decode()
    for words, icon, course in CATEGORY_ICONS:
        if any(word in plain for word in words):
            return icon, course
    return "🍴", MAIN


def category_icon(name: str) -> str:
    return _category_kind(name)[0]


def category_course(name: str) -> int:
    return _category_kind(name)[1]


@dataclass
class ComboView:
    item: MenuItem
    groups: list[ComboGroup]


@dataclass
class Carta:
    """The full carta as the diner reads it: only what can be ordered today,
    each combo with what it includes and the dishes that are not already
    offered inside a combo. Sold-out dishes are left out."""

    currency_symbol: str = ""
    combos: list[ComboView] = field(default_factory=list)
    # Dishes outside every combo (the «Extras» when there are combos).
    items: list[MenuItem] = field(default_factory=list)
    included: set[str] = field(default_factory=set)
    # Every orderable product by id, in backend order.
    orderable: dict[str, MenuItem] = field(default_factory=dict)

    @property
    def available(self) -> bool:
        return bool(self.combos or self.items)


def money(symbol: str, value: Decimal) -> str:
    try:
        amount = Decimal(value).quantize(Decimal("0.01"))
    except (InvalidOperation, ValueError):
        amount = Decimal("0.00")
    return f"{symbol} {amount}".strip()


def _available_groups(groups: list[ComboGroup]) -> list[ComboGroup]:
    visible = []
    for group in groups:
        options = [option for option in group.options if option.available]
        if options:
            visible.append(group.model_copy(update={"options": options}))
    return visible


async def build_carta(
    fudia: FudiaBackend, token: str, menu: MenuResponse
) -> Carta:
    orderable = [item for item in menu.items if item.status in ORDERABLE_STATUSES]

    async def detail(item: MenuItem) -> ComboView | None:
        try:
            combo = await fudia.get_combo(token, item.productId)
        except (FudiaError, ValueError):
            # Shown as a plain item: its detail stays one question away.
            return None
        groups = _available_groups(combo.groups)
        return ComboView(item=item, groups=groups) if groups else None

    combo_items = [item for item in orderable if item.isCombo]
    details = await asyncio.gather(*(detail(item) for item in combo_items))
    combos = [view for view in details if view is not None]
    detailed = {view.item.productId for view in combos}
    included = {
        option.productId
        for view in combos
        for group in view.groups
        for option in group.options
    }
    items = [
        item
        for item in orderable
        if item.productId not in detailed and item.productId not in included
    ]
    return Carta(
        currency_symbol=menu.currencySymbol,
        combos=combos,
        items=items,
        included=included,
        orderable={item.productId: item for item in orderable},
    )


def _dish_line(item: MenuItem, symbol: str) -> str:
    return f"- {item.name} ({money(symbol, item.price)})"


def _menu_dish_line(name: str, symbol: str, surcharge: Decimal) -> str:
    """A dish inside a menu: the menu price covers it, so only an extra
    charge for choosing it is shown."""
    line = f"- {name}"
    if surcharge > 0:
        line += f" (+{money(symbol, surcharge)})"
    return line


def _by_category(lines: list[tuple[MenuItem, str]]) -> list[str]:
    """«Categoría:» followed by its dishes, categories in backend order."""
    categories: dict[str, list[str]] = {}
    for item, line in lines:
        heading = (item.categoryName or "").strip() or UNCATEGORIZED_HEADING
        categories.setdefault(f"{category_icon(heading)} {heading}:", []).append(line)
    return [entry for heading, dishes in categories.items() for entry in (heading, *dishes)]


def _combo_section(view: ComboView, carta: Carta) -> list[str]:
    """The combo with every dish it includes, grouped by the dish's category
    in course order (entradas, segundos, bebidas, postres). The combo price
    covers its dishes, so they carry no price of their own."""
    symbol = carta.currency_symbol
    order = {product_id: index for index, product_id in enumerate(carta.orderable)}
    # (course, backend position, heading, line) per dish, once each.
    dishes: dict[str, tuple[int, int, str, str]] = {}
    for group in view.groups:
        for option in group.options:
            if option.productId in dishes:
                continue
            item = carta.orderable.get(option.productId)
            # A dish missing from the menu listing stays under the combo group.
            heading = (
                (item.categoryName or "").strip() or UNCATEGORIZED_HEADING
                if item
                else group.name
            )
            dishes[option.productId] = (
                category_course(heading),
                order.get(option.productId, len(order)),
                f"{category_icon(heading)} {heading}:",
                _menu_dish_line(option.name, symbol, option.surcharge),
            )
    categories: dict[str, list[str]] = {}
    for _, _, heading, line in sorted(dishes.values()):
        categories.setdefault(heading, []).append(line)
    lines = [f"{COMBO_ICON} {view.item.name} ({money(symbol, view.item.price)})"]
    lines += [entry for heading, rows in categories.items() for entry in (heading, *rows)]
    return lines


def _sections(carta: Carta) -> list[list[str]]:
    """Each combo with all the dishes it includes, then «Extras» with the
    dishes outside any combo by category. Without combos: recommended
    dishes first, then each category."""
    symbol = carta.currency_symbol
    if carta.combos:
        sections = [_combo_section(view, carta) for view in carta.combos]
        extras = _by_category([(item, _dish_line(item, symbol)) for item in carta.items])
        if extras:
            sections.append([EXTRAS_HEADING, *extras])
        return sections

    recommended = [_dish_line(item, symbol) for item in carta.items if item.recommended]
    sections = [[f"{RECOMMENDED_HEADING}:", *recommended]] if recommended else []
    rest = _by_category(
        [(item, _dish_line(item, symbol)) for item in carta.items if not item.recommended]
    )
    # One section per category, so a blank line separates them.
    for line in rest:
        if not line.startswith("- "):
            sections.append([line])
        else:
            sections[-1].append(line)
    return sections


def render_carta(carta: Carta, max_chars: int) -> str:
    """The carta as plain WhatsApp text, within `max_chars`. Shared by the
    QR welcome and by search_menu, so both show it the same way."""
    if not carta.available:
        return NONE_AVAILABLE

    sections = _sections(carta)
    total = sum(1 for section in sections for line in section if line.startswith("- "))
    # Room for the "more" note.
    budget = max_chars - 120
    pieces: list[str] = []
    length = 0
    shown = 0
    for section in sections:
        for index, line in enumerate(section):
            piece = ("\n\n" if index == 0 else "\n") + line if pieces else line
            if length + len(piece) > budget:
                break
            pieces.append(piece)
            length += len(piece)
            shown += line.startswith("- ")
        else:
            continue
        break
    # A heading left without dishes at the cut is dropped.
    while pieces and not pieces[-1].lstrip("\n").startswith("- "):
        pieces.pop()

    text = "".join(pieces)
    hidden = total - shown
    if hidden:
        more = "opción más" if hidden == 1 else "opciones más"
        text += (
            f"\n\nHay {hidden} {more}. Escríbeme una categoría o un plato "
            "y te muestro el resto."
        )
    return text
