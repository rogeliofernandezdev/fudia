from __future__ import annotations

import re
import unicodedata
from decimal import Decimal
from typing import Any

from src.domain.models import CartLine, ConversationSession
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
                        "code": "combo_requires_selection_support",
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
                            "selections": [],
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
