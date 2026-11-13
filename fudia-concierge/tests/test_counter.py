"""Venta por mostrador y consentimiento promocional en Concierge."""

from __future__ import annotations

from typing import Any

import pytest

from src.domain.models import (
    ConversationSession,
    CustomerProfile,
    EffectRecord,
    ProcessingRecord,
    TableContext,
)
from src.infrastructure.state_store import MemoryConversationStore
from src.services.concierge_service import ConciergeService, agent_message
from src.services.consent import (
    asks_to_withdraw,
    consent_answer,
    consent_question,
)
from src.services.processing import recovery_reply
from src.services.start_codes import looks_like_start_message, start_message
from src.services.tools import ConciergeTools
from tests.test_concierge_service import FakeFudia

PHONE = "51999999999"
QR = "FUDIA:" + "a" * 32
QUESTION = (
    "¿Te gustaría recibir ofertas, promociones y beneficios exclusivos de "
    "Restaurante Demo por WhatsApp?\n\n- Sí, quiero recibirlos\n- No, gracias"
)


class CounterFudia(FakeFudia):
    """foods-backend with a counter QR: every requestId is a new order."""

    def __init__(self, *, known: bool = True, consent: str = "granted") -> None:
        super().__init__()
        self.known = known
        self.consent_status = consent  # type: ignore[assignment]
        self.orders: dict[str, str] = {}

    async def resolve_table(self, token: str) -> TableContext:
        table = await super().resolve_table(token)
        return table.model_copy(update={"name": "Mostrador", "serviceType": "counter"})

    async def get_customer(self, token: str, phone: str) -> CustomerProfile:
        if not self.known:
            return CustomerProfile(marketingConsent=self.consent_status)
        return (await super().get_customer(token, phone)).model_copy(update={"customerName": "Ana Quispe"})

    async def create_order(
        self,
        token: str,
        phone: str,
        lines: list[dict[str, Any]],
        conversation_id: str,
        request_id: str,
        customer_name: str = "",
    ) -> Any:
        result = await super().create_order(
            token, phone, lines, conversation_id, request_id, customer_name
        )
        self.order_calls[-1]["customer_name"] = customer_name
        self.orders.setdefault(request_id, f"o{len(self.orders) + 1}")
        number = len(self.orders)
        return result.model_copy(
            update={"id": self.orders[request_id], "code": f"PED-00{number}"}
        )


class OrderingAgent:
    """Adds one product, prepares the summary and confirms on «sí»."""

    def __init__(self) -> None:
        self.messages: list[str] = []
        self.results: list[dict[str, Any]] = []

    async def reply(self, session, user_message, tool_handler) -> str:
        self.messages.append(user_message)
        text = user_message.rsplit("Mensaje del cliente:\n", 1)[-1].lower()
        if text.startswith("soy "):
            self.results.append(
                await tool_handler(
                    "set_delivery_details", {"customerName": text[4:].title()}
                )
            )
            return "Gracias. ¿Qué deseas pedir?"
        if text.startswith("quiero"):
            await tool_handler("add_item", {"productId": "p1", "quantity": 1})
            self.results.append(await tool_handler("prepare_confirmation", {}))
            return "¿Confirmas tu pedido?"
        if text == "sí":
            self.results.append(await tool_handler("confirm_order", {}))
            return "Tu pedido fue registrado."
        if text == "la cuenta":
            self.results.append(await tool_handler("request_bill", {}))
            return "En el mostrador cada pedido se paga en caja."
        return "¿Algo más?"


def counter_service(
    fudia: FakeFudia, agent: OrderingAgent
) -> tuple[ConciergeService, MemoryConversationStore]:
    store = MemoryConversationStore()
    return ConciergeService(store, fudia, agent), store


def test_counter_start_message_keeps_its_name_and_is_recognized() -> None:
    message = start_message("Mostrador", "K7P2", "counter")
    assert message.endswith("\nMostrador · K7P2")
    assert "Mesa" not in message
    assert looks_like_start_message("Mostrador · K7P2")
    assert start_message("7", "K7P2").endswith("\nMesa 7 · K7P2")


@pytest.mark.asyncio
async def test_counter_qr_greets_a_returning_customer_and_opens_counter_mode() -> None:
    fudia = CounterFudia()
    agent = OrderingAgent()
    service, store = counter_service(fudia, agent)

    reply = await service.handle_message(PHONE, QR)

    assert reply.startswith(
        "🍽️ Bienvenido a Restaurante Demo\n📍 Mostrador\n\n"
        "Hola, Ana Quispe. Pide aquí, recoge en el mostrador y paga en caja."
    )
    assert reply.endswith("¿Qué deseas pedir?")
    session = await store.get(PHONE)
    assert session is not None and session.mode == "counter"
    assert session.delivery.customer_name == "Ana Quispe"
    assert fudia.opened[-1]["kind"] == "table"


@pytest.mark.asyncio
async def test_new_counter_customer_is_asked_for_a_name_before_ordering() -> None:
    fudia = CounterFudia(known=False)
    agent = OrderingAgent()
    service, store = counter_service(fudia, agent)

    reply = await service.handle_message(PHONE, QR)
    assert reply.endswith("Para empezar, escríbeme tus nombres y apellidos.")

    await service.handle_message(PHONE, "Quiero un lomo")
    assert agent.results[-1]["code"] == "customer_name_required"
    assert "Nombre del cliente: pendiente" in agent.messages[-1]
    assert fudia.order_calls == []

    await service.handle_message(PHONE, "Soy Rosa Flores")
    session = await store.get(PHONE)
    assert session is not None and session.delivery.customer_name == "Rosa Flores"
    await service.handle_message(PHONE, "Quiero un lomo")
    await service.handle_message(PHONE, "Sí")
    assert fudia.order_calls[-1]["customer_name"] == "Rosa Flores"


@pytest.mark.asyncio
async def test_each_counter_order_is_independent_and_has_no_bill() -> None:
    fudia = CounterFudia()
    agent = OrderingAgent()
    service, store = counter_service(fudia, agent)
    await service.handle_message(PHONE, QR)

    await service.handle_message(PHONE, "Quiero un lomo")
    await service.handle_message(PHONE, "Sí")
    first = agent.results[-1]
    assert first["ok"] is True and first["order"]["code"] == "PED-001"
    assert first["payment"] == "Se recoge en el mostrador y se paga en caja."

    await service.handle_message(PHONE, "Quiero un lomo")
    await service.handle_message(PHONE, "Sí")
    second = agent.results[-1]
    assert second["order"]["code"] == "PED-002"
    assert second["order"]["id"] != first["order"]["id"]
    assert fudia.order_calls[0]["request_id"] != fudia.order_calls[1]["request_id"]
    assert {call["customer_name"] for call in fudia.order_calls} == {"Ana Quispe"}

    session = await store.get(PHONE)
    assert session is not None and session.mode == "counter" and session.cart == []
    assert "Modo: venta por mostrador en Restaurante Demo" in agent.messages[-1]

    assert session is not None
    bill = await ConciergeTools(fudia, session, "la cuenta").execute("request_bill", {})
    assert bill["code"] == "counter_has_no_bill"
    assert fudia.bill_calls == []


@pytest.mark.asyncio
async def test_counter_mode_has_no_delivery() -> None:
    fudia = CounterFudia()
    session = ConversationSession(phone=PHONE, qr_token="a" * 32, mode="counter")
    tools = ConciergeTools(fudia, session, "para delivery")
    result = await tools.execute("set_fulfillment", {"type": "delivery"})
    assert result["code"] == "counter_pickup_only"
    named = await tools.execute(
        "set_delivery_details", {"customerName": "Luis Rojas", "address": "Av. 1"}
    )
    assert named["ok"] is True and named["counter"]["customerName"] == "Luis Rojas"
    assert session.delivery.address == ""


def test_agent_message_describes_the_counter_attention() -> None:
    session = ConversationSession(phone=PHONE, qr_token="a" * 32, mode="counter")
    message = agent_message(session, "Hola")
    assert "Modo: venta por mostrador" in message
    assert "se paga en caja" in message
    assert "Nombre del cliente: pendiente" in message
    assert message.endswith("Mensaje del cliente:\nHola")


def test_consent_question_and_answers() -> None:
    assert consent_question("Restaurante Demo") == QUESTION
    assert consent_answer("Sí, quiero recibirlos") == "granted"
    assert consent_answer("si") == "granted"
    assert consent_answer("No, gracias") == "declined"
    assert consent_answer("no") == "declined"
    assert consent_answer("quiero otro lomo") is None
    assert consent_answer("sí, y agrégame una gaseosa") is None
    assert asks_to_withdraw("Ya no quiero promociones")
    assert asks_to_withdraw("quiero darme de baja de las ofertas")
    assert not asks_to_withdraw("no quiero cebolla")


@pytest.mark.asyncio
async def test_consent_is_asked_once_after_the_first_order_and_answered() -> None:
    fudia = CounterFudia(consent="pending")
    agent = OrderingAgent()
    service, store = counter_service(fudia, agent)
    await service.handle_message(PHONE, QR)

    await service.handle_message(PHONE, "Quiero un lomo")
    reply = await service.handle_message(PHONE, "Sí")
    assert reply == f"Tu pedido fue registrado.\n\n{QUESTION}"
    asked = fudia.consent_calls[-1]
    assert asked["decision"] == "asked" and asked["evidence"] == QUESTION

    answered = await service.handle_message(PHONE, "Sí, quiero recibirlos")
    assert answered.startswith("Listo, te enviaremos ofertas y beneficios de Restaurante Demo")
    granted = fudia.consent_calls[-1]
    assert granted["decision"] == "granted"
    assert granted["evidence"] == "Sí, quiero recibirlos"
    assert len(agent.results) == 2  # the answer never reached the model

    await service.handle_message(PHONE, "Quiero un lomo")
    second = await service.handle_message(PHONE, "Sí")
    assert "¿Te gustaría recibir ofertas" not in second
    assert [call["decision"] for call in fudia.consent_calls] == ["asked", "granted"]

    withdrawn = await service.handle_message(PHONE, "Ya no quiero promociones")
    assert withdrawn == "Listo, ya no te enviaremos promociones de Restaurante Demo por WhatsApp."
    assert fudia.consent_calls[-1]["decision"] == "withdrawn"
    session = await store.get(PHONE)
    assert session is not None and session.consent_prompt_pending is False


@pytest.mark.asyncio
async def test_ignoring_the_consent_question_does_not_interrupt_the_order() -> None:
    fudia = CounterFudia(consent="pending")
    agent = OrderingAgent()
    service, store = counter_service(fudia, agent)
    await service.handle_message(PHONE, QR)
    await service.handle_message(PHONE, "Quiero un lomo")
    await service.handle_message(PHONE, "Sí")

    reply = await service.handle_message(PHONE, "Quiero un lomo")
    assert reply == "¿Confirmas tu pedido?"
    assert [call["decision"] for call in fudia.consent_calls] == ["asked"]
    session = await store.get(PHONE)
    assert session is not None and session.consent_prompt_pending is False
    assert session.cart and session.awaiting_confirmation


@pytest.mark.asyncio
async def test_already_answered_consent_is_never_asked_again() -> None:
    fudia = CounterFudia(consent="declined")
    agent = OrderingAgent()
    service, _ = counter_service(fudia, agent)
    await service.handle_message(PHONE, QR)
    await service.handle_message(PHONE, "Quiero un lomo")
    reply = await service.handle_message(PHONE, "Sí")
    assert reply == "Tu pedido fue registrado."
    assert fudia.consent_calls == []


@pytest.mark.asyncio
async def test_table_flow_is_unchanged_and_never_asks_consent() -> None:
    fudia = FakeFudia()
    fudia.consent_status = "pending"
    agent = OrderingAgent()
    store = MemoryConversationStore()
    service = ConciergeService(store, fudia, agent)
    welcome = await service.handle_message(PHONE, QR)
    assert "📍 Mesa M1" in welcome and "mostrador" not in welcome

    await service.handle_message(PHONE, "Quiero un lomo")
    reply = await service.handle_message(PHONE, "Sí")
    assert reply == "Tu pedido fue registrado."
    assert fudia.consent_calls == []
    assert "customer_name" not in fudia.order_calls[-1] or not fudia.order_calls[-1].get("customer_name")
    session = await store.get(PHONE)
    assert session is not None and session.mode == "table"


def test_recovery_reply_rebuilds_counter_order_and_question() -> None:
    session = ConversationSession(phone=PHONE, qr_token="a" * 32, mode="counter")
    record = ProcessingRecord(
        message_id="m1",
        effects=[
            EffectRecord(
                kind="create_order",
                request_id="r1",
                payload={"mode": "counter", "customerName": "Ana Quispe"},
                result={"order": {"code": "PED-009", "total": "25.00"}},
            ),
            EffectRecord(
                kind="marketing_consent",
                request_id="consent-ask-c1",
                payload={"decision": "asked"},
                result={"decision": "asked", "question": QUESTION},
            ),
        ],
    )
    assert recovery_reply(record, session) == (
        "Tu pedido PED-009 fue registrado. Total: 25.00. "
        f"Recógelo en el mostrador y págalo en caja.\n\n{QUESTION}"
    )


@pytest.mark.asyncio
async def test_withdrawal_works_outside_the_counter() -> None:
    fudia = FakeFudia()
    agent = OrderingAgent()
    service = ConciergeService(MemoryConversationStore(), fudia, agent)
    await service.handle_message(PHONE, QR)
    reply = await service.handle_message(PHONE, "Ya no quiero promociones")
    assert reply == "Listo, ya no te enviaremos promociones de Restaurante Demo por WhatsApp."
    assert fudia.consent_calls[-1]["decision"] == "withdrawn"
    assert agent.messages == []
