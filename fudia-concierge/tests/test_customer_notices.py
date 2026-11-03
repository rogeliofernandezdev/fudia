"""Aviso «pedido listo» al cliente: de la bandeja de foods-backend a WhatsApp."""

from __future__ import annotations

import asyncio

import pytest

from src.domain.models import ChatMessage, ConversationSession, CustomerNotice, InboundMessage
from src.infrastructure.fudia_client import _endpoint_label
from src.infrastructure.queue import MemoryInboundQueue
from src.infrastructure.state_store import MemoryConversationStore
from src.services.concierge_service import ConciergeService
from src.services.customer_notices import (
    CUSTOMER_WINDOW_SECONDS,
    NOTICE_MESSAGE_PREFIX,
    NoticeRelay,
    order_ready_text,
)
from src.services.queue_worker import QueueWorker
from tests.test_concierge_service import FakeFudia
from tests.test_recovery import FlakyWhatsApp

PHONE = "51911111111"
NOW = 1_800_000_000.0


def notice(**changes: object) -> CustomerNotice:
    base = CustomerNotice(
        id="7f1c",
        orderId="o1",
        orderCode="PED-0142",
        fulfillment="counter",
        customerPhone=PHONE,
        conversationId="conv-1",
        organizationName="La Panchita",
        locationName="Miraflores",
    )
    return base.model_copy(update=changes)


class Source:
    def __init__(self, notices: list[CustomerNotice]) -> None:
        self.notices = notices
        self.dispatched: list[str] = []

    async def pending_customer_notices(self, limit: int = 20) -> list[CustomerNotice]:
        return self.notices[:limit]

    async def mark_customer_notice_dispatched(self, notice_id: str) -> None:
        self.dispatched.append(notice_id)


class BrokenQueue(MemoryInboundQueue):
    async def enqueue(self, message: InboundMessage) -> bool:
        raise ConnectionError("redis down")


class QuietAgent:
    async def respond(self, *args: object, **kwargs: object) -> str:
        return "Hola"


def test_order_ready_text_is_plain_and_names_the_venue() -> None:
    assert order_ready_text(notice()) == (
        "✅ Tu pedido PED-0142 está listo.\n"
        "Recógelo en el mostrador de La Panchita · Miraflores."
    )
    assert order_ready_text(notice(fulfillment="pickup")) == (
        "✅ Tu pedido PED-0142 está listo para recoger.\n"
        "Te esperamos en La Panchita · Miraflores."
    )
    assert order_ready_text(notice(orderCode="", organizationName="", locationName="")) == (
        "✅ Tu pedido está listo.\nRecógelo en el mostrador."
    )
    for text in (order_ready_text(notice()), order_ready_text(notice(fulfillment="pickup"))):
        assert "*" not in text


def test_notice_endpoints_have_their_own_metric_label() -> None:
    assert _endpoint_label("/v1/integrations/concierge/notices/pending") == "notices"
    assert _endpoint_label("/v1/integrations/concierge/notices/abc/dispatched") == "notices"


@pytest.mark.asyncio
async def test_relay_enqueues_each_notice_once_and_confirms_it() -> None:
    queue = MemoryInboundQueue()
    source = Source([notice(customerPhone="+51 911 111 111"), notice(id="sin-numero", customerPhone="")])
    relay = NoticeRelay(source, queue, channel_id="meta-phone")

    assert await relay.poll_once() == 2
    deliveries = await queue.read()
    assert len(deliveries) == 1
    message = deliveries[0].message
    assert message.message_id == NOTICE_MESSAGE_PREFIX + "7f1c"
    assert message.phone == PHONE and message.sender_phone_id == "meta-phone"
    assert message.notice is not None and message.notice.orderCode == "PED-0142"
    assert source.dispatched == ["7f1c", "sin-numero"]

    # Un reclamo vencido devuelve el mismo aviso: la cola no lo duplica y se confirma otra vez.
    source.notices = [notice()]
    assert await relay.poll_once() == 1
    assert await queue.read() == []
    assert source.dispatched[-1] == "7f1c"


@pytest.mark.asyncio
async def test_relay_does_not_confirm_a_notice_it_could_not_enqueue() -> None:
    source = Source([notice()])
    relay = NoticeRelay(source, BrokenQueue())
    assert await relay.poll_once() == 0
    assert source.dispatched == []


async def active_session(store: MemoryConversationStore, last_activity_at: float) -> None:
    await store.save(
        ConversationSession(
            phone=PHONE,
            last_activity_at=last_activity_at,
            messages=[ChatMessage(role="user", content="Un ceviche")],
        )
    )


def notice_message() -> InboundMessage:
    return InboundMessage(message_id=NOTICE_MESSAGE_PREFIX + "7f1c", phone=PHONE, text="", notice=notice())


@pytest.mark.asyncio
async def test_service_prepares_the_notice_and_keeps_it_in_the_history() -> None:
    store = MemoryConversationStore()
    await active_session(store, NOW - 600)
    service = ConciergeService(store, FakeFudia(), QuietAgent(), clock=lambda: NOW)  # type: ignore[arg-type]

    record = await service.process_inbound(notice_message())
    assert record.stage == "reply_ready"
    assert record.reply == order_ready_text(notice())
    assert record.conversation_id == ""
    session = await store.get(PHONE)
    assert session is not None
    assert session.messages[-1] == ChatMessage(role="assistant", content=record.reply)

    # Reintento del mismo aviso: misma respuesta, sin repetir el historial.
    again = await service.process_inbound(notice_message())
    assert again.reply == record.reply
    session = await store.get(PHONE)
    assert session is not None and len(session.messages) == 2


@pytest.mark.asyncio
async def test_service_skips_notices_outside_the_whatsapp_window() -> None:
    for prepare in ("no_session", "stale"):
        store = MemoryConversationStore()
        if prepare == "stale":
            await active_session(store, NOW - CUSTOMER_WINDOW_SECONDS - 1)
        service = ConciergeService(store, FakeFudia(), QuietAgent(), clock=lambda: NOW)  # type: ignore[arg-type]
        record = await service.process_inbound(notice_message())
        assert record.stage == "superseded" and record.reply == ""


@pytest.mark.asyncio
async def test_worker_relays_and_sends_the_notice_to_the_customer() -> None:
    store = MemoryConversationStore()
    await active_session(store, NOW - 60)
    service = ConciergeService(store, FakeFudia(), QuietAgent(), clock=lambda: NOW)  # type: ignore[arg-type]
    queue = MemoryInboundQueue()
    whatsapp = FlakyWhatsApp()
    source = Source([notice()])
    worker = QueueWorker(
        queue,
        service,
        whatsapp,
        records=store,
        notices=NoticeRelay(source, queue),
        notice_poll_seconds=0.01,
    )
    task = asyncio.create_task(worker.run())
    for _ in range(200):
        if whatsapp.sent:
            break
        await asyncio.sleep(0.01)
    source.notices = []
    await worker.stop()
    await asyncio.wait_for(task, timeout=2)

    assert [(phone, text) for phone, text, _ in whatsapp.sent] == [(PHONE, order_ready_text(notice()))]
    assert source.dispatched[0] == "7f1c"
