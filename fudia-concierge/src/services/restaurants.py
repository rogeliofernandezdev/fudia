"""Deterministic restaurant selection; models never resolve tenant ids."""

from __future__ import annotations

import re
from typing import Protocol

from src.domain.whatsapp import RestaurantOption, RestaurantPage, RestaurantSelection, VerifiedCustomer


class RestaurantDirectory(Protocol):
    async def restaurants(
        self,
        channel_key: str,
        identity: VerifiedCustomer,
        only_visited: bool = True,
        offset: int = 0,
    ) -> RestaurantPage: ...
    async def restaurant_context(self, channel_key: str, location_id: str) -> RestaurantOption: ...


def option_number(text: str, count: int) -> int | None:
    match = re.fullmatch(r"\s*([1-9][0-9]?)[.)]?\s*", text)
    if match is None:
        return None
    number = int(match.group(1))
    return number - 1 if number <= count else None


def selection_text(selection: RestaurantSelection) -> str:
    lines = ["🍽️ ¿De qué restaurante deseas pedir hoy?", ""]
    for index, option in enumerate(selection.options, 1):
        venue = " · ".join(part for part in (option.organizationName, option.locationName) if part)
        lines.append(f"{index}. {venue}")
    lines.extend(["", "Responde con el número de tu elección."])
    if selection.has_more:
        lines.append("Escribe «más» para ver otros restaurantes.")
    if selection.offset:
        lines.append("Escribe «anteriores» para volver.")
    return "\n".join(lines)
