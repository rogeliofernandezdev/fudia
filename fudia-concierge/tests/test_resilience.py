import asyncio

import pytest

from src.domain.models import ConversationSession, InboundMessage
from src.infrastructure.concurrency import (
    MemoryConversationLock,
    conversation_identity,
)
from src.infrastructure.queue import MemoryInboundQueue, QueueDelivery
from src.infrastructure.state_store import MemoryConversationStore
from src.services.queue_worker import QueueWorker


@pytest.mark.asyncio
async def test_session_store_isolates_same_customer_by_channel() -> None:
    store = MemoryConversationStore()
    first = ConversationSession(
        phone="51999999999",
        channel_key="meta-a",
    )
    second = ConversationSession(
        phone="51999999999",
        channel_key="meta-b",
    )
    await store.save(first)
    await store.save(second)

    loaded_a = await store.get(
        "51999999999",
        "meta-a",
    )
    loaded_b = await store.get(
        "51999999999",
        "meta-b",
    )

    assert loaded_a is not None
    assert loaded_b is not None
    assert (
        loaded_a.conversation_id
        != loaded_b.conversation_id
    )


@pytest.mark.asyncio
async def test_memory_lock_serializes_same_conversation() -> None:
    lock = MemoryConversationLock()
    token = await lock.acquire("same")
    blocked = False

    async def contender() -> None:
        nonlocal blocked
        blocked = True
        next_token = await lock.acquire("same")
        blocked = False
        await lock.release("same", next_token)

    task = asyncio.create_task(contender())
    await asyncio.sleep(0)
    assert blocked is True
    await lock.release("same", token)
    await task
    assert blocked is False


@pytest.mark.asyncio
async def test_queue_deduplicates_retries_and_allows_failed_redelivery() -> None:
    queue = MemoryInboundQueue(max_attempts=2)
    message = InboundMessage(
        message_id="wamid.1",
        phone="51999999999",
        text="hola",
    )

    assert await queue.enqueue(message) is True
    assert await queue.enqueue(message) is False

    delivery = (await queue.read())[0]
    assert await queue.retry(delivery) is True
    retry_delivery = (await queue.read())[0]
    assert retry_delivery.attempt == 1

    assert await queue.retry(retry_delivery) is False
    assert queue.status["wamid.1"] == "failed"
    assert await queue.enqueue(message) is True


class FakeService:
    def __init__(self) -> None:
        self.calls = 0

    async def handle_message(
        self,
        phone: str,
        text: str,
        channel_id: str = "",
    ) -> str:
        self.calls += 1
        return "respuesta"


class FakeWhatsApp:
    def __init__(self) -> None:
        self.sent: list[tuple[str, str]] = []

    async def send_text(
        self,
        phone: str,
        message: str,
    ) -> None:
        self.sent.append(
            (phone, message)
        )


class BlockingService:
    def __init__(self) -> None:
        self.active = 0
        self.peak = 0
        self.two_active = asyncio.Event()
        self.release = asyncio.Event()

    async def handle_message(
        self,
        phone: str,
        text: str,
        channel_id: str = "",
    ) -> str:
        self.active += 1
        self.peak = max(self.peak, self.active)
        if self.active >= 2:
            self.two_active.set()
        try:
            await self.release.wait()
            return "respuesta"
        finally:
            self.active -= 1


class TouchTrackingQueue(MemoryInboundQueue):
    def __init__(self) -> None:
        super().__init__()
        self.touches = 0

    async def touch(
        self,
        delivery: QueueDelivery,
    ) -> bool:
        self.touches += 1
        return True


class LockingService:
    def __init__(self) -> None:
        self.lock = MemoryConversationLock()
        self.active_total = 0
        self.peak_total = 0
        self.active_by_identity: dict[str, int] = {}
        self.peak_by_identity: dict[str, int] = {}
        self.two_active = asyncio.Event()
        self.release = asyncio.Event()

    async def handle_message(
        self,
        phone: str,
        text: str,
        channel_id: str = "",
    ) -> str:
        identity = conversation_identity(
            phone,
            channel_id or "default",
        )
        token = await self.lock.acquire(identity)
        self.active_total += 1
        current = (
            self.active_by_identity.get(identity, 0) + 1
        )
        self.active_by_identity[identity] = current
        self.peak_total = max(
            self.peak_total,
            self.active_total,
        )
        self.peak_by_identity[identity] = max(
            self.peak_by_identity.get(identity, 0),
            current,
        )
        if self.active_total >= 2:
            self.two_active.set()
        try:
            await self.release.wait()
            return "respuesta"
        finally:
            self.active_total -= 1
            self.active_by_identity[identity] -= 1
            await self.lock.release(identity, token)


async def wait_for_queue_status(
    queue: MemoryInboundQueue,
    message_ids: set[str],
    status: str,
) -> None:
    async def ready() -> None:
        while not all(
            queue.status.get(message_id) == status
            for message_id in message_ids
        ):
            await asyncio.sleep(0.01)

    await asyncio.wait_for(
        ready(),
        timeout=2,
    )


@pytest.mark.asyncio
async def test_queue_worker_bounds_parallelism() -> None:
    queue = MemoryInboundQueue()
    service = BlockingService()
    whatsapp = FakeWhatsApp()
    worker = QueueWorker(
        queue,
        service,
        whatsapp,
        max_concurrency=2,
    )
    message_ids = {
        "wamid.parallel.1",
        "wamid.parallel.2",
        "wamid.parallel.3",
    }
    for index, message_id in enumerate(
        sorted(message_ids),
        start=1,
    ):
        await queue.enqueue(
            InboundMessage(
                message_id=message_id,
                phone=f"5199999999{index}",
                text="hola",
                sender_phone_id="meta-phone",
            )
        )

    task = asyncio.create_task(worker.run())
    await asyncio.wait_for(
        service.two_active.wait(),
        timeout=1,
    )

    assert service.peak == 2
    assert service.active == 2

    service.release.set()
    await wait_for_queue_status(
        queue,
        message_ids,
        "completed",
    )
    await worker.stop()
    await asyncio.wait_for(task, timeout=2)

    assert service.peak == 2
    assert len(whatsapp.sent) == 3


@pytest.mark.asyncio
async def test_worker_parallelizes_distinct_conversations_but_serializes_same_one() -> None:
    queue = MemoryInboundQueue()
    service = LockingService()
    whatsapp = FakeWhatsApp()
    worker = QueueWorker(
        queue,
        service,
        whatsapp,
        max_concurrency=3,
    )
    messages = [
        InboundMessage(
            message_id="wamid.same.1",
            phone="51911111111",
            text="uno",
            sender_phone_id="meta-a",
        ),
        InboundMessage(
            message_id="wamid.same.2",
            phone="51911111111",
            text="dos",
            sender_phone_id="meta-a",
        ),
        InboundMessage(
            message_id="wamid.other",
            phone="51922222222",
            text="otro",
            sender_phone_id="meta-a",
        ),
    ]
    for message in messages:
        await queue.enqueue(message)

    task = asyncio.create_task(worker.run())
    await asyncio.wait_for(
        service.two_active.wait(),
        timeout=1,
    )

    same_identity = conversation_identity(
        "51911111111",
        "meta-a",
    )
    assert service.peak_total == 2
    assert service.peak_by_identity[same_identity] == 1

    service.release.set()
    await wait_for_queue_status(
        queue,
        {message.message_id for message in messages},
        "completed",
    )
    await worker.stop()
    await asyncio.wait_for(task, timeout=2)

    assert len(whatsapp.sent) == 3


@pytest.mark.asyncio
async def test_queue_worker_renews_visibility_while_processing() -> None:
    queue = TouchTrackingQueue()
    service = BlockingService()
    whatsapp = FakeWhatsApp()
    worker = QueueWorker(
        queue,
        service,
        whatsapp,
        max_concurrency=1,
        visibility_heartbeat_seconds=0.01,
    )
    message_id = "wamid.visibility"
    await queue.enqueue(
        InboundMessage(
            message_id=message_id,
            phone="51933333333",
            text="hola",
        )
    )

    task = asyncio.create_task(worker.run())
    while queue.touches == 0:
        await asyncio.sleep(0.01)

    assert service.active == 1
    assert queue.touches >= 1

    service.release.set()
    await wait_for_queue_status(
        queue,
        {message_id},
        "completed",
    )
    await worker.stop()
    await asyncio.wait_for(task, timeout=2)


@pytest.mark.asyncio
async def test_queue_worker_stop_drains_in_flight_delivery() -> None:
    queue = MemoryInboundQueue()
    service = BlockingService()
    whatsapp = FakeWhatsApp()
    worker = QueueWorker(
        queue,
        service,
        whatsapp,
        max_concurrency=1,
    )
    message_id = "wamid.shutdown"
    await queue.enqueue(
        InboundMessage(
            message_id=message_id,
            phone="51911111111",
            text="hola",
        )
    )

    task = asyncio.create_task(worker.run())
    while service.active == 0:
        await asyncio.sleep(0)

    await worker.stop()
    assert task.done() is False

    service.release.set()
    await asyncio.wait_for(task, timeout=2)

    assert queue.status[message_id] == "completed"
    assert len(whatsapp.sent) == 1


@pytest.mark.asyncio
async def test_hard_worker_cancellation_leaves_delivery_unacked() -> None:
    queue = MemoryInboundQueue()
    service = BlockingService()
    whatsapp = FakeWhatsApp()
    worker = QueueWorker(
        queue,
        service,
        whatsapp,
        max_concurrency=1,
    )
    message_id = "wamid.cancelled"
    await queue.enqueue(
        InboundMessage(
            message_id=message_id,
            phone="51944444444",
            text="hola",
        )
    )

    task = asyncio.create_task(worker.run())
    while service.active == 0:
        await asyncio.sleep(0)

    task.cancel()
    with pytest.raises(asyncio.CancelledError):
        await task

    assert queue.status[message_id] == "queued"
    assert whatsapp.sent == []
    assert service.active == 0


@pytest.mark.asyncio
async def test_queue_worker_acks_only_after_reply_is_sent() -> None:
    queue = MemoryInboundQueue()
    service = FakeService()
    whatsapp = FakeWhatsApp()
    worker = QueueWorker(
        queue,
        service,
        whatsapp,
    )
    message = InboundMessage(
        message_id="wamid.2",
        phone="51999999999",
        text="quiero una bebida",
        sender_phone_id="meta-phone",
    )
    await queue.enqueue(message)
    delivery = (await queue.read())[0]

    await worker.process(delivery)

    assert service.calls == 1
    assert whatsapp.sent == [
        (
            "51999999999",
            "respuesta",
        )
    ]
    assert queue.status["wamid.2"] == "completed"
