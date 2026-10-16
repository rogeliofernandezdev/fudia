"""Redis scripts against a real Redis: fencing, ownership, order, retention
and dead letters. Run with FUDIA_REDIS_INTEGRATION=1 and REDIS_URL."""

from __future__ import annotations

import asyncio
import os
from collections.abc import AsyncIterator
from uuid import uuid4

import pytest

from src.domain.models import ConversationSession, InboundMessage, ProcessingRecord
from src.infrastructure.concurrency import RedisConversationLock, lock_key
from src.infrastructure.queue import RedisInboundQueue
from src.infrastructure.state_store import RedisConversationStore
from src.services.ports import ConversationLease, LockLostError

REDIS_URL = os.getenv("REDIS_URL", "")
pytestmark = pytest.mark.skipif(
    os.getenv("FUDIA_REDIS_INTEGRATION") != "1" or not REDIS_URL,
    reason="set FUDIA_REDIS_INTEGRATION=1 and REDIS_URL for Redis integration",
)


def inbound(message_id: str, phone: str = "51911111111") -> InboundMessage:
    return InboundMessage(
        message_id=message_id,
        phone=phone,
        text="hola",
        sender_phone_id="meta-phone",
    )


@pytest.fixture
async def queues() -> AsyncIterator[tuple[RedisInboundQueue, RedisInboundQueue]]:
    stream = f"fudia:concierge:test:{uuid4().hex}"
    first = RedisInboundQueue(
        REDIS_URL, stream=stream, group="it", consumer="c1",
        max_attempts=2, visibility_timeout_ms=60_000,
    )
    second = RedisInboundQueue(
        REDIS_URL, stream=stream, group="it", consumer="c2",
        max_attempts=2, visibility_timeout_ms=0,
    )
    await first.start()
    yield first, second
    await first.client.delete(stream, f"{stream}:dead")
    await first.close()
    await second.close()


@pytest.mark.asyncio
async def test_fenced_commit_rejects_a_worker_whose_lock_moved() -> None:
    lock = RedisConversationLock(REDIS_URL, ttl_seconds=30, wait_seconds=1)
    store = RedisConversationStore(REDIS_URL, ttl_seconds=60)
    identity = uuid4().hex
    try:
        old = ConversationLease(lock, identity, await lock.acquire(identity))
        await lock.client.delete(lock_key(identity))  # lease expired
        new = ConversationLease(lock, identity, await lock.acquire(identity))
        current = ConversationSession(phone=identity, channel_key="meta-phone")
        await store.commit(new, current)

        with pytest.raises(LockLostError):
            await store.commit(
                old,
                ConversationSession(phone=identity, channel_key="meta-phone"),
                ProcessingRecord(message_id=f"old-{identity}"),
            )

        raw = await store.client.get(f"fudia:concierge:session:{identity}")
        assert raw is not None
        assert ConversationSession.model_validate_json(raw).conversation_id == (
            current.conversation_id
        )
        assert await store.get_record(f"old-{identity}") is None
    finally:
        await lock.client.delete(
            lock_key(identity), f"fudia:concierge:session:{identity}"
        )
        await lock.close()
        await store.close()


@pytest.mark.asyncio
async def test_touch_does_not_steal_an_entry_recovered_by_another_worker(
    queues: tuple[RedisInboundQueue, RedisInboundQueue],
) -> None:
    first, second = queues
    await first.enqueue(inbound(f"m-{uuid4().hex}", uuid4().hex))
    delivery = (await first.read())[0]
    assert await first.touch(delivery) is True

    recovered = await second.read()  # visibility 0: reclaims at once
    assert [item.entry_id for item in recovered] == [delivery.entry_id]

    assert await first.touch(delivery) is False
    owners = await first.client.xpending_range(
        first.stream, first.group, min="-", max="+", count=10
    )
    assert [owner["consumer"] for owner in owners] == ["c2"]


@pytest.mark.asyncio
async def test_order_waits_for_the_previous_message_and_survives_retries(
    queues: tuple[RedisInboundQueue, RedisInboundQueue],
) -> None:
    first, _ = queues
    phone = uuid4().hex
    await first.enqueue(inbound(f"m1-{phone}", phone))
    await first.enqueue(inbound(f"m2-{phone}", phone))
    one, two = await first.read()
    assert (one.sequence, two.sequence) == (1, 2)

    assert await first.wait_turn(two, 0.1) is False
    assert await first.retry(one) is True
    retried = (await first.read())[0]
    assert retried.sequence == 1
    assert await first.wait_turn(two, 0.1) is False

    await first.ack(retried)
    assert await first.wait_turn(two, 0.1) is True


@pytest.mark.asyncio
async def test_trim_keeps_pending_and_undelivered_entries(
    queues: tuple[RedisInboundQueue, RedisInboundQueue],
) -> None:
    first, _ = queues
    first.retention_seconds = 0
    for index in range(3):
        await first.enqueue(inbound(f"t{index}-{uuid4().hex}", uuid4().hex))
    acked, pending, _ = await first.read(limit=3)
    await first.ack(acked)
    undelivered = f"t3-{uuid4().hex}"
    await first.enqueue(inbound(undelivered, uuid4().hex))
    await asyncio.sleep(0.01)  # the retention cutoff has millisecond precision

    await first.trim()

    entries = await first.client.xrange(first.stream)
    ids = [entry_id for entry_id, _ in entries]
    assert acked.entry_id not in ids
    assert pending.entry_id in ids
    assert any(undelivered in values["payload"] for _, values in entries)


@pytest.mark.asyncio
async def test_dead_letter_requeue_keeps_the_message_id(
    queues: tuple[RedisInboundQueue, RedisInboundQueue],
) -> None:
    first, _ = queues
    first.max_attempts = 1
    message_id = f"dead-{uuid4().hex}"
    await first.enqueue(inbound(message_id, uuid4().hex))
    delivery = (await first.read())[0]
    assert await first.retry(delivery, "RuntimeError") is False

    letters = await first.dead_letters()
    assert [letter.message.message_id for letter in letters] == [message_id]
    assert letters[0].error == "RuntimeError"
    assert letters[0].sequence == 1

    assert await first.requeue_dead(letters[0].entry_id) is True
    assert await first.dead_letters() == []
    again = (await first.read())[0]
    assert again.message.message_id == message_id
    assert again.attempt == 0
    assert again.sequence == 1


@pytest.mark.asyncio
async def test_worker_on_redis_resends_the_persisted_reply_after_a_send_failure(
    queues: tuple[RedisInboundQueue, RedisInboundQueue],
) -> None:
    from src.services.concierge_service import ConciergeService
    from src.services.queue_worker import QueueWorker
    from tests.test_recovery import ADD_P1, TOKEN_A, Backend, FlakyWhatsApp, ScriptedAgent

    first, _ = queues
    store = RedisConversationStore(REDIS_URL, ttl_seconds=600)
    lock = RedisConversationLock(REDIS_URL, ttl_seconds=30, wait_seconds=1)
    backend = Backend()
    agent = ScriptedAgent([ADD_P1], reply="Agregué p1.")
    concierge = ConciergeService(
        store, backend, agent, conversation_lock=lock  # type: ignore[arg-type]
    )
    whatsapp = FlakyWhatsApp()
    worker = QueueWorker(first, concierge, whatsapp, records=store)
    phone = uuid4().hex
    try:
        for message_id, text in (("qr", f"FUDIA:{TOKEN_A}"), ("add", "un p1")):
            await first.enqueue(
                InboundMessage(
                    message_id=f"{message_id}-{phone}",
                    phone=phone,
                    text=text,
                    sender_phone_id="meta-phone",
                )
            )
        await worker.process((await first.read(limit=1))[0])  # qr
        whatsapp.failures = 1
        await worker.process((await first.read(limit=1))[0])  # send fails
        await worker.process((await first.read(limit=1))[0])  # resend only

        assert agent.calls == 1
        assert [sent[1] for sent in whatsapp.sent][-1] == "Agregué p1."
        session = await store.get(phone, "meta-phone")
        assert session is not None
        assert [line.quantity for line in session.cart] == [1]
        record = await store.get_record(f"add-{phone}")
        assert record is not None and record.stage == "sent"
        assert (await first.stats()).pending == 0
    finally:
        await store.close()
        await lock.close()
