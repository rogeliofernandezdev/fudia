from __future__ import annotations

from decimal import Decimal
from typing import Any

from src.domain.models import ConversationSession
from src.infrastructure.fudia_client import FudiaClient


class ToolContext:
    def __init__(
        self,
        fudia: FudiaClient,
        session: ConversationSession,
        user_message: str,
    ) -> None:
        self.fudia = fudia
        self.session = session
        self.user_message = user_message

    def token(self) -> str:
        if not self.session.qr_token:
            raise RuntimeError("La sesión no tiene QR.")
        return self.session.qr_token

    def invalidate_confirmation(self) -> None:
        self.session.awaiting_confirmation = False
        self.session.pending_order_request_id = None

    def cart_payload(self) -> dict[str, Any]:
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
        return {
            "items": items,
            "total": str(total.quantize(Decimal("0.01"))),
        }
