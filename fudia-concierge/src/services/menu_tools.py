from __future__ import annotations

from typing import Any

from src.services.carta import ORDERABLE_STATUSES, build_carta, render_carta
from src.services.tooling import ToolContext

# Leaves room in a WhatsApp message for an intro line and a closing question.
CARTA_TEXT_CHARS = 3300


def unavailable_carta_reply(result: dict[str, Any]) -> str:
    """Render the live carta instead of letting the model select a few alternatives."""
    if result.get("ok") is not True or result.get("unavailable") is not True:
        return ""
    text = result.get("cartaText")
    if not isinstance(text, str) or not text:
        return ""
    if result.get("noneAvailable") is True:
        return f"{text}\n\nSi necesitas ayuda, puedo avisar al personal del local."
    return (
        "Lo que buscas no está disponible hoy. Te comparto nuestra carta:\n\n"
        f"{text}\n\n¿Qué deseas pedir?"
    )


class MenuTools:
    def __init__(self, context: ToolContext) -> None:
        self.context = context

    async def execute(
        self,
        name: str,
        args: dict[str, Any],
    ) -> dict[str, Any]:
        if name == "search_menu":
            query = str(args.get("query", "")).strip()
            menu = await self.context.fudia.search_menu(
                self.context.token(),
                query=query,
            )
            matched = bool(menu.items) or not query
            unavailable = bool(query) and not any(
                item.status in ORDERABLE_STATUSES for item in menu.items
            )
            if unavailable:
                # Missing or sold-out results must lead back to the live carta.
                menu = await self.context.fudia.search_menu(
                    self.context.token(),
                    query="",
                )
            # The full carta carries each combo with what it includes, so the
            # agent shows it once instead of repeating its dishes.
            full = not query or unavailable
            carta = (
                await build_carta(self.context.fudia, self.context.token(), menu)
                if full
                else None
            )
            included = carta.included if carta else set()
            return {
                "ok": True,
                "query": query,
                "matched": matched,
                "unavailable": unavailable,
                "menuEmpty": not menu.items,
                "noneAvailable": bool(carta and not carta.available),
                "currencySymbol": menu.currencySymbol,
                # Ready to send: grouped like the QR welcome, without sold out.
                "cartaText": render_carta(carta, CARTA_TEXT_CHARS) if carta else "",
                "combos": [
                    {
                        "productId": view.item.productId,
                        "name": view.item.name,
                        "price": str(view.item.price),
                        "groups": [
                            {
                                "name": group.name,
                                "minSelections": group.minSelections,
                                "maxSelections": group.maxSelections,
                                "options": [
                                    {
                                        "productId": option.productId,
                                        "name": option.name,
                                        "surcharge": str(option.surcharge),
                                    }
                                    for option in group.options
                                ],
                            }
                            for group in view.groups
                        ],
                    }
                    for view in (carta.combos if carta else [])
                ],
                "items": [
                    {
                        "productId": item.productId,
                        "name": item.name,
                        "description": item.description,
                        "price": str(item.price),
                        "categoryName": item.categoryName,
                        "status": item.status,
                        "isCombo": item.isCombo,
                        "hasModifiers": item.hasModifiers,
                        "recommended": item.recommended,
                        "includedInCombo": item.productId in included,
                    }
                    for item in menu.items
                ],
            }

        if name == "get_modifier_options":
            product_id = str(args.get("productId", "")).strip()
            if not product_id:
                return {"ok": False, "code": "invalid_product"}
            item = await self.context.fudia.get_product(
                self.context.token(),
                product_id,
            )
            if item is None:
                return {"ok": False, "code": "product_not_found"}
            if item.isCombo:
                return {
                    "ok": False,
                    "code": "combo_requires_options",
                    "name": item.name,
                }
            config = await self.context.fudia.get_modifiers(
                self.context.token(),
                product_id,
            )
            return {
                "ok": True,
                "product": {
                    "productId": item.productId,
                    "name": item.name,
                    "price": str(item.price),
                },
                "groups": [
                    {
                        "groupId": group.id,
                        "name": group.name,
                        "required": group.required,
                        "minSelections": group.minSelections,
                        "maxSelections": group.maxSelections,
                        "options": [
                            {
                                "optionId": option.id,
                                "name": option.name,
                                "surcharge": str(option.surcharge),
                            }
                            for option in group.options
                        ],
                    }
                    for group in config.groups
                ],
            }

        if name == "get_combo_options":
            product_id = str(args.get("productId", "")).strip()
            if not product_id:
                return {"ok": False, "code": "invalid_combo"}
            combo = await self.context.fudia.get_combo(
                self.context.token(),
                product_id,
            )
            return {
                "ok": True,
                "combo": {
                    "productId": combo.id,
                    "name": combo.name,
                    "price": str(combo.price),
                    "groups": [
                        {
                            "groupId": group.id,
                            "name": group.name,
                            "required": group.required,
                            "minSelections": group.minSelections,
                            "maxSelections": group.maxSelections,
                            "options": [
                                {
                                    "productId": option.productId,
                                    "name": option.name,
                                    "surcharge": str(option.surcharge),
                                    "available": option.available,
                                }
                                for option in group.options
                            ],
                        }
                        for group in combo.groups
                    ],
                },
            }

        return {"ok": False, "code": "unknown_menu_tool"}
