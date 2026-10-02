from __future__ import annotations

from decimal import Decimal
from typing import Any

from src.services.tooling import ToolContext


class ServiceTools:
    def __init__(self, context: ToolContext) -> None:
        self.context = context

    async def execute(
        self,
        name: str,
        args: dict[str, Any],
    ) -> dict[str, Any]:
        if name == "request_human":
            reason = str(args.get("reason", "")).strip()
            if len(reason) > 240:
                return {
                    "ok": False,
                    "code": "handoff_reason_too_long",
                }
            if not reason:
                reason = "El comensal solicitó atención del personal."
            handoff = await self.context.fudia.request_handoff(
                self.context.token(),
                self.context.session.phone,
                self.context.session.conversation_id,
                reason,
            )
            self.context.session.handoff_pending = True
            return {
                "ok": True,
                "status": str(handoff.get("status", "pending")),
                "tableName": str(handoff.get("tableName", "")),
            }

        if name == "request_bill":
            if self.context.session.cart:
                return {
                    "ok": False,
                    "code": "unconfirmed_cart",
                    "message": (
                        "Hay productos en el carrito que todavía no fueron "
                        "confirmados."
                    ),
                    "cart": self.context.cart_payload(),
                }
            bill = await self.context.fudia.request_bill(
                self.context.token(),
                self.context.session.phone,
                self.context.session.conversation_id,
            )
            return {
                "ok": True,
                "bill": {
                    "orderId": bill.orderId,
                    "code": bill.code,
                    "tableName": bill.tableName,
                    "status": bill.status,
                    "currencySymbol": bill.currencySymbol,
                    "items": [
                        {
                            "name": item.name,
                            "quantity": str(item.qty),
                            "unitPrice": str(item.unitPrice),
                            "lineTotal": str(
                                (
                                    item.qty * item.unitPrice
                                ).quantize(Decimal("0.01"))
                            ),
                            "note": item.note,
                        }
                        for item in bill.items
                    ],
                    "total": str(bill.total),
                    "paidAmount": str(bill.paidAmount),
                    "remainingAmount": str(bill.remainingAmount),
                    "paymentStatus": bill.paymentStatus,
                },
            }

        return {"ok": False, "code": "unknown_service_tool"}
