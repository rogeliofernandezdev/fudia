"""Delivery Fudia: mensajes a las centrales y sus botones Aceptar, Rechazar y Recoger."""

from __future__ import annotations

import pytest

from src.api.webhook import _incoming_partner_actions, _incoming_text_messages
from src.domain.models import (
    CustomerNotice,
    DeliveryActionResult,
    DeliveryMessageItem,
    DeliveryPartnerMessage,
    InboundMessage,
    PartnerAction,
)
from src.infrastructure.fudia_client import _endpoint_label
from src.infrastructure.state_store import MemoryConversationStore
from src.services.concierge_service import ConciergeService
from src.services.customer_notices import notice_text
from src.services.delivery_partners import (
    DeliveryTemplates,
    PartnerMessageRelay,
    action_reply,
    button_payload,
    offer_params,
    offer_text,
    parse_payload,
    ready_params,
    template_param,
)
from src.services.ports import WhatsAppDeliveryUncertain
from tests.test_concierge_service import FakeFudia

TOKEN = "3f2a9c1e-5b7d-4e8f-9a0b-1c2d3e4f5a6b"
CENTRAL = "51987654321"


def message(**changes: object) -> DeliveryPartnerMessage:
    data: dict[str, object] = {
        "id": "msg-1",
        "kind": "offer",
        "partnerName": "Central Norte",
        "partnerPhone": "+51 987 654 321",
        "actionToken": TOKEN,
        "expiresInSeconds": 180,
        "orderCode": "PED-0142",
        "organizationName": "La Rinconada",
        "locationName": "Miraflores",
        "locationAddress": "Av. Larco 123",
        "customerName": "Ana",
        "customerPhone": "999123456",
        "address": "Av. Perú 100",
        "reference": "Puerta azul",
        "latitude": -12.1219,
        "longitude": -77.0297,
        "distanceKm": 2.35,
        "notes": "Tocar\nel timbre",
        "items": [
            DeliveryMessageItem(quantity=2, name="Lomo saltado", note="Sin ají"),
            DeliveryMessageItem(quantity=1, name="Chicha morada"),
        ],
        "currencySymbol": "S/",
        "subtotal": "60.00",
        "deliveryFee": "5.00",
        "total": "65.00",
    }
    data.update(changes)
    return DeliveryPartnerMessage.model_validate(data)


def test_payload_round_trip_and_rejects_foreign_payloads() -> None:
    assert parse_payload(button_payload("accept", TOKEN)) == PartnerAction(action="accept", token=TOKEN)
    assert parse_payload(f"fdx:pickup:{TOKEN.upper()}") == PartnerAction(action="pickup", token=TOKEN)
    for raw in ("Aceptar", f"fdx:pay:{TOKEN}", "fdx:accept:not-a-token", f"x:accept:{TOKEN}"):
        assert parse_payload(raw) is None


def test_template_params_are_single_line_and_carry_what_the_central_pays_and_collects() -> None:
    params = offer_params(message())
    assert len(params) == 7
    assert all("\n" not in value and "  " not in value for value in params)
    assert params[0] == "PED-0142"
    assert params[1] == "La Rinconada · Miraflores (Av. Larco 123)"
    assert params[2].startswith("Av. Perú 100 · Ref.: Puerta azul · 2.4 km · https://maps.google.com/?q=")
    assert params[3] == "Ana · 999123456"
    assert params[4] == "2 × Lomo saltado (Sin ají), 1 × Chicha morada"
    # La central paga al local solo el pedido y cobra al cliente pedido + su delivery.
    assert params[5] == "S/ 60.00"
    assert params[6] == "S/ 65.00 (incluye tu delivery S/ 5.00)"
    assert ready_params(message(kind="ready")) == [
        "PED-0142",
        "La Rinconada · Miraflores (Av. Larco 123)",
        "S/ 60.00",
    ]
    assert template_param("") == "-"
    assert len(template_param("x" * 500)) == 300


def test_offer_text_lists_the_order_without_markdown() -> None:
    text = offer_text(message(latitude=None, longitude=None, distanceKm=None))
    assert text.splitlines() == [
        "🛵 Nuevo delivery PED-0142",
        "Recoge en: La Rinconada · Miraflores (Av. Larco 123)",
        "Entrega en: Av. Perú 100 · Ref.: Puerta azul",
        "Cliente: Ana · 999123456",
        "Pedido:",
        "- 2 × Lomo saltado (Sin ají)",
        "- 1 × Chicha morada",
        "Notas: Tocar el timbre",
        "Pagas al local: S/ 60.00",
        "Cobras al cliente: S/ 65.00 (incluye tu delivery S/ 5.00)",
        "Responde en 3 minutos.",
    ]
    assert "*" not in text


def test_action_replies_are_idempotent_for_retries() -> None:
    accepted = action_reply(
        DeliveryActionResult(outcome="accepted", orderCode="PED-1", locationName="Miraflores")
    )
    assert accepted == action_reply(
        DeliveryActionResult(outcome="already_accepted", orderCode="PED-1", locationName="Miraflores")
    )
    assert "PED-1" in accepted and "Miraflores" in accepted
    assert "Usa los botones" in action_reply(DeliveryActionResult(outcome="not_found"))
    assert "ya no está disponible" in action_reply(DeliveryActionResult(outcome="taken", orderCode="PED-1"))
    assert "Cobra al cliente" in action_reply(DeliveryActionResult(outcome="picked_up", orderCode="PED-1"))


def test_on_the_way_notice_text() -> None:
    notice = CustomerNotice(id="n1", kind="order_on_the_way", orderCode="PED-9", fulfillment="delivery")
    assert notice_text(notice) == "🛵 Tu pedido PED-9 va en camino.\nPaga al recibirlo."


def test_paid_order_does_not_instruct_the_central_to_collect_again() -> None:
    reply = action_reply(DeliveryActionResult(outcome="payment_conflict", orderCode="PED-9"))
    assert "ya registró un pago" in reply
    assert "No lo recojas ni cobres al cliente" in reply


def test_endpoint_label() -> None:
    assert _endpoint_label("/v1/integrations/concierge/delivery-messages/pending") == "delivery_partners"
    assert _endpoint_label("/v1/integrations/concierge/delivery-actions") == "delivery_partners"


class Source:
    def __init__(self, messages: list[DeliveryPartnerMessage]) -> None:
        self.messages = messages
        self.dispatched: list[str] = []

    async def pending_delivery_messages(self, limit: int = 20) -> list[DeliveryPartnerMessage]:
        return self.messages[:limit]

    async def mark_delivery_message_dispatched(self, message_id: str) -> None:
        self.dispatched.append(message_id)


class Sender:
    def __init__(self, fail: Exception | None = None) -> None:
        self.fail = fail
        self.templates: list[tuple[str, str, str, list[str], list[str]]] = []
        self.buttons: list[tuple[str, str, list[tuple[str, str]]]] = []
        self.texts: list[tuple[str, str]] = []

    async def send_text(self, phone: str, message: str, callback_ref: str = "") -> None:
        if self.fail:
            raise self.fail
        self.texts.append((phone, message))

    async def send_template(
        self, phone: str, name: str, language: str, body: list[str], button_payloads: list[str]
    ) -> None:
        if self.fail:
            raise self.fail
        self.templates.append((phone, name, language, body, button_payloads))

    async def send_buttons(self, phone: str, body: str, buttons: list[tuple[str, str]]) -> None:
        if self.fail:
            raise self.fail
        self.buttons.append((phone, body, buttons))


TEMPLATES = DeliveryTemplates(
    offer="fudia_delivery_oferta", ready="fudia_delivery_listo", cancelled="fudia_delivery_cancelado"
)


@pytest.mark.asyncio
async def test_relay_sends_templates_with_button_payloads_and_confirms() -> None:
    source = Source([message(), message(id="msg-2", kind="ready"), message(id="msg-3", kind="cancelled")])
    sender = Sender()
    assert await PartnerMessageRelay(source, sender, TEMPLATES).poll_once() == 3
    offer, ready, cancelled = sender.templates
    assert offer[:3] == (CENTRAL, "fudia_delivery_oferta", "es")
    assert offer[4] == [f"fdx:accept:{TOKEN}", f"fdx:reject:{TOKEN}"]
    assert ready[1] == "fudia_delivery_listo" and ready[4] == [f"fdx:pickup:{TOKEN}"]
    assert cancelled[1] == "fudia_delivery_cancelado" and cancelled[4] == []
    assert source.dispatched == ["msg-1", "msg-2", "msg-3"]


@pytest.mark.asyncio
async def test_relay_without_templates_sends_interactive_buttons() -> None:
    source = Source([message(), message(id="msg-2", kind="ready"), message(id="msg-3", kind="cancelled")])
    sender = Sender()
    assert await PartnerMessageRelay(source, sender, DeliveryTemplates()).poll_once() == 3
    (phone, body, buttons), (_, ready_body, ready_buttons) = sender.buttons
    assert phone == CENTRAL and body.startswith("🛵 Nuevo delivery PED-0142")
    assert buttons == [(f"fdx:accept:{TOKEN}", "Aceptar"), (f"fdx:reject:{TOKEN}", "Rechazar")]
    assert ready_buttons == [(f"fdx:pickup:{TOKEN}", "Recoger")] and "listo para recoger" in ready_body
    assert len(sender.texts) == 1 and "No lo recojas" in sender.texts[0][1]


@pytest.mark.asyncio
async def test_relay_retries_failed_sends_but_confirms_uncertain_ones() -> None:
    source = Source([message()])
    assert await PartnerMessageRelay(source, Sender(RuntimeError("meta down")), TEMPLATES).poll_once() == 0
    assert source.dispatched == []
    # Meta pudo haberla recibido: no se repite una oferta que corre contra el reloj.
    await PartnerMessageRelay(source, Sender(WhatsAppDeliveryUncertain("timeout")), TEMPLATES).poll_once()
    assert source.dispatched == ["msg-1"]


def webhook(*messages: dict[str, object]) -> dict[str, object]:
    return {
        "entry": [
            {"changes": [{"value": {"metadata": {"phone_number_id": "meta-1"}, "messages": list(messages)}}]}
        ]
    }


def test_webhook_routes_central_buttons_and_keeps_other_quick_replies_as_text() -> None:
    payload = webhook(
        {
            "id": "b1",
            "from": CENTRAL,
            "type": "button",
            "button": {"payload": f"fdx:accept:{TOKEN}", "text": "Aceptar"},
        },
        {
            "id": "b2",
            "from": CENTRAL,
            "type": "interactive",
            "interactive": {
                "type": "button_reply",
                "button_reply": {"id": f"fdx:pickup:{TOKEN}", "title": "Recoger"},
            },
        },
        {
            "id": "b3",
            "from": "51911111111",
            "type": "button",
            "button": {"payload": "otro", "text": "Ver carta"},
        },
    )
    assert _incoming_partner_actions(payload) == [
        ("b1", CENTRAL, PartnerAction(action="accept", token=TOKEN), "Aceptar", "meta-1"),
        ("b2", CENTRAL, PartnerAction(action="pickup", token=TOKEN), "Recoger", "meta-1"),
    ]
    assert _incoming_text_messages(payload) == [("b3", "51911111111", "Ver carta", "meta-1")]


class ActionFudia(FakeFudia):
    def __init__(self, outcome: str = "accepted") -> None:
        super().__init__()
        self.outcome = outcome
        self.actions: list[tuple[str, str, str]] = []

    async def apply_delivery_action(self, action: str, token: str, phone: str) -> DeliveryActionResult:
        self.actions.append((action, token, phone))
        return DeliveryActionResult(outcome=self.outcome, orderCode="PED-0142", locationName="Miraflores")


class NoAgent:
    async def respond(self, *args: object, **kwargs: object) -> str:  # pragma: no cover - never called
        raise AssertionError("a central button must not reach the model")


@pytest.mark.asyncio
async def test_service_applies_the_button_without_model_or_session() -> None:
    store = MemoryConversationStore()
    fudia = ActionFudia()
    service = ConciergeService(store, fudia, NoAgent())  # type: ignore[arg-type]
    inbound = InboundMessage(
        message_id="b1",
        phone=CENTRAL,
        text="Aceptar",
        partner_action=PartnerAction(action="accept", token=TOKEN),
    )
    record = await service.process_inbound(inbound)
    assert fudia.actions == [("accept", TOKEN, CENTRAL)]
    assert record.stage == "reply_ready" and record.reply.startswith("✅ El pedido PED-0142 es tuyo.")
    assert record.conversation_id == ""
    assert await store.get(CENTRAL) is None

    # El reintento del mismo mensaje responde lo ya preparado sin volver a llamar al backend.
    again = await service.process_inbound(inbound)
    assert again.reply == record.reply and len(fudia.actions) == 1
