"""Delivery y recojo cuando la atención en mesa terminó."""

from __future__ import annotations

from decimal import Decimal
from typing import Any

import pytest

from src.api.webhook import LOCATION_TEXT, _incoming_locations
from src.domain.models import (
    ConversationSession,
    DeliveryOptions,
    DeliveryQuote,
    DeliveryTier,
    EffectRecord,
    GeoPoint,
    InboundMessage,
    OrderResult,
    OrderStatus,
    ProcessingRecord,
    VenueContext,
)
from src.infrastructure.state_store import MemoryConversationStore
from src.services.concierge_service import ConciergeService, agent_message
from src.services.ports import FudiaError
from src.services.processing import recovery_reply
from src.services.tools import ConciergeTools
from tests.test_concierge_service import FakeFudia

PHONE = "51999999999"
QR = "FUDIA:" + "a" * 32
PIN = GeoPoint(latitude=-12.12, longitude=-77.03)


class DeliveryFudia(FakeFudia):
    def __init__(
        self,
        *,
        active: bool = True,
        delivery_available: bool = True,
        covered: bool = True,
    ) -> None:
        super().__init__()
        self.active = active
        self.delivery_available = delivery_available
        self.covered = covered
        self.fee = Decimal("8.00")
        self.fee_changed = False
        self.delivery_orders: list[dict[str, Any]] = []
        self.quote_calls: list[tuple[float, float]] = []

    async def get_order_status(
        self, token: str, order_id: str, conversation_id: str
    ) -> OrderStatus:
        return OrderStatus(
            orderId=order_id,
            code="PED-001",
            channel="whatsapp",
            status="preparando" if self.active else "entregado",
            total=Decimal("32.50"),
            active=self.active,
        )

    async def get_delivery_options(self, token: str) -> DeliveryOptions:
        return DeliveryOptions(
            locationName="Local principal",
            locationAddress="Av. Principal 100",
            locationPhone="014445555",
            currencySymbol="S/",
            deliveryAvailable=self.delivery_available,
            maxDistanceKm=Decimal("6") if self.delivery_available else Decimal("0"),
            tiers=(
                [
                    DeliveryTier(maxDistanceKm=Decimal("3"), fee=Decimal("5")),
                    DeliveryTier(maxDistanceKm=Decimal("6"), fee=Decimal("8")),
                ]
                if self.delivery_available
                else []
            ),
        )

    async def quote_delivery(
        self, token: str, latitude: float, longitude: float
    ) -> DeliveryQuote:
        self.quote_calls.append((latitude, longitude))
        if not self.delivery_available:
            raise FudiaError(409, "delivery_unavailable", "Sin delivery.")
        distance = Decimal("4.20") if self.covered else Decimal("9.80")
        return DeliveryQuote(
            covered=self.covered,
            distanceKm=distance,
            fee=self.fee if self.covered else Decimal("0"),
            maxDistanceKm=Decimal("6.00"),
            currencySymbol="S/",
        )

    async def create_delivery_order(
        self, token: str, order: dict[str, Any]
    ) -> OrderResult:
        if self.fee_changed:
            self.fee = Decimal("10.00")
            raise FudiaError(409, "delivery_fee_changed", "La tarifa cambió.")
        self.delivery_orders.append(order)
        return OrderResult(
            id="d1", code="PED-002", status="confirmado", total=Decimal("40.50")
        )


class RecordingEngine:
    def __init__(self) -> None:
        self.messages: list[str] = []

    async def reply(self, session, user_message, tool_handler) -> str:
        self.messages.append(user_message)
        return "¿Qué deseas pedir?"


class Clock:
    def __init__(self) -> None:
        self.now = 1_000_000.0

    def __call__(self) -> float:
        return self.now


def delivery_session(**updates: Any) -> ConversationSession:
    session = ConversationSession(
        phone=PHONE,
        qr_token="a" * 32,
        mode="delivery",
        venue=VenueContext(
            organizationName="Restaurante Demo", locationName="Local principal"
        ),
    )
    for key, value in updates.items():
        setattr(session, key, value)
    return session


def test_webhook_accepts_location_pins_and_ignores_invalid_ones() -> None:
    def payload(location: Any) -> dict[str, Any]:
        return {
            "entry": [
                {
                    "changes": [
                        {
                            "value": {
                                "metadata": {"phone_number_id": "pid"},
                                "messages": [
                                    {
                                        "id": "wamid.1",
                                        "from": PHONE,
                                        "type": "location",
                                        "location": location,
                                    }
                                ],
                            }
                        }
                    ]
                }
            ]
        }

    found = _incoming_locations(
        payload({"latitude": -12.12, "longitude": -77.03, "name": "Casa"})
    )
    assert found == [
        ("wamid.1", PHONE, GeoPoint(latitude=-12.12, longitude=-77.03, name="Casa"), "pid")
    ]
    assert _incoming_locations(payload({"latitude": 120, "longitude": 0})) == []
    assert _incoming_locations(payload({"latitude": "x"})) == []
    assert _incoming_locations(payload("bad")) == []
    assert LOCATION_TEXT


async def test_finished_table_account_switches_to_delivery_mode() -> None:
    store = MemoryConversationStore()
    fudia = DeliveryFudia()
    engine = RecordingEngine()
    service = ConciergeService(store, fudia, engine)

    await service.handle_message(PHONE, QR)
    session = await store.get(PHONE)
    assert session is not None
    session.last_order_id = "o1"
    session.cart = []
    await store.save(session)
    table_conversation = session.conversation_id

    # Account still open: the conversation keeps serving the table.
    await service.handle_message(PHONE, "Quiero otra gaseosa")
    session = await store.get(PHONE)
    assert session is not None and session.mode == "table"
    assert "Modo: mesa abierta, Mesa M1" in engine.messages[-1]
    assert engine.messages[-1].endswith("Mensaje del cliente:\nQuiero otra gaseosa")

    fudia.active = False
    reply = await service.handle_message(PHONE, "Quiero pedir para llevar")

    session = await store.get(PHONE)
    assert session is not None
    assert session.mode == "delivery"
    assert session.table is None
    assert session.last_order_id is None
    assert session.conversation_id != table_conversation
    assert reply.startswith("Tu atención en mesa ya terminó")
    assert "recojo en el local o delivery" in reply
    assert "Modo: pedido sin mesa" in engine.messages[-1]
    assert engine.messages[-1].endswith("Quiero pedir para llevar")


async def test_idle_table_without_orders_ends_after_the_idle_window() -> None:
    store = MemoryConversationStore()
    clock = Clock()
    service = ConciergeService(
        store, DeliveryFudia(), RecordingEngine(), clock=clock,
        table_idle_seconds=3_600,
    )
    await service.handle_message(PHONE, QR)

    clock.now += 1_800
    await service.handle_message(PHONE, "hola")
    session = await store.get(PHONE)
    assert session is not None and session.mode == "table"

    clock.now += 3_601
    reply = await service.handle_message(PHONE, "hola")
    session = await store.get(PHONE)
    assert session is not None and session.mode == "delivery"
    assert "atención en mesa ya terminó" in reply


async def test_remembered_venue_takes_orders_after_the_session_expired() -> None:
    store = MemoryConversationStore()
    service = ConciergeService(store, DeliveryFudia(), RecordingEngine())
    await service.handle_message(PHONE, QR)
    memory = await store.get_memory(PHONE)
    assert memory is not None and memory.qr_token == "a" * 32

    store.items.clear()  # session TTL expired, memory survives
    reply = await service.handle_message(PHONE, "Hola, quiero pedir")

    assert reply.startswith("Hola de nuevo")
    session = await store.get(PHONE)
    assert session is not None
    assert session.mode == "delivery" and session.qr_token == "a" * 32


async def test_without_qr_or_memory_the_diner_must_scan() -> None:
    service = ConciergeService(
        MemoryConversationStore(), DeliveryFudia(), RecordingEngine()
    )
    reply = await service.handle_message(PHONE, "Quiero delivery")
    assert "vuelve a escanear el QR" in reply


async def test_location_pin_is_quoted_by_backend_without_the_model() -> None:
    store = MemoryConversationStore()
    fudia = DeliveryFudia()
    engine = RecordingEngine()
    service = ConciergeService(store, fudia, engine)
    await service.handle_message(PHONE, QR)
    store.items.clear()

    record = await service.process_inbound(
        InboundMessage(
            message_id="pin-1", phone=PHONE, text=LOCATION_TEXT, location=PIN
        )
    )

    assert fudia.quote_calls == [(-12.12, -77.03)]
    assert engine.messages == []
    assert "4.20 km" in record.reply and "S/ 8.00" in record.reply
    assert "dirección exacta" in record.reply
    session = await store.get(PHONE)
    assert session is not None
    assert session.delivery.fulfillment == "delivery"
    assert session.delivery.fee == Decimal("8.00")


async def test_location_out_of_range_offers_pickup() -> None:
    store = MemoryConversationStore()
    service = ConciergeService(store, DeliveryFudia(covered=False), RecordingEngine())
    await service.handle_message(PHONE, QR)
    store.items.clear()
    await service.handle_message(PHONE, "hola")

    record = await service.process_inbound(
        InboundMessage(message_id="pin-2", phone=PHONE, text=LOCATION_TEXT, location=PIN)
    )
    assert "9.80 km" in record.reply and "6.00 km" in record.reply
    assert "recoger" in record.reply
    session = await store.get(PHONE)
    assert session is not None and session.delivery.fee is None


async def test_location_at_an_open_table_is_not_quoted() -> None:
    store = MemoryConversationStore()
    fudia = DeliveryFudia()
    service = ConciergeService(store, fudia, RecordingEngine())
    await service.handle_message(PHONE, QR)

    record = await service.process_inbound(
        InboundMessage(message_id="pin-3", phone=PHONE, text=LOCATION_TEXT, location=PIN)
    )
    assert fudia.quote_calls == []
    assert "en la mesa" in record.reply


async def test_delivery_tools_are_unavailable_at_the_table() -> None:
    session = ConversationSession(phone=PHONE, qr_token="a" * 32)
    tools = ConciergeTools(DeliveryFudia(), session, "delivery")
    result = await tools.execute("set_fulfillment", {"type": "delivery"})
    assert result["code"] == "not_available_at_table"


async def test_delivery_order_requires_details_and_sends_quoted_fee() -> None:
    fudia = DeliveryFudia()
    session = delivery_session()
    tools = ConciergeTools(fudia, session, "quiero uno")
    await tools.execute("add_item", {"productId": "p1", "quantity": 1, "note": ""})

    missing = await tools.execute("prepare_confirmation", {})
    assert missing["code"] == "delivery_details_required"
    assert missing["missing"] == ["fulfillment"]

    assert (await tools.execute("set_fulfillment", {"type": "delivery"}))["ok"]
    missing = await tools.execute("prepare_confirmation", {})
    assert set(missing["missing"]) == {"customerName", "location", "address"}

    quote = await fudia.quote_delivery("a" * 32, PIN.latitude, PIN.longitude)
    from src.services.delivery_tools import apply_quote

    apply_quote(session, PIN, quote)
    await tools.execute(
        "set_delivery_details",
        {"customerName": "Ana", "address": "Av. Larco 123", "reference": "Puerta azul"},
    )
    prepared = await tools.execute("prepare_confirmation", {})
    assert prepared["ok"] is True
    assert prepared["delivery"]["deliveryFee"] == "8.00"
    assert prepared["delivery"]["total"] == "40.50"

    confirmed = await ConciergeTools(fudia, session, "Sí").execute("confirm_order", {})
    assert confirmed["ok"] is True
    assert fudia.order_calls == []  # never the table endpoint
    order = fudia.delivery_orders[0]
    assert order["fulfillment"] == "delivery"
    assert order["quotedFee"] == "8.00"
    assert (order["latitude"], order["longitude"]) == (-12.12, -77.03)
    assert order["address"] == "Av. Larco 123"
    assert order["requestId"] and order["conversationId"] == session.conversation_id
    assert session.cart == [] and session.last_order_id == "d1"


async def test_pickup_order_needs_only_a_name() -> None:
    fudia = DeliveryFudia(delivery_available=False)
    session = delivery_session()
    tools = ConciergeTools(fudia, session, "para recoger")
    await tools.execute("add_item", {"productId": "p1", "quantity": 1, "note": ""})

    unavailable = await tools.execute("set_fulfillment", {"type": "delivery"})
    assert unavailable["code"] == "delivery_unavailable"
    await tools.execute("set_fulfillment", {"type": "pickup"})
    await tools.execute(
        "set_delivery_details", {"customerName": "Luis", "address": "", "reference": ""}
    )
    prepared = await tools.execute("prepare_confirmation", {})
    assert prepared["ok"] is True and prepared["delivery"]["deliveryFee"] == "0.00"

    await ConciergeTools(fudia, session, "confirmo").execute("confirm_order", {})
    order = fudia.delivery_orders[0]
    assert order["fulfillment"] == "pickup"
    assert "quotedFee" not in order and "address" not in order


async def test_changed_fee_is_requoted_and_needs_a_new_confirmation() -> None:
    fudia = DeliveryFudia()
    session = delivery_session()
    tools = ConciergeTools(fudia, session, "quiero uno")
    await tools.execute("add_item", {"productId": "p1", "quantity": 1, "note": ""})
    await tools.execute("set_fulfillment", {"type": "delivery"})
    from src.services.delivery_tools import apply_quote

    apply_quote(session, PIN, await fudia.quote_delivery("t", 0, 0))
    await tools.execute(
        "set_delivery_details", {"customerName": "Ana", "address": "Calle 1", "reference": ""}
    )
    await tools.execute("prepare_confirmation", {})
    fudia.fee_changed = True

    result = await ConciergeTools(fudia, session, "Sí").execute("confirm_order", {})

    assert result["code"] == "delivery_fee_changed"
    assert session.delivery.fee == Decimal("10.00")
    assert session.awaiting_confirmation is False
    assert session.pending_order_request_id is None
    assert len(session.cart) == 1


async def test_bill_and_staff_are_not_available_without_a_table() -> None:
    session = delivery_session()
    tools = ConciergeTools(DeliveryFudia(), session, "la cuenta")
    bill = await tools.execute("request_bill", {})
    assert bill["code"] == "bill_not_available_for_delivery"
    human = await tools.execute("request_human", {"reason": "ayuda"})
    assert human["code"] == "handoff_not_available_for_delivery"
    assert human["locationPhone"] == "014445555"


async def test_order_status_reads_backend() -> None:
    session = delivery_session(last_order_id="d1")
    result = await ConciergeTools(DeliveryFudia(), session, "¿y mi pedido?").execute(
        "check_order_status", {}
    )
    assert result["ok"] is True
    assert result["order"]["statusLabel"] == "en preparación"


def test_agent_message_carries_delivery_state_only_without_table() -> None:
    table = ConversationSession(phone=PHONE, qr_token="a" * 32)
    assert agent_message(table, "hola") == "hola"
    note = agent_message(delivery_session(), "hola")
    assert "Tipo de entrega: pendiente de elegir" in note
    assert "Falta: fulfillment" in note
    assert note.endswith("hola")


def test_recovery_reply_for_delivery_order() -> None:
    record = ProcessingRecord(
        message_id="m",
        effects=[
            EffectRecord(
                kind="create_order",
                payload={"mode": "delivery", "order": {"fulfillment": "pickup"}},
                result={"order": {"code": "PED-9", "total": "30"}},
            )
        ],
    )
    reply = recovery_reply(record, delivery_session())
    assert "PED-9" in reply and "recogerlo" in reply


@pytest.mark.parametrize("change", ["fulfillment", "details"])
async def test_delivery_changes_invalidate_a_prepared_confirmation(change: str) -> None:
    fudia = DeliveryFudia()
    session = delivery_session()
    tools = ConciergeTools(fudia, session, "x")
    await tools.execute("add_item", {"productId": "p1", "quantity": 1, "note": ""})
    await tools.execute("set_fulfillment", {"type": "pickup"})
    await tools.execute(
        "set_delivery_details", {"customerName": "Ana", "address": "", "reference": ""}
    )
    await tools.execute("prepare_confirmation", {})
    assert session.awaiting_confirmation

    if change == "fulfillment":
        await tools.execute("set_fulfillment", {"type": "delivery"})
    else:
        await tools.execute(
            "set_delivery_details", {"customerName": "Otra", "address": "", "reference": ""}
        )
    assert session.awaiting_confirmation is False


async def test_crash_after_delivery_order_replays_the_same_request_id() -> None:
    from tests.test_recovery import (
        ADD_P1,
        Backend,
        ScriptedAgent,
        WorkerDied,
        message,
        open_table,
        service,
        session_of,
    )

    class DeliveryBackend(Backend):
        def __init__(self) -> None:
            super().__init__()
            self.delivery_calls: list[str] = []
            self.die_after_delivery = False

        async def create_delivery_order(
            self, token: str, order: dict[str, Any]
        ) -> OrderResult:
            request_id = str(order["requestId"])
            self.delivery_calls.append(request_id)
            if request_id not in self.orders:
                self.orders[request_id] = OrderResult(
                    id="d1", code="PED-777", status="confirmado", total=Decimal("15.00")
                )
            if self.die_after_delivery:
                self.die_after_delivery = False
                raise WorkerDied("backend created the delivery, worker died")
            return self.orders[request_id]

    store, backend = MemoryConversationStore(), DeliveryBackend()
    await open_table(store, backend)
    session = await session_of(store)
    session.mode = "delivery"
    session.table = None
    session.delivery.fulfillment = "pickup"
    session.delivery.customer_name = "Ana"
    await store.save(session)
    await service(store, backend, ScriptedAgent([ADD_P1])).process_inbound(
        message("wamid.d-cart", "un p1")
    )

    confirm = [("prepare_confirmation", {}), ("confirm_order", {})]
    backend.die_after_delivery = True
    with pytest.raises(WorkerDied):
        await service(store, backend, ScriptedAgent(confirm)).process_inbound(
            message("wamid.d-yes", "sí")
        )
    restarted = ScriptedAgent(confirm)
    record = await service(store, backend, restarted).process_inbound(
        message("wamid.d-yes", "sí")
    )

    assert restarted.calls == 0
    assert len(backend.delivery_calls) == 2
    assert len(set(backend.delivery_calls)) == 1
    assert len(backend.orders) == 1
    assert "PED-777" in record.reply and "recogerlo" in record.reply
    session = await session_of(store)
    assert session.cart == [] and session.last_order_id == "d1"
