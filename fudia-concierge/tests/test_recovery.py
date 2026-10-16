"""Regression tests for recovery, idempotency, ordering and delivery.

Each test names the acceptance criterion it covers. They use the in-process
store, lock and queue; the Redis scripts are covered by
tests/test_redis_integration.py against a real Redis.
"""

from __future__ import annotations

import asyncio
from decimal import Decimal
from typing import Any

import httpx
import pytest

from src.api.webhook import _outbound_statuses
from src.domain.models import (
    BillSummary,
    ConversationSession,
    InboundMessage,
    MenuItem,
    OrderResult,
    ProcessingRecord,
    TableContext,
)
from src.infrastructure.concurrency import MemoryConversationLock
from src.infrastructure.queue import MemoryInboundQueue, retention_floor
from src.infrastructure.state_store import (
    MemoryConversationStore,
    RedisConversationStore,
)
from src.infrastructure.whatsapp_adapter import MetaWhatsAppAdapter
from src.services.concierge_service import ConciergeService
from src.services.outbox import DeliveryOutcome, ReplyDispatcher
from src.services.ports import (
    ConversationLease,
    LockLostError,
    WhatsAppDeliveryUncertain,
    conversation_identity,
    message_ref,
)
from src.services.queue_worker import QueueWorker

PHONE = "51911111111"
CHANNEL = "meta-phone"
TOKEN_A = "a" * 32
TOKEN_B = "b" * 32


class WorkerDied(RuntimeError):
    """Stands in for a crash: nothing after it runs in this attempt."""


class Backend:
    """foods-backend fake with the real idempotency contract: repeating a
    `requestId` returns the round already created."""

    def __init__(self) -> None:
        self.orders: dict[str, OrderResult] = {}
        self.order_calls: list[str] = []
        self.handoff_calls = 0
        self.bill_calls = 0
        self.bill_requests: list[str] = []
        self.die_after_order = False
        self.die_after_bill = False

    async def resolve_table(self, token: str) -> TableContext:
        return TableContext(
            name="M1" if token == TOKEN_A else "M9",
            seats=4,
            organizationName=(
                "Restaurante A" if token == TOKEN_A else "Restaurante B"
            ),
            locationName="Local",
            conciergeEnabled=True,
        )

    async def get_product(self, token: str, product_id: str) -> MenuItem:
        return MenuItem(
            productId=product_id,
            name=f"Producto {product_id}",
            price=Decimal("10.00"),
        )

    async def create_order(
        self,
        token: str,
        phone: str,
        lines: list[dict[str, Any]],
        conversation_id: str,
        request_id: str,
    ) -> OrderResult:
        self.order_calls.append(request_id)
        if request_id not in self.orders:
            self.orders[request_id] = OrderResult(
                id=f"o{len(self.orders) + 1}",
                code=f"PED-{len(self.orders) + 1:03d}",
                status="confirmado",
                total=Decimal("10.00"),
            )
        if self.die_after_order:
            self.die_after_order = False
            raise WorkerDied("backend confirmed, worker died before saving")
        return self.orders[request_id]

    async def request_handoff(
        self, token: str, phone: str, conversation_id: str, reason: str
    ) -> dict[str, Any]:
        self.handoff_calls += 1
        return {"status": "pending", "tableName": "M1"}

    async def get_handoff_status(
        self, token: str, conversation_id: str
    ) -> dict[str, Any]:
        return {"status": "resolved"}

    async def request_bill(
        self, token: str, phone: str, conversation_id: str, request_id: str = ""
    ) -> BillSummary:
        self.bill_calls += 1
        # foods-backend records each requestId once (audit_log unique index).
        if request_id not in self.bill_requests:
            self.bill_requests.append(request_id)
        if self.die_after_bill:
            self.die_after_bill = False
            raise WorkerDied("backend recorded the bill request, worker died")
        return BillSummary(
            orderId="o1",
            code="PED-001",
            tableName="M1",
            status="abierta",
            currencySymbol="S/",
            total=Decimal("20.00"),
            paidAmount=Decimal("0"),
            remainingAmount=Decimal("20.00"),
            paymentStatus="pending",
        )


class ScriptedAgent:
    """Runs a fixed list of tool calls, like a model would, then answers."""

    def __init__(
        self,
        steps: list[tuple[str, dict[str, Any]]],
        reply: str = "Listo.",
        die_after_tools: bool = False,
        block: asyncio.Event | None = None,
    ) -> None:
        self.steps = steps
        self.reply_text = reply
        self.die_after_tools = die_after_tools
        self.block = block
        self.calls = 0
        self.started = asyncio.Event()

    async def reply(self, session, user_message, tool_handler) -> str:  # type: ignore[no-untyped-def]
        self.calls += 1
        self.started.set()
        for index, (name, args) in enumerate(self.steps):
            await tool_handler(name, args)
            if self.block is not None and index == 0:
                await self.block.wait()
        if self.die_after_tools:
            raise WorkerDied("model call interrupted")
        return self.reply_text


class FlakyWhatsApp:
    def __init__(self, failures: int = 0, uncertain: int = 0) -> None:
        self.failures = failures
        self.uncertain = uncertain
        self.sent: list[tuple[str, str, str]] = []
        self.attempts = 0

    async def send_text(
        self, phone: str, message: str, callback_ref: str = ""
    ) -> None:
        self.attempts += 1
        if self.uncertain:
            self.uncertain -= 1
            raise WhatsAppDeliveryUncertain("timeout after write")
        if self.failures:
            self.failures -= 1
            raise RuntimeError("Meta 503")
        self.sent.append((phone, message, callback_ref))


def message(message_id: str, text: str, sequence: int = 0) -> InboundMessage:
    return InboundMessage(
        message_id=message_id,
        phone=PHONE,
        text=text,
        sender_phone_id=CHANNEL,
        sequence=sequence,
    )


class FixedRouter:
    def __init__(self, route: str) -> None:
        self.value = route

    async def route(self, session, user_message) -> str:  # type: ignore[no-untyped-def]
        return self.value


SERVICE_TOOLS = {"request_bill", "request_human"}


def service(
    store: MemoryConversationStore,
    backend: Backend,
    agent: ScriptedAgent,
    lock: MemoryConversationLock | None = None,
    heartbeat: float = 5.0,
) -> ConciergeService:
    # Route to the specialist that is allowed to run the scripted tools.
    intent = (
        "service"
        if agent.steps and agent.steps[0][0] in SERVICE_TOOLS
        else "order"
    )
    return ConciergeService(
        store,
        backend,  # type: ignore[arg-type]
        FixedRouter(intent),  # type: ignore[arg-type]
        {"menu": agent, "order": agent, "service": agent},
        conversation_lock=lock,
        lock_heartbeat_seconds=heartbeat,
    )


async def open_table(
    store: MemoryConversationStore,
    backend: Backend,
    token: str = TOKEN_A,
    sequence: int = 0,
) -> None:
    starter = service(store, backend, ScriptedAgent([]))
    await starter.process_inbound(
        message(f"qr-{token}", f"FUDIA:{token}", sequence)
    )


async def session_of(store: MemoryConversationStore) -> ConversationSession:
    session = await store.get(PHONE, CHANNEL)
    assert session is not None
    return session


ADD_P1 = ("add_item", {"productId": "p1", "quantity": 1, "note": ""})


@pytest.mark.asyncio
async def test_1_duplicate_message_does_not_add_the_product_twice() -> None:
    store, backend = MemoryConversationStore(), Backend()
    await open_table(store, backend)
    agent = ScriptedAgent([ADD_P1])
    concierge = service(store, backend, agent)

    first = await concierge.process_inbound(message("wamid.add", "un p1"))
    again = await concierge.process_inbound(message("wamid.add", "un p1"))

    assert agent.calls == 1
    assert again.reply == first.reply
    session = await session_of(store)
    assert [line.quantity for line in session.cart] == [1]


@pytest.mark.asyncio
async def test_1_queue_rejects_a_redelivered_message_id() -> None:
    queue = MemoryInboundQueue()
    assert await queue.enqueue(message("wamid.dup", "hola")) is True
    assert await queue.enqueue(message("wamid.dup", "hola")) is False


@pytest.mark.asyncio
async def test_2_whatsapp_failure_resends_without_running_tools_again() -> None:
    store, backend = MemoryConversationStore(), Backend()
    await open_table(store, backend)
    agent = ScriptedAgent([ADD_P1], reply="Agregué p1.")
    whatsapp = FlakyWhatsApp(failures=1)
    queue = MemoryInboundQueue()
    worker = QueueWorker(
        queue, service(store, backend, agent), whatsapp, records=store
    )
    await queue.enqueue(message("wamid.send", "un p1"))

    await worker.process((await queue.read())[0])  # send fails -> retry
    await worker.process((await queue.read())[0])  # only resends

    assert agent.calls == 1
    assert [sent[1] for sent in whatsapp.sent] == ["Agregué p1."]
    assert queue.status["wamid.send"] == "completed"
    session = await session_of(store)
    assert [line.quantity for line in session.cart] == [1]
    record = await store.get_record("wamid.send")
    assert record is not None and record.stage == "sent"
    assert record.send_attempts == 2


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("steps", "expected"),
    [
        ([ADD_P1], "Actualicé tu pedido"),
        ([ADD_P1, ("prepare_confirmation", {})], "¿Confirmas este pedido?"),
        ([("request_human", {"reason": "mozo"})], "personal del local"),
        ([("request_bill", {})], "saldo pendiente S/ 20.00"),
    ],
)
async def test_3_restart_answers_from_persisted_effects(
    steps: list[tuple[str, dict[str, Any]]],
    expected: str,
) -> None:
    store, backend = MemoryConversationStore(), Backend()
    await open_table(store, backend)
    crashed = ScriptedAgent(steps, die_after_tools=True)
    with pytest.raises(WorkerDied):
        await service(store, backend, crashed).process_inbound(
            message("wamid.crash", "pedido")
        )

    restarted = ScriptedAgent(steps)
    record = await service(store, backend, restarted).process_inbound(
        message("wamid.crash", "pedido")
    )

    assert restarted.calls == 0
    assert expected in record.reply
    assert record.stage == "reply_ready"
    session = await session_of(store)
    assert sum(line.quantity for line in session.cart) == (
        1 if steps[0] == ADD_P1 else 0
    )
    assert backend.handoff_calls <= 1
    assert backend.bill_calls <= 1


@pytest.mark.asyncio
async def test_4_confirmed_round_keeps_its_request_id_after_a_crash() -> None:
    store, backend = MemoryConversationStore(), Backend()
    await open_table(store, backend)
    await service(store, backend, ScriptedAgent([ADD_P1])).process_inbound(
        message("wamid.cart", "un p1")
    )
    # Prepare and confirm in the same turn: the request id only existed in
    # memory before this change.
    confirm = [("prepare_confirmation", {}), ("confirm_order", {})]
    backend.die_after_order = True
    with pytest.raises(WorkerDied):
        await service(store, backend, ScriptedAgent(confirm)).process_inbound(
            message("wamid.yes", "sí")
        )

    restarted = ScriptedAgent(confirm)
    record = await service(store, backend, restarted).process_inbound(
        message("wamid.yes", "sí")
    )

    assert restarted.calls == 0
    assert len(backend.orders) == 1
    assert len(set(backend.order_calls)) == 1
    assert len(backend.order_calls) == 2
    assert "PED-001" in record.reply
    session = await session_of(store)
    assert session.cart == []
    assert session.pending_order_request_id is None
    assert session.last_order_id == "o1"


@pytest.mark.asyncio
async def test_4_bill_request_keeps_its_request_id_after_a_crash() -> None:
    store, backend = MemoryConversationStore(), Backend()
    await open_table(store, backend)
    backend.die_after_bill = True
    steps = [("request_bill", {})]
    with pytest.raises(WorkerDied):
        await service(store, backend, ScriptedAgent(steps)).process_inbound(
            message("wamid.bill", "la cuenta")
        )

    restarted = ScriptedAgent(steps)
    record = await service(store, backend, restarted).process_inbound(
        message("wamid.bill", "la cuenta")
    )

    assert restarted.calls == 0
    assert backend.bill_calls == 2
    assert len(backend.bill_requests) == 1
    assert backend.bill_requests[0]
    assert "saldo pendiente S/ 20.00" in record.reply


@pytest.mark.asyncio
async def test_10_max_resends_zero_never_duplicates_an_uncertain_reply() -> None:
    records = MemoryConversationStore()
    whatsapp = FlakyWhatsApp(uncertain=1)
    now = [1_000.0]
    dispatcher = ReplyDispatcher(
        records, whatsapp, reconcile_seconds=30, max_resends=0, clock=lambda: now[0]
    )
    inbound = message("wamid.no-resend", "hola")
    record = ProcessingRecord(
        message_id=inbound.message_id, stage="reply_ready", reply="Hola"
    )

    assert await dispatcher.deliver(inbound, record) is DeliveryOutcome.DEFERRED
    now[0] += 31
    assert await dispatcher.deliver(inbound, record) is DeliveryOutcome.FAILED
    assert whatsapp.attempts == 1


@pytest.mark.asyncio
async def test_5_stale_lease_cannot_write() -> None:
    store = MemoryConversationStore()
    lock = MemoryConversationLock()
    identity = conversation_identity(PHONE, CHANNEL)
    old = ConversationLease(lock, identity, await lock.acquire(identity))
    lock.expire(identity)
    new = ConversationLease(lock, identity, await lock.acquire(identity))
    newer_state = ConversationSession(phone=PHONE, channel_key=CHANNEL)
    await store.commit(new, newer_state)

    with pytest.raises(LockLostError):
        await store.commit(
            old,
            ConversationSession(phone=PHONE, channel_key=CHANNEL),
            ProcessingRecord(message_id="wamid.old"),
        )

    assert old.lost is True
    assert (await session_of(store)).conversation_id == newer_state.conversation_id
    assert await store.get_record("wamid.old") is None


@pytest.mark.asyncio
async def test_5_lost_lease_interrupts_the_old_worker() -> None:
    store, backend = MemoryConversationStore(), Backend()
    await open_table(store, backend)
    lock = MemoryConversationLock()
    release = asyncio.Event()
    slow = ScriptedAgent(
        [ADD_P1, ("add_item", {"productId": "p2", "quantity": 1})],
        block=release,
    )
    old_worker = service(store, backend, slow, lock=lock, heartbeat=0.01)
    task = asyncio.create_task(
        old_worker.process_inbound(message("wamid.slow", "p1 y p2"))
    )
    await slow.started.wait()
    lock.expire(conversation_identity(PHONE, CHANNEL))

    with pytest.raises(LockLostError):
        await asyncio.wait_for(task, timeout=2)
    release.set()

    new_worker = service(
        store, backend, ScriptedAgent([("add_item", {"productId": "p3", "quantity": 1})]),
        lock=lock,
    )
    await new_worker.process_inbound(message("wamid.next", "p3"))
    await asyncio.sleep(0.05)

    session = await session_of(store)
    # p1 was checkpointed before the lease was lost; p2 never ran.
    assert [line.product_id for line in session.cart] == ["p1", "p3"]


@pytest.mark.asyncio
async def test_6_distinct_conversations_progress_in_parallel_without_mixing() -> None:
    store, backend = MemoryConversationStore(), Backend()
    release = asyncio.Event()
    agents: list[ScriptedAgent] = []
    phones = ["51900000001", "51900000002"]
    for index, phone in enumerate(phones, start=1):
        starter = service(store, backend, ScriptedAgent([]))
        await starter.process_inbound(
            InboundMessage(
                message_id=f"qr-{phone}",
                phone=phone,
                text=f"FUDIA:{TOKEN_A}",
                sender_phone_id=CHANNEL,
            )
        )
        agents.append(
            ScriptedAgent(
                [("add_item", {"productId": f"p{index}", "quantity": index})],
                block=release,
            )
        )
    lock = MemoryConversationLock()
    tasks = [
        asyncio.create_task(
            service(store, backend, agent, lock=lock).process_inbound(
                InboundMessage(
                    message_id=f"wamid.{phone}",
                    phone=phone,
                    text="pedido",
                    sender_phone_id=CHANNEL,
                )
            )
        )
        for agent, phone in zip(agents, phones, strict=True)
    ]
    await asyncio.wait_for(
        asyncio.gather(*(agent.started.wait() for agent in agents)), timeout=1
    )
    release.set()
    await asyncio.gather(*tasks)

    for index, phone in enumerate(phones, start=1):
        session = await store.get(phone, CHANNEL)
        assert session is not None
        assert [(line.product_id, line.quantity) for line in session.cart] == [
            (f"p{index}", index)
        ]


class OrderedService:
    def __init__(self, fail_first: str = "") -> None:
        self.order: list[str] = []
        self.fail_first = fail_first

    async def process_inbound(self, inbound: InboundMessage) -> ProcessingRecord:
        await asyncio.sleep(0.02 if inbound.text == "1" else 0)
        if inbound.message_id == self.fail_first:
            self.fail_first = ""
            raise RuntimeError("model unavailable")
        self.order.append(inbound.text)
        return ProcessingRecord(
            message_id=inbound.message_id, stage="reply_ready", reply=inbound.text
        )


@pytest.mark.asyncio
@pytest.mark.parametrize("fail_first", ["", "wamid.order.1"])
async def test_7_messages_of_one_conversation_keep_their_order(
    fail_first: str,
) -> None:
    queue = MemoryInboundQueue()
    ordered = OrderedService(fail_first)
    whatsapp = FlakyWhatsApp()
    worker = QueueWorker(queue, ordered, whatsapp, max_concurrency=4)
    for index in ("1", "2", "3"):
        await queue.enqueue(message(f"wamid.order.{index}", index))

    task = asyncio.create_task(worker.run())
    while len(whatsapp.sent) < 3:
        await asyncio.sleep(0.01)
    await worker.stop()
    await asyncio.wait_for(task, timeout=2)

    assert ordered.order == ["1", "2", "3"]
    assert [sent[1] for sent in whatsapp.sent] == ["1", "2", "3"]


@pytest.mark.asyncio
async def test_8_new_qr_resets_cart_and_confirmation_of_the_previous_table() -> None:
    store, backend = MemoryConversationStore(), Backend()
    await open_table(store, backend, TOKEN_A)
    await service(
        store, backend, ScriptedAgent([ADD_P1, ("prepare_confirmation", {})])
    ).process_inbound(message("wamid.a", "p1"))
    before = await session_of(store)
    assert before.awaiting_confirmation is True

    await open_table(store, backend, TOKEN_B)

    after = await session_of(store)
    assert after.qr_token == TOKEN_B
    assert after.cart == []
    assert after.awaiting_confirmation is False
    assert after.pending_order_request_id is None
    assert after.conversation_id != before.conversation_id


@pytest.mark.asyncio
async def test_8_message_from_the_previous_table_does_not_reach_the_new_one() -> None:
    store, backend = MemoryConversationStore(), Backend()
    await open_table(store, backend, TOKEN_A, sequence=1)
    # Enqueued before the second QR (sequence 3) but delivered after it.
    late = message("wamid.late", "sí", sequence=2)
    await open_table(store, backend, TOKEN_B, sequence=3)
    agent = ScriptedAgent([ADD_P1])

    record = await service(store, backend, agent).process_inbound(late)

    assert record.stage == "superseded"
    assert agent.calls == 0
    assert (await session_of(store)).cart == []


@pytest.mark.asyncio
async def test_8_reply_prepared_for_the_previous_table_is_not_sent() -> None:
    store, backend = MemoryConversationStore(), Backend()
    await open_table(store, backend, TOKEN_A)
    record = await service(store, backend, ScriptedAgent([ADD_P1])).process_inbound(
        message("wamid.old-reply", "p1")
    )
    await open_table(store, backend, TOKEN_B)
    whatsapp = FlakyWhatsApp()

    outcome = await ReplyDispatcher(store, whatsapp).deliver(
        message("wamid.old-reply", "p1"), record
    )

    assert outcome is DeliveryOutcome.SUPERSEDED
    assert whatsapp.sent == []


def test_9_retention_never_trims_pending_or_undelivered_entries() -> None:
    cutoff = 10_000
    # Nothing pending, everything delivered: trim up to the cutoff.
    assert retention_floor(cutoff, "20000-0", None) == "10000-0"
    # Oldest pending entry is older than the cutoff: keep it.
    assert retention_floor(cutoff, "20000-0", "5000-3") == "5000-3"
    # Entries after the last delivered id were never read: keep them.
    assert retention_floor(cutoff, "7000-1", None) == "7000-1"


def test_9_records_needed_for_recovery_outlive_finished_ones() -> None:
    store = RedisConversationStore(
        "redis://localhost:6379/0",
        ttl_seconds=60,
        record_ttl_seconds=100,
        pending_record_ttl_seconds=1_000,
    )
    for stage, ttl in (
        ("effect_pending", 1_000),
        ("reply_ready", 1_000),
        ("uncertain", 1_000),
        ("sent", 100),
        ("superseded", 100),
    ):
        record = ProcessingRecord(message_id="m", stage=stage)  # type: ignore[arg-type]
        assert store._record_ttl(record) == ttl


@pytest.mark.asyncio
async def test_10_ambiguous_timeout_is_reconciled_with_meta_status() -> None:
    records = MemoryConversationStore()
    whatsapp = FlakyWhatsApp(uncertain=1)
    now = [1_000.0]
    dispatcher = ReplyDispatcher(
        records, whatsapp, reconcile_seconds=30, clock=lambda: now[0]
    )
    inbound = message("wamid.timeout", "hola")
    record = ProcessingRecord(
        message_id=inbound.message_id, stage="reply_ready", reply="Hola"
    )

    first = await dispatcher.deliver(inbound, record)
    assert first is DeliveryOutcome.DEFERRED
    assert record.stage == "uncertain"

    now[0] += 5
    assert await dispatcher.deliver(inbound, record) is DeliveryOutcome.DEFERRED

    await records.record_outbound_status(message_ref(inbound.message_id), "delivered")
    assert await dispatcher.deliver(inbound, record) is DeliveryOutcome.SENT
    assert whatsapp.attempts == 1  # never resent


@pytest.mark.asyncio
async def test_10_unconfirmed_send_is_resent_once_then_marked_failed() -> None:
    records = MemoryConversationStore()
    whatsapp = FlakyWhatsApp(uncertain=2)
    now = [1_000.0]
    dispatcher = ReplyDispatcher(
        records, whatsapp, reconcile_seconds=30, max_resends=1, clock=lambda: now[0]
    )
    inbound = message("wamid.lost", "hola")
    record = ProcessingRecord(
        message_id=inbound.message_id, stage="reply_ready", reply="Hola"
    )

    assert await dispatcher.deliver(inbound, record) is DeliveryOutcome.DEFERRED
    now[0] += 31
    assert await dispatcher.deliver(inbound, record) is DeliveryOutcome.DEFERRED
    assert record.resends == 1
    now[0] += 31
    assert await dispatcher.deliver(inbound, record) is DeliveryOutcome.FAILED
    assert whatsapp.attempts == 2
    assert record.last_error == "delivery_unconfirmed"


@pytest.mark.asyncio
async def test_10_adapter_separates_not_sent_from_uncertain() -> None:
    calls: list[dict[str, Any]] = []
    connect_failures = [1]

    async def handler(request: httpx.Request) -> httpx.Response:
        import json

        calls.append(json.loads(request.content))
        if connect_failures[0]:
            connect_failures[0] -= 1
            raise httpx.ConnectError("refused", request=request)
        raise httpx.ReadTimeout("no answer", request=request)

    client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    adapter = MetaWhatsAppAdapter("token", "phone-id", "v99.0", client=client)

    with pytest.raises(WhatsAppDeliveryUncertain):
        await adapter.send_text(PHONE, "Hola", callback_ref="f" * 64)

    # The connection error is retried; the read timeout is not.
    assert len(calls) == 2
    assert calls[-1]["biz_opaque_callback_data"] == "f" * 64
    await client.aclose()


def test_10_webhook_reads_status_receipts_for_our_references() -> None:
    ref = "a" * 64
    payload = {
        "entry": [
            {
                "changes": [
                    {
                        "value": {
                            "metadata": {"phone_number_id": "meta-phone"},
                            "statuses": [
                                {"status": "delivered", "biz_opaque_callback_data": ref},
                                {"status": "read", "biz_opaque_callback_data": "x"},
                                {"status": "sent"},
                            ],
                        }
                    }
                ]
            }
        ]
    }
    assert _outbound_statuses(payload, "meta-phone") == [(ref, "delivered")]
    assert _outbound_statuses(payload, "other-phone") == []


@pytest.mark.asyncio
async def test_4_bill_request_sends_request_id_and_retries_safely() -> None:
    import json

    from src.infrastructure.fudia_client import FudiaClient

    bodies: list[dict[str, Any]] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        bodies.append(json.loads(request.content))
        if len(bodies) == 1:
            return httpx.Response(503, json={"code": "unavailable"})
        return httpx.Response(
            200,
            json={
                "orderId": "o1",
                "code": "PED-001",
                "tableName": "M1",
                "status": "abierta",
                "currencySymbol": "S/",
                "items": [],
                "total": "20.00",
                "paidAmount": "0",
                "remainingAmount": "20.00",
                "paymentStatus": "pending",
            },
        )

    client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    fudia = FudiaClient("https://api.test", "key", client=client)

    bill = await fudia.request_bill(TOKEN_A, PHONE, "conv-1", "bill-key-1")

    assert bill.remainingAmount == Decimal("20.00")
    # Idempotent with a requestId: the 503 is retried with the same key.
    assert [body["requestId"] for body in bodies] == ["bill-key-1", "bill-key-1"]
    await client.aclose()
