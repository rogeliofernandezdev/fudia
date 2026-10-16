from __future__ import annotations

from decimal import Decimal
from typing import Any
from uuid import uuid4

from src.domain.models import CartLine, CartModifier, CartSelection
from src.services.confirmation import explicit_confirmation
from src.services.menu_tools import MenuTools
from src.services.ports import FudiaError
from src.services.tooling import ToolContext


class OrderTools:
    def __init__(self, context: ToolContext) -> None:
        self.context = context
        self.menu = MenuTools(context)

    async def execute(
        self,
        name: str,
        args: dict[str, Any],
    ) -> dict[str, Any]:
        if name in {
            "search_menu",
            "get_modifier_options",
            "get_combo_options",
        }:
            return await self.menu.execute(name, args)

        if name == "add_item":
            product_id = str(args.get("productId", "")).strip()
            quantity = float(args.get("quantity", 0))
            note = str(args.get("note", "")).strip()
            if not product_id or quantity <= 0:
                return {"ok": False, "code": "invalid_item"}

            item = await self.context.fudia.get_product(
                self.context.token(),
                product_id,
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
            if item.hasModifiers:
                return {
                    "ok": False,
                    "code": "modifier_review_required",
                    "name": item.name,
                }

            existing = next(
                (
                    line
                    for line in self.context.session.cart
                    if line.product_id == product_id
                    and not line.modifiers
                    and not line.selections
                ),
                None,
            )
            if existing:
                existing.quantity += quantity
                if note:
                    existing.note = note
                existing.unit_price = item.price
                existing.name = item.name
            else:
                self.context.session.cart.append(
                    CartLine(
                        product_id=item.productId,
                        name=item.name,
                        quantity=quantity,
                        unit_price=item.price,
                        note=note,
                    )
                )
            return await self._cart_changed()

        if name == "add_modified_item":
            return await self._add_modified_item(args)

        if name == "add_combo_item":
            return await self._add_combo_item(args)

        if name == "remove_item":
            product_id = str(args.get("productId", "")).strip()
            before = len(self.context.session.cart)
            self.context.session.cart = [
                line
                for line in self.context.session.cart
                if line.product_id != product_id
            ]
            if len(self.context.session.cart) == before:
                return {
                    "ok": False,
                    "cart": self.context.cart_payload(),
                }
            return await self._cart_changed()

        if name == "view_cart":
            return {
                "ok": True,
                "cart": self.context.cart_payload(),
            }

        if name == "prepare_confirmation":
            if not self.context.session.cart:
                return {"ok": False, "code": "empty_cart"}
            self.context.session.awaiting_confirmation = True
            if not self.context.session.pending_order_request_id:
                self.context.session.pending_order_request_id = uuid4().hex
            cart = self.context.cart_payload()
            await self.context.journal.local_effect(
                "confirmation",
                {"cart": cart},
            )
            return {
                "ok": True,
                "requiresExplicitConfirmation": True,
                "cart": cart,
            }

        if name == "confirm_order":
            return await self._confirm_order()

        return {"ok": False, "code": "unknown_order_tool"}

    async def _add_modified_item(
        self,
        args: dict[str, Any],
    ) -> dict[str, Any]:
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

        item = await self.context.fudia.get_product(
            self.context.token(),
            product_id,
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

        config = await self.context.fudia.get_modifiers(
            self.context.token(),
            product_id,
        )
        groups = {group.id: group for group in config.groups}
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
            if (
                not group_id
                or not option_id
                or key in seen
                or group_id not in groups
            ):
                return {
                    "ok": False,
                    "code": "invalid_modifier_selection",
                }
            seen.add(key)
            selected_by_group.setdefault(group_id, []).append(option_id)

        selected_modifiers: list[CartModifier] = []
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
                option.id: option
                for option in group.options
            }
            for option_id in selected:
                option = options.get(option_id)
                if option is None:
                    return {
                        "ok": False,
                        "code": "invalid_modifier_selection",
                        "group": group.name,
                    }
                selected_modifiers.append(
                    CartModifier(
                        group_id=group.id,
                        group_name=group.name,
                        option_id=option.id,
                        name=option.name,
                        surcharge=option.surcharge,
                    )
                )
                surcharge_total += option.surcharge

        self.context.session.cart.append(
            CartLine(
                product_id=item.productId,
                name=item.name,
                quantity=quantity,
                unit_price=item.price + surcharge_total,
                note=note,
                item_type="product",
                modifiers=selected_modifiers,
            )
        )
        return await self._cart_changed()

    async def _add_combo_item(
        self,
        args: dict[str, Any],
    ) -> dict[str, Any]:
        product_id = str(args.get("productId", "")).strip()
        quantity = float(args.get("quantity", 0))
        note = str(args.get("note", "")).strip()
        raw_selections = args.get("selections", [])
        if (
            not product_id
            or quantity <= 0
            or not isinstance(raw_selections, list)
        ):
            return {"ok": False, "code": "invalid_combo"}

        combo = await self.context.fudia.get_combo(
            self.context.token(),
            product_id,
        )
        groups = {group.id: group for group in combo.groups}
        selected_by_group: dict[str, list[str]] = {}
        seen: set[tuple[str, str]] = set()

        for raw in raw_selections:
            if not isinstance(raw, dict):
                return {
                    "ok": False,
                    "code": "invalid_combo_selection",
                }
            group_id = str(raw.get("groupId", "")).strip()
            option_id = str(raw.get("productId", "")).strip()
            key = (group_id, option_id)
            if (
                not group_id
                or not option_id
                or key in seen
                or group_id not in groups
            ):
                return {
                    "ok": False,
                    "code": "invalid_combo_selection",
                }
            seen.add(key)
            selected_by_group.setdefault(group_id, []).append(option_id)

        selections: list[CartSelection] = []
        surcharge_total = Decimal("0")
        for group in combo.groups:
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
                    "code": "combo_group_incomplete",
                    "group": group.name,
                    "minSelections": minimum,
                    "maxSelections": group.maxSelections,
                }
            options = {
                option.productId: option
                for option in group.options
            }
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

        self.context.session.cart.append(
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
        return await self._cart_changed()

    async def _cart_changed(self) -> dict[str, Any]:
        self.context.invalidate_confirmation()
        cart = self.context.cart_payload()
        await self.context.journal.local_effect("cart", {"cart": cart})
        return {"ok": True, "cart": cart}

    async def _confirm_order(self) -> dict[str, Any]:
        session = self.context.session
        if not session.cart:
            return {"ok": False, "code": "empty_cart"}
        if (
            not session.awaiting_confirmation
            or not explicit_confirmation(self.context.user_message)
        ):
            return {
                "ok": False,
                "code": "confirmation_required",
                "requiresExplicitConfirmation": True,
            }

        # The idempotency key lives in the session before the backend sees
        # it: a retry after a crash, a lost lease or a network error reuses
        # it and recovers the same round instead of creating another one.
        if not session.pending_order_request_id:
            session.pending_order_request_id = uuid4().hex
        request_id = session.pending_order_request_id
        lines = order_lines(session.cart)
        await self.context.journal.before_effect(
            "create_order",
            request_id,
            {
                "token": self.context.token(),
                "phone": session.phone,
                "conversationId": session.conversation_id,
                "lines": lines,
            },
        )
        try:
            result = await self.context.fudia.create_order(
                self.context.token(),
                session.phone,
                lines,
                conversation_id=session.conversation_id,
                request_id=request_id,
            )
        except FudiaError as exc:
            definitive = 400 <= exc.status_code < 500
            await self.context.journal.effect_failed(exc.code, definitive)
            raise
        session.cart = []
        session.awaiting_confirmation = False
        session.pending_order_request_id = None
        session.last_order_id = result.id
        order = {
            "id": result.id,
            "code": result.code,
            "status": result.status,
            "total": str(result.total),
        }
        await self.context.journal.after_effect({"order": order})
        return {"ok": True, "order": order}


def order_lines(cart: list[CartLine]) -> list[dict[str, Any]]:
    return [
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
            "modifiers": [
                {
                    "groupId": modifier.group_id,
                    "optionId": modifier.option_id,
                }
                for modifier in line.modifiers
            ],
        }
        for line in cart
    ]
