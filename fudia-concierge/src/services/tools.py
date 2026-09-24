from __future__ import annotations

import re
import unicodedata
from decimal import Decimal
from typing import Any

from src.domain.models import (
    CartLine,
    CartModifier,
    CartSelection,
    ConversationSession,
)
from src.infrastructure.fudia_client import FudiaClient, FudiaError


def _normalized(text: str) -> str:
    value = unicodedata.normalize("NFKD", text.lower())
    value = "".join(ch for ch in value if not unicodedata.combining(ch))
    value = re.sub(r"[^a-z0-9 ]+", " ", value)
    return " ".join(value.split())


def explicit_confirmation(text: str) -> bool:
    value = _normalized(text)
    if not value or value.startswith("no"):
        return False
    exact = {
        "si",
        "confirmo",
        "confirmar",
        "de acuerdo",
        "ok",
        "dale",
        "haz el pedido",
        "hacer el pedido",
        "envia el pedido",
        "enviar el pedido",
    }
    return value in exact or value.startswith("si confirm")


class ConciergeTools:
    def __init__(
        self,
        fudia: FudiaClient,
        session: ConversationSession,
        user_message: str,
    ) -> None:
        self.fudia = fudia
        self.session = session
        self.user_message = user_message

    def _token(self) -> str:
        if not self.session.qr_token:
            raise RuntimeError("La sesión no tiene QR.")
        return self.session.qr_token

    def _cart_payload(self) -> dict[str, Any]:
        total = Decimal("0")
        items: list[dict[str, Any]] = []
        for line in self.session.cart:
            line_total = line.unit_price * Decimal(str(line.quantity))
            total += line_total
            items.append(
                {
                    "productId": line.product_id,
                    "name": line.name,
                    "quantity": line.quantity,
                    "unitPrice": str(line.unit_price),
                    "lineTotal": str(line_total.quantize(Decimal("0.01"))),
                    "note": line.note,
                    "itemType": line.item_type,
                    "selections": [
                        {
                            "groupId": selection.group_id,
                            "groupName": selection.group_name,
                            "productId": selection.product_id,
                            "name": selection.name,
                            "surcharge": str(selection.surcharge),
                        }
                        for selection in line.selections
                    ],
                    "modifiers": [
                        {
                            "groupId": modifier.group_id,
                            "groupName": modifier.group_name,
                            "optionId": modifier.option_id,
                            "name": modifier.name,
                            "surcharge": str(modifier.surcharge),
                        }
                        for modifier in line.modifiers
                    ],
                }
            )
        return {"items": items, "total": str(total.quantize(Decimal("0.01")))}

    async def execute(self, name: str, args: dict[str, Any]) -> dict[str, Any]:
        try:
            if name == "search_menu":
                menu = await self.fudia.search_menu(
                    self._token(), query=str(args.get("query", ""))
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

            if name == "add_item":
                product_id = str(args.get("productId", "")).strip()
                quantity = float(args.get("quantity", 0))
                note = str(args.get("note", "")).strip()
                if not product_id or quantity <= 0:
                    return {"ok": False, "code": "invalid_item"}

                item = await self.fudia.get_product(self._token(), product_id)
                if item is None:
                    return {"ok": False, "code": "product_not_found"}
                if item.status not in {"available", "low"}:
                    return {"ok": False, "code": "product_unavailable", "name": item.name}
                if item.isCombo:
                    return {
                        "ok": False,
                        "code": "combo_requires_options",
                        "name": item.name,
                    }
                if item.hasModifiers:
                    return {
                        "ok": False,
                        "code": "modifier_review_required",
                        "name": item.name,
                    }

                existing = next(
                    (line for line in self.session.cart if line.product_id == product_id),
                    None,
                )
                if existing:
                    existing.quantity += quantity
                    if note:
                        existing.note = note
                    existing.unit_price = item.price
                    existing.name = item.name
                else:
                    self.session.cart.append(
                        CartLine(
                            product_id=item.productId,
                            name=item.name,
                            quantity=quantity,
                            unit_price=item.price,
                            note=note,
                        )
                    )
                self.session.awaiting_confirmation = False
                return {"ok": True, "cart": self._cart_payload()}

            if name == "get_modifier_options":
                product_id = str(args.get("productId", "")).strip()
                if not product_id:
                    return {"ok": False, "code": "invalid_product"}
                item = await self.fudia.get_product(self._token(), product_id)
                if item is None:
                    return {"ok": False, "code": "product_not_found"}
                if item.isCombo:
                    return {
                        "ok": False,
                        "code": "combo_requires_options",
                        "name": item.name,
                    }
                config = await self.fudia.get_modifiers(
                    self._token(), product_id
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

            if name == "add_modified_item":
                product_id = str(args.get("productId", "")).strip()
                quantity = float(args.get("quantity", 0))
                note = str(args.get("note", "")).strip()
                raw_modifiers = args.get("modifiers", [])
                if (
                    not product_id
                    or quantity <= 0
                    or not isinstance(raw_modifiers, list)
                ):
                    return {"ok": False, "code": "invalid_item"}

                item = await self.fudia.get_product(
                    self._token(), product_id
                )
                if item is None:
                    return {"ok": False, "code": "product_not_found"}
                if item.status not in {"available", "low"}:
                    return {
                        "ok": False,
                        "code": "product_unavailable",
                        "name": item.name,
                    }
                if item.isCombo:
                    return {
                        "ok": False,
                        "code": "combo_requires_options",
                        "name": item.name,
                    }

                config = await self.fudia.get_modifiers(
                    self._token(), product_id
                )
                groups_by_id = {group.id: group for group in config.groups}
                selected_by_group: dict[str, list[str]] = {}
                seen: set[tuple[str, str]] = set()
                for raw in raw_modifiers:
                    if not isinstance(raw, dict):
                        return {
                            "ok": False,
                            "code": "invalid_modifier_selection",
                        }
                    group_id = str(raw.get("groupId", "")).strip()
                    option_id = str(raw.get("optionId", "")).strip()
                    key = (group_id, option_id)
                    if not group_id or not option_id or key in seen:
                        return {
                            "ok": False,
                            "code": "invalid_modifier_selection",
                        }
                    if group_id not in groups_by_id:
                        return {
                            "ok": False,
                            "code": "invalid_modifier_selection",
                        }
                    seen.add(key)
                    selected_by_group.setdefault(group_id, []).append(
                        option_id
                    )

                modifiers: list[CartModifier] = []
                surcharge_total = Decimal("0")
                for group in config.groups:
                    selected = selected_by_group.get(group.id, [])
                    minimum = max(
                        group.minSelections,
                        1 if group.required else 0,
                    )
                    if (
                        len(selected) < minimum
                        or len(selected) > group.maxSelections
                    ):
                        return {
                            "ok": False,
                            "code": "modifier_group_incomplete",
                            "group": group.name,
                            "minSelections": minimum,
                            "maxSelections": group.maxSelections,
                        }
                    options = {
                        option.id: option for option in group.options
                    }
                    for option_id in selected:
                        option = options.get(option_id)
                        if option is None:
                            return {
                                "ok": False,
                                "code": "invalid_modifier_selection",
                                "group": group.name,
                            }
                        modifiers.append(
                            CartModifier(
                                group_id=group.id,
                                group_name=group.name,
                                option_id=option.id,
                                name=option.name,
                                surcharge=option.surcharge,
                            )
                        )
                        surcharge_total += option.surcharge

                self.session.cart.append(
                    CartLine(
                        product_id=item.productId,
                        name=item.name,
                        quantity=quantity,
                        unit_price=item.price + surcharge_total,
                        note=note,
                        item_type="product",
                        modifiers=modifiers,
                    )
                )
                self.session.awaiting_confirmation = False
                return {"ok": True, "cart": self._cart_payload()}

            if name == "get_combo_options":
                product_id = str(args.get("productId", "")).strip()
                if not product_id:
                    return {"ok": False, "code": "invalid_combo"}
                combo = await self.fudia.get_combo(self._token(), product_id)
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

            if name == "add_combo_item":
                product_id = str(args.get("productId", "")).strip()
                quantity = float(args.get("quantity", 0))
                note = str(args.get("note", "")).strip()
                raw_selections = args.get("selections", [])
                if not product_id or quantity <= 0 or not isinstance(raw_selections, list):
                    return {"ok": False, "code": "invalid_combo"}

                combo = await self.fudia.get_combo(self._token(), product_id)
                groups_by_id = {group.id: group for group in combo.groups}
                selected_by_group: dict[str, list[str]] = {}
                seen: set[tuple[str, str]] = set()

                for raw in raw_selections:
                    if not isinstance(raw, dict):
                        return {"ok": False, "code": "invalid_combo_selection"}
                    group_id = str(raw.get("groupId", "")).strip()
                    option_id = str(raw.get("productId", "")).strip()
                    key = (group_id, option_id)
                    if not group_id or not option_id or key in seen:
                        return {"ok": False, "code": "invalid_combo_selection"}
                    if group_id not in groups_by_id:
                        return {"ok": False, "code": "invalid_combo_selection"}
                    seen.add(key)
                    selected_by_group.setdefault(group_id, []).append(option_id)

                selections: list[CartSelection] = []
                surcharge_total = Decimal("0")
                for group in combo.groups:
                    selected = selected_by_group.get(group.id, [])
                    minimum = max(group.minSelections, 1 if group.required else 0)
                    if len(selected) < minimum or len(selected) > group.maxSelections:
                        return {
                            "ok": False,
                            "code": "combo_group_incomplete",
                            "group": group.name,
                            "minSelections": minimum,
                            "maxSelections": group.maxSelections,
                        }
                    options = {option.productId: option for option in group.options}
                    for option_id in selected:
                        option = options.get(option_id)
                        if option is None:
                            return {
                                "ok": False,
                                "code": "invalid_combo_selection",
                                "group": group.name,
                            }
                        if not option.available:
                            return {
                                "ok": False,
                                "code": "combo_option_unavailable",
                                "name": option.name,
                            }
                        selections.append(
                            CartSelection(
                                group_id=group.id,
                                group_name=group.name,
                                product_id=option.productId,
                                name=option.name,
                                surcharge=option.surcharge,
                            )
                        )
                        surcharge_total += option.surcharge

                self.session.cart.append(
                    CartLine(
                        product_id=combo.id,
                        name=combo.name,
                        quantity=quantity,
                        unit_price=combo.price + surcharge_total,
                        note=note,
                        item_type="combo",
                        selections=selections,
                    )
                )
                self.session.awaiting_confirmation = False
                return {"ok": True, "cart": self._cart_payload()}

            if name == "remove_item":
                product_id = str(args.get("productId", "")).strip()
                before = len(self.session.cart)
                self.session.cart = [
                    line for line in self.session.cart if line.product_id != product_id
                ]
                self.session.awaiting_confirmation = False
                return {
                    "ok": len(self.session.cart) != before,
                    "cart": self._cart_payload(),
                }

            if name == "view_cart":
                return {"ok": True, "cart": self._cart_payload()}

            if name == "prepare_confirmation":
                if not self.session.cart:
                    return {"ok": False, "code": "empty_cart"}
                self.session.awaiting_confirmation = True
                return {
                    "ok": True,
                    "requiresExplicitConfirmation": True,
                    "cart": self._cart_payload(),
                }

            if name == "confirm_order":
                if not self.session.cart:
                    return {"ok": False, "code": "empty_cart"}
                if not self.session.awaiting_confirmation or not explicit_confirmation(
                    self.user_message
                ):
                    return {
                        "ok": False,
                        "code": "confirmation_required",
                        "requiresExplicitConfirmation": True,
                    }

                result = await self.fudia.create_order(
                    self._token(),
                    self.session.phone,
                    [
                        {
                            "productId": line.product_id,
                            "qty": line.quantity,
                            "note": line.note,
                            "selections": [
                                {
                                    "groupId": selection.group_id,
                                    "productId": selection.product_id,
                                }
                                for selection in line.selections
                            ],
                        }
                        for line in self.session.cart
                    ],
                    conversation_id=self.session.conversation_id,
                )
                self.session.cart = []
                self.session.awaiting_confirmation = False
                self.session.last_order_id = result.id
                return {
                    "ok": True,
                    "order": {
                        "id": result.id,
                        "code": result.code,
                        "status": result.status,
                        "total": str(result.total),
                    },
                }

            return {"ok": False, "code": "unknown_tool"}
        except FudiaError as exc:
            return {
                "ok": False,
                "code": exc.code,
                "message": exc.message,
                "status": exc.status_code,
            }
