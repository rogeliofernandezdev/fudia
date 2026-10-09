from __future__ import annotations

from typing import Any

from src.services.tooling import ToolContext


class MenuTools:
    def __init__(self, context: ToolContext) -> None:
        self.context = context

    async def execute(
        self,
        name: str,
        args: dict[str, Any],
    ) -> dict[str, Any]:
        if name == "search_menu":
            menu = await self.context.fudia.search_menu(
                self.context.token(),
                query=str(args.get("query", "")),
            )
            return {
                "ok": True,
                "currencySymbol": menu.currencySymbol,
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
