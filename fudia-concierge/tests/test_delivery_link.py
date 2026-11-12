"""Enlace/QR de delivery del local: abre el pedido de delivery o recojo sin mesa."""

from __future__ import annotations

import pytest

from src.domain.models import TableContext
from src.infrastructure.state_store import MemoryConversationStore
from src.services.concierge_service import ConciergeService
from src.services.start_codes import looks_like_start_message, start_message
from src.services.welcome import welcome_message
from tests.test_concierge_service import FakeFudia

PHONE = "51999999999"
LINK = "FUDIA:" + "a" * 32


class DeliveryLinkFudia(FakeFudia):
    """foods-backend con el enlace de delivery del local (serviceType=delivery)."""

    async def resolve_table(self, token: str) -> TableContext:
        table = await super().resolve_table(token)
        return table.model_copy(update={"name": "Delivery", "serviceType": "delivery"})


class QuietAgent:
    async def reply(self, session, user_message, tool_handler) -> str:  # type: ignore[no-untyped-def]
        return "¿Algo más?"


def test_delivery_start_message_is_recognized() -> None:
    message = start_message("Delivery", "K7P2", "delivery")
    assert message == "¡Hola! 👋 Quiero pedir delivery.\nDelivery · K7P2"
    assert "Mesa" not in message
    assert looks_like_start_message("Quiero pedir delivery")


def test_delivery_welcome_names_the_location_without_a_table() -> None:
    table = TableContext(
        name="Delivery",
        seats=0,
        organizationName="La Rinconada",
        locationName="Miraflores",
        serviceType="delivery",
    )
    reply = welcome_message(table, None, customer_name="Ana")
    assert reply == (
        "🍽️ Bienvenido a La Rinconada\n🛵 Miraflores\n\n"
        "Hola, Ana. Pide aquí para delivery o para recoger en el local.\n\n"
        "¿Qué deseas pedir?"
    )
    assert "Mesa" not in reply and "📍" not in reply


@pytest.mark.asyncio
async def test_delivery_link_opens_delivery_mode_bound_to_the_location() -> None:
    fudia = DeliveryLinkFudia()
    store = MemoryConversationStore()
    service = ConciergeService(store, fudia, QuietAgent())  # type: ignore[arg-type]

    reply = await service.handle_message(PHONE, LINK)

    assert "Pide aquí para delivery o para recoger en el local." in reply
    assert fudia.opened[-1]["kind"] == "delivery"
    session = await store.get(PHONE)
    assert session is not None
    assert session.mode == "delivery" and session.table is None
    assert session.qr_token == "a" * 32
    assert session.venue is not None and session.venue.organizationName == "Restaurante Demo"
    assert session.delivery.customer_name == "Ana"
    # La memoria del último local queda en el enlace: un chat posterior sigue en delivery.
    memory = session.memory()
    assert memory is not None and memory.qr_token == "a" * 32
