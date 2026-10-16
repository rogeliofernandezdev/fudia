"""Delivery y recojo cuando la atención en mesa terminó.

foods-backend define si hay delivery, la distancia y la tarifa; el bot solo
guarda lo que el cliente eligió y lo que el backend cotizó para su pin.
"""

from __future__ import annotations

from decimal import Decimal
from typing import Any

from src.domain.models import (
    ConversationSession,
    DeliveryQuote,
    GeoPoint,
)
from src.services.tooling import ToolContext, cart_payload

NOT_IN_DELIVERY_MODE = {
    "ok": False,
    "code": "not_available_at_table",
    "message": (
        "Mientras la atención en mesa está abierta los pedidos van a la "
        "mesa; delivery y recojo están disponibles al terminar la atención."
    ),
}


def _money(value: Decimal | None) -> str:
    return str((value or Decimal("0")).quantize(Decimal("0.01")))


def apply_quote(
    session: ConversationSession,
    point: GeoPoint,
    quote: DeliveryQuote,
) -> None:
    """A shared pin means delivery: store it with the backend's quote."""
    delivery = session.delivery
    delivery.fulfillment = "delivery"
    delivery.location = point
    delivery.covered = quote.covered
    delivery.distance_km = quote.distanceKm
    delivery.fee = quote.fee if quote.covered else None
    delivery.max_distance_km = quote.maxDistanceKm
    delivery.currency_symbol = quote.currencySymbol


def missing_details(session: ConversationSession) -> list[str]:
    delivery = session.delivery
    missing: list[str] = []
    if delivery.fulfillment is None:
        return ["fulfillment"]
    if not delivery.customer_name:
        missing.append("customerName")
    if delivery.fulfillment == "delivery":
        if delivery.location is None:
            missing.append("location")
        elif not delivery.covered:
            missing.append("locationInRange")
        if not delivery.address:
            missing.append("address")
    return missing


def delivery_summary(session: ConversationSession) -> dict[str, Any]:
    delivery = session.delivery
    cart = cart_payload(session)
    subtotal = Decimal(cart["total"])
    fee = (
        delivery.fee or Decimal("0")
        if delivery.fulfillment == "delivery"
        else Decimal("0")
    )
    summary: dict[str, Any] = {
        "fulfillment": delivery.fulfillment,
        "customerName": delivery.customer_name,
        "subtotal": _money(subtotal),
        "deliveryFee": _money(fee),
        "total": _money(subtotal + fee),
        "currencySymbol": delivery.currency_symbol,
        "payment": "Se paga al recibir o al recoger el pedido.",
        "missing": missing_details(session),
    }
    if delivery.fulfillment == "delivery":
        summary.update(
            {
                "address": delivery.address,
                "reference": delivery.reference,
                "locationShared": delivery.location is not None,
                "covered": delivery.covered,
                "distanceKm": (
                    _money(delivery.distance_km)
                    if delivery.distance_km is not None
                    else None
                ),
                "maxDistanceKm": (
                    _money(delivery.max_distance_km)
                    if delivery.max_distance_km is not None
                    else None
                ),
            }
        )
    return summary


def delivery_order(
    session: ConversationSession,
    request_id: str,
    lines: list[dict[str, Any]],
) -> dict[str, Any]:
    """Body of POST delivery-orders; persisted as the replay payload."""
    delivery = session.delivery
    order: dict[str, Any] = {
        "fulfillment": delivery.fulfillment,
        "customerName": delivery.customer_name,
        "customerPhone": session.phone,
        "conversationId": session.conversation_id,
        "requestId": request_id,
        "items": lines,
    }
    if delivery.fulfillment == "delivery" and delivery.location is not None:
        order.update(
            {
                "address": delivery.address,
                "reference": delivery.reference,
                "latitude": delivery.location.latitude,
                "longitude": delivery.location.longitude,
                "quotedFee": _money(delivery.fee),
            }
        )
    return order


class DeliveryTools:
    def __init__(self, context: ToolContext) -> None:
        self.context = context

    async def execute(
        self,
        name: str,
        args: dict[str, Any],
    ) -> dict[str, Any]:
        if name == "get_delivery_options":
            return await self._options()
        if self.context.session.mode != "delivery":
            return dict(NOT_IN_DELIVERY_MODE)
        if name == "set_fulfillment":
            return await self._set_fulfillment(args)
        if name == "set_delivery_details":
            return await self._set_details(args)
        return {"ok": False, "code": "unknown_delivery_tool"}

    async def _options(self) -> dict[str, Any]:
        options = await self.context.fudia.get_delivery_options(
            self.context.token()
        )
        return {
            "ok": True,
            "atTable": self.context.session.mode == "table",
            "locationName": options.locationName,
            "locationAddress": options.locationAddress,
            "currencySymbol": options.currencySymbol,
            "pickupAvailable": options.pickupAvailable,
            "deliveryAvailable": options.deliveryAvailable,
            "maxDistanceKm": _money(options.maxDistanceKm),
            "tiers": [
                {
                    "upToKm": _money(tier.maxDistanceKm),
                    "fee": _money(tier.fee),
                }
                for tier in options.tiers
            ],
            "distanceRule": "Distancia en línea recta desde el local.",
            "payment": "Se paga al recibir o al recoger el pedido.",
        }

    async def _set_fulfillment(self, args: dict[str, Any]) -> dict[str, Any]:
        kind = str(args.get("type", "")).strip()
        if kind not in {"pickup", "delivery"}:
            return {"ok": False, "code": "invalid_fulfillment"}
        if kind == "delivery":
            options = await self.context.fudia.get_delivery_options(
                self.context.token()
            )
            if not options.deliveryAvailable:
                return {
                    "ok": False,
                    "code": "delivery_unavailable",
                    "pickupAvailable": options.pickupAvailable,
                    "locationAddress": options.locationAddress,
                }
            self.context.session.delivery.currency_symbol = (
                options.currencySymbol
            )
        self.context.session.delivery.fulfillment = kind  # type: ignore[assignment]
        return await self._changed()

    async def _set_details(self, args: dict[str, Any]) -> dict[str, Any]:
        delivery = self.context.session.delivery
        name = str(args.get("customerName", "")).strip()
        address = str(args.get("address", "")).strip()
        reference = str(args.get("reference", "")).strip()
        if len(name) > 160 or len(address) > 240 or len(reference) > 240:
            return {"ok": False, "code": "delivery_details_too_long"}
        if name:
            delivery.customer_name = name
        if address:
            delivery.address = address
        if reference:
            delivery.reference = reference
        return await self._changed()

    async def _changed(self) -> dict[str, Any]:
        # Fulfillment, address or fee changes need a new confirmation.
        self.context.invalidate_confirmation()
        summary = delivery_summary(self.context.session)
        await self.context.journal.local_effect("delivery", summary)
        return {"ok": True, "delivery": summary}
