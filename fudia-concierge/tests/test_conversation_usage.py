"""Conversaciones contadas por foods-backend y tope mensual de Concierge."""

from __future__ import annotations

import json
from typing import Any

import httpx
import pytest

from src.domain.models import InboundMessage
from src.infrastructure.fudia_client import FudiaClient
from src.infrastructure.state_store import MemoryConversationStore
from src.services.concierge_service import (
    CONVERSATION_LIMIT_DELIVERY_REPLY,
    CONVERSATION_LIMIT_TABLE_REPLY,
    ConciergeService,
    conversation_id_for,
)
from src.services.ports import CONVERSATION_LIMIT_CODE, FudiaError, message_ref
from src.services.usage_meter import TokenMeter, metering, record_tokens
from tests.test_concierge_service import FakeFudia, QuietEngine
from tests.test_delivery import DeliveryFudia

PHONE = "51999999999"
TOKEN = "a" * 32
QR = "FUDIA:" + TOKEN


def inbound(message_id: str, text: str) -> InboundMessage:
    return InboundMessage(message_id=message_id, phone=PHONE, text=text)


class SpendingEngine:
    """Stands in for a model adapter: it records tokens like the real ones."""

    async def reply(self, session, user_message, tool_handler) -> str:
        record_tokens("lite", 120, 30)
        record_tokens("flash", 40, 10)
        record_tokens("lite", 5, 0)
        return "Te muestro la carta."


async def test_qr_opens_and_counts_one_table_conversation() -> None:
    store, fudia = MemoryConversationStore(), FakeFudia()
    service = ConciergeService(store, fudia, QuietEngine())

    await service.process_inbound(inbound("wamid.qr", QR))
    # Redelivery of the same message reuses the same conversation.
    await service.process_inbound(inbound("wamid.qr", QR))

    session = await store.get(PHONE)
    assert session is not None
    assert session.conversation_id == conversation_id_for("wamid.qr")
    assert fudia.opened == [
        {"token": TOKEN, "conversation_id": session.conversation_id, "kind": "table"}
    ]


async def test_limit_reached_answers_soberly_and_opens_nothing() -> None:
    store, fudia = MemoryConversationStore(), FakeFudia()
    fudia.conversation_limit_reached = True
    service = ConciergeService(store, fudia, QuietEngine())

    reply = await service.handle_message(PHONE, QR)

    assert reply == CONVERSATION_LIMIT_TABLE_REPLY
    assert await store.get(PHONE) is None
    lowered = reply.lower()
    for word in ("plan", "tope", "límite", "limite", "conversaciones"):
        assert word not in lowered
    assert "*" not in reply and "\n" not in reply


async def test_conversation_in_progress_is_never_cut() -> None:
    store, fudia = MemoryConversationStore(), FakeFudia()
    service = ConciergeService(store, fudia, QuietEngine())
    await service.handle_message(PHONE, QR)
    before = await store.get(PHONE)
    assert before is not None

    fudia.conversation_limit_reached = True
    reply = await service.handle_message(PHONE, "¿Qué platos tienen?")
    assert reply == "¿Qué deseas agregar?"

    # Scanning again would start a new conversation: refused, but the open
    # one stays exactly as it was and keeps answering.
    assert await service.handle_message(PHONE, QR) == CONVERSATION_LIMIT_TABLE_REPLY
    after = await store.get(PHONE)
    assert after is not None
    assert after.conversation_id == before.conversation_id
    assert after.qr_token == TOKEN
    assert await service.handle_message(PHONE, "Gracias") == "¿Qué deseas agregar?"


async def test_backend_failure_on_open_keeps_serving() -> None:
    class DownFudia(FakeFudia):
        async def open_conversation(
            self, token: str, conversation_id: str, kind: str
        ) -> None:
            raise FudiaError(503, "fudia_unreachable", "No pudimos comunicarnos.")

    store = MemoryConversationStore()
    service = ConciergeService(store, DownFudia(), QuietEngine())

    reply = await service.handle_message(PHONE, QR)

    assert reply != CONVERSATION_LIMIT_TABLE_REPLY
    session = await store.get(PHONE)
    assert session is not None and session.qr_token == TOKEN


async def test_delivery_after_the_table_is_a_new_conversation() -> None:
    store, fudia = MemoryConversationStore(), DeliveryFudia()
    service = ConciergeService(store, fudia, QuietEngine())
    await service.handle_message(PHONE, QR)
    session = await store.get(PHONE)
    assert session is not None
    session.last_order_id = "o1"
    await store.save(session)

    fudia.active = False
    fudia.conversation_limit_reached = True
    reply = await service.handle_message(PHONE, "Quiero pedir para llevar")
    assert reply == CONVERSATION_LIMIT_DELIVERY_REPLY
    unchanged = await store.get(PHONE)
    assert unchanged is not None and unchanged.mode == "table"

    fudia.conversation_limit_reached = False
    await service.process_inbound(inbound("wamid.delivery", "Quiero pedir para llevar"))
    delivery = await store.get(PHONE)
    assert delivery is not None and delivery.mode == "delivery"
    assert delivery.conversation_id == conversation_id_for("wamid.delivery")
    assert [o["kind"] for o in fudia.opened] == ["table", "delivery"]


async def test_turn_tokens_are_reported_per_tier_for_the_conversation() -> None:
    store, fudia = MemoryConversationStore(), FakeFudia()
    service = ConciergeService(store, fudia, SpendingEngine())
    await service.process_inbound(inbound("wamid.qr", QR))
    # The welcome uses no model: nothing to report.
    assert fudia.usage_reports == []

    await service.process_inbound(inbound("wamid.turn", "Quiero ver la carta"))

    session = await store.get(PHONE)
    assert session is not None
    assert fudia.usage_reports == [
        {
            "token": TOKEN,
            "conversation_id": session.conversation_id,
            "request_id": f"{message_ref('wamid.turn')[:32]}:1",
            "tiers": [
                {"tier": "flash", "inputTokens": 40, "outputTokens": 10},
                {"tier": "lite", "inputTokens": 125, "outputTokens": 30},
            ],
        }
    ]


async def test_usage_report_failure_never_breaks_the_reply() -> None:
    class FailingReports(FakeFudia):
        async def report_usage(self, *args: Any) -> None:
            raise FudiaError(503, "fudia_unreachable", "No pudimos comunicarnos.")

    store = MemoryConversationStore()
    service = ConciergeService(store, FailingReports(), SpendingEngine())
    await service.handle_message(PHONE, QR)

    assert await service.handle_message(PHONE, "Quiero ver la carta") == (
        "Te muestro la carta."
    )


def test_meter_is_scoped_to_the_turn() -> None:
    record_tokens("lite", 10, 10)  # outside a turn: ignored
    with metering() as meter:
        record_tokens("lite", 3, 1)
        record_tokens("flash", 0, 0)
    record_tokens("lite", 99, 99)
    assert meter.tiers() == [{"tier": "lite", "inputTokens": 3, "outputTokens": 1}]
    assert TokenMeter().tiers() == []


async def test_client_opens_conversations_and_reports_usage() -> None:
    calls: list[tuple[str, dict[str, Any]]] = []

    def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content)
        calls.append((request.url.path, body))
        assert request.headers["X-Fudia-Concierge-Key"] == "key"
        if body.get("conversationId") == "full":
            return httpx.Response(
                409,
                json={"code": CONVERSATION_LIMIT_CODE, "message": "Sin cupo."},
            )
        if request.url.path.endswith("/usage"):
            return httpx.Response(204)
        return httpx.Response(201, json={"conversationId": "c1", "kind": "table", "month": "2026-10"})

    client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    fudia = FudiaClient("https://api.test", "key", client=client)

    await fudia.open_conversation(TOKEN, "c1", "table")
    await fudia.report_usage(
        TOKEN, "c1", "r1", [{"tier": "lite", "inputTokens": 1, "outputTokens": 2}]
    )
    with pytest.raises(FudiaError) as rejected:
        await fudia.open_conversation(TOKEN, "full", "delivery")

    assert rejected.value.code == CONVERSATION_LIMIT_CODE
    assert calls[0] == (
        f"/v1/integrations/concierge/{TOKEN}/conversations",
        {"conversationId": "c1", "kind": "table"},
    )
    assert calls[1] == (
        f"/v1/integrations/concierge/{TOKEN}/conversations/c1/usage",
        {"requestId": "r1", "tiers": [{"tier": "lite", "inputTokens": 1, "outputTokens": 2}]},
    )
    await client.aclose()
