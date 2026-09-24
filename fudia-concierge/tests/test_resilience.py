import asyncio

import pytest

from src.domain.models import ConversationSession, InboundMessage
from src.infrastructure.concurrency import MemoryConversationLock
from src.infrastructure.queue import MemoryInboundQueue
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
        recipient_phone: str = "",
        channel_id: str = "",
    ) -> str:
        self.calls += 1
        return "respuesta"


class FakeWhatsApp:
    def __init__(self) -> None:
        self.sent: list[tuple[str, str, str]] = []

    async def send_text(
        self,
        phone: str,
        message: str,
        sender_phone_id: str = "",
    ) -> None:
        self.sent.append(
            (phone, message, sender_phone_id)
        )


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
        recipient_phone="+51 999",
    )
    await queue.enqueue(message)
    delivery = (await queue.read())[0]

    await worker.process(delivery)

    assert service.calls == 1
    assert whatsapp.sent == [
        (
            "51999999999",
            "respuesta",
            "meta-phone",
        )
    ]
    assert queue.status["wamid.2"] == "completed"
