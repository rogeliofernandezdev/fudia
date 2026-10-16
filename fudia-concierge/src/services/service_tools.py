from __future__ import annotations

from collections.abc import Awaitable
from decimal import Decimal
from typing import Any, TypeVar
from uuid import uuid4

from src.domain.models import BillSummary
from src.services.ports import FudiaError
from src.services.tooling import ToolContext

T = TypeVar("T")

ORDER_STATUS_LABELS = {
    "nuevo": "recibido",
    "confirmado": "confirmado",
    "preparando": "en preparación",
    "listo": "listo",
    "en_camino": "en camino",
    "entregado": "entregado",
    "cancelado": "cancelado",
}


class ServiceTools:
    def __init__(self, context: ToolContext) -> None:
        self.context = context

    async def execute(
        self,
        name: str,
        args: dict[str, Any],
    ) -> dict[str, Any]:
        if name == "check_order_status":
            return await self._order_status()

        if self.context.session.mode == "delivery" and name in {
            "request_bill",
            "request_human",
        }:
            return await self._not_at_table(name)

        if name == "request_human":
            reason = str(args.get("reason", "")).strip()
            if len(reason) > 240:
                return {
                    "ok": False,
                    "code": "handoff_reason_too_long",
                }
            if not reason:
                reason = "El comensal solicitó atención del personal."
            session = self.context.session
            # foods-backend keeps one pending handoff per conversation, so a
            # replay after a failure reuses it instead of paging staff twice.
            await self.context.journal.before_effect(
                "request_handoff",
                session.conversation_id,
                {
                    "token": self.context.token(),
                    "phone": session.phone,
                    "conversationId": session.conversation_id,
                    "reason": reason,
                },
            )
            handoff = await self._effect(
                self.context.fudia.request_handoff(
                    self.context.token(),
                    session.phone,
                    session.conversation_id,
                    reason,
                )
            )
            session.handoff_pending = True
            result = {
                "ok": True,
                "status": str(handoff.get("status", "pending")),
                "tableName": str(handoff.get("tableName", "")),
            }
            await self.context.journal.after_effect(
                {"status": result["status"]}
            )
            return result

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
            session = self.context.session
            # Stable key: persisted before the call and reused by recovery,
            # so foods-backend records this request for the bill only once.
            request_id = uuid4().hex
            await self.context.journal.before_effect(
                "request_bill",
                request_id,
                {
                    "token": self.context.token(),
                    "phone": session.phone,
                    "conversationId": session.conversation_id,
                },
            )
            bill = await self._effect(
                self.context.fudia.request_bill(
                    self.context.token(),
                    session.phone,
                    session.conversation_id,
                    request_id,
                )
            )
            await self.context.journal.after_effect(bill_summary(bill))
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

    async def _order_status(self) -> dict[str, Any]:
        session = self.context.session
        if not session.last_order_id:
            return {"ok": False, "code": "no_confirmed_order"}
        status = await self.context.fudia.get_order_status(
            self.context.token(),
            session.last_order_id,
            session.conversation_id,
        )
        return {
            "ok": True,
            "order": {
                "code": status.code,
                "channel": status.channel,
                "status": status.status,
                "statusLabel": ORDER_STATUS_LABELS.get(
                    status.status, status.status
                ),
                "total": str(status.total),
                "deliveryFee": str(status.deliveryFee),
                "paymentStatus": status.paymentStatus,
                "active": status.active,
            },
        }

    async def _not_at_table(self, name: str) -> dict[str, Any]:
        # Without a table there is no bill to request nor staff to send.
        options = await self.context.fudia.get_delivery_options(
            self.context.token()
        )
        return {
            "ok": False,
            "code": (
                "bill_not_available_for_delivery"
                if name == "request_bill"
                else "handoff_not_available_for_delivery"
            ),
            "message": (
                "El pago de delivery o recojo se hace al recibir o recoger "
                "el pedido."
                if name == "request_bill"
                else "Para ayuda con tu pedido comunícate con el local."
            ),
            "locationName": options.locationName,
            "locationPhone": options.locationPhone,
        }

    async def _effect(self, operation: Awaitable[T]) -> T:
        try:
            return await operation
        except FudiaError as exc:
            await self.context.journal.effect_failed(
                exc.code,
                400 <= exc.status_code < 500,
            )
            raise


def bill_summary(bill: BillSummary) -> dict[str, Any]:
    """Fields a recovery reply needs; the full bill stays in the backend."""
    return {
        "code": bill.code,
        "currencySymbol": bill.currencySymbol,
        "total": str(bill.total),
        "paidAmount": str(bill.paidAmount),
        "remainingAmount": str(bill.remainingAmount),
    }
