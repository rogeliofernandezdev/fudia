from __future__ import annotations

import asyncio
from dataclasses import dataclass, field
from decimal import Decimal, InvalidOperation

from src.domain.models import ComboGroup, MenuItem, MenuResponse
from src.services.ports import FudiaBackend, FudiaError

ORDERABLE_STATUSES = {"available", "low"}


@dataclass
class ComboView:
    item: MenuItem
    groups: list[ComboGroup]


@dataclass
class Carta:
    """The full carta as the diner reads it: each combo with what it
    includes, and the dishes that are not already offered inside a combo."""

    currency_symbol: str = ""
    combos: list[ComboView] = field(default_factory=list)
    items: list[MenuItem] = field(default_factory=list)
    included: set[str] = field(default_factory=set)


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
    )


def combo_heading(view: ComboView, symbol: str) -> str:
    return f"{view.item.name} ({money(symbol, view.item.price)}):"


def combo_lines(view: ComboView, symbol: str) -> list[str]:
    """One line per group: «- Segundo: Ají de gallina o Lomo saltado»."""
    lines = []
    for group in view.groups:
        names = [
            option.name
            + (
                f" (+{money(symbol, option.surcharge)})"
                if option.surcharge > 0
                else ""
            )
            for option in group.options
        ]
        joined = (
            names[0]
            if len(names) == 1
            else f"{', '.join(names[:-1])} o {names[-1]}"
        )
        lines.append(f"- {group.name}: {joined}")
    return lines
