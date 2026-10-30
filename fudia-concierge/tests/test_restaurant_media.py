from __future__ import annotations

import httpx
import pytest

from src.domain.models import InboundMessage, ProcessingRecord, ReplyImage
from src.infrastructure.state_store import MemoryConversationStore
from src.infrastructure.whatsapp_adapter import MetaWhatsAppAdapter
from src.services.outbox import DeliveryOutcome, ReplyDispatcher


class Sender:
    def __init__(self) -> None:
        self.sent: list[str] = []
        self.failed = False

    async def send_text(self, phone: str, message: str, callback_ref: str = "") -> None:
        self.sent.append("text")

    async def send_image(self, phone: str, url: str, caption: str = "", callback_ref: str = "") -> None:
        if url.endswith("2.jpg") and not self.failed:
            self.failed = True
            raise RuntimeError("retry second image")
        self.sent.append(url)


async def test_media_retry_resumes_at_failed_page_without_resending_text_or_logo() -> None:
    store, sender = MemoryConversationStore(), Sender()
    record = ProcessingRecord(
        message_id="media",
        stage="reply_ready",
        reply="Carta",
        media=[
            ReplyImage(url="https://storage.example/logo.png"),
            ReplyImage(url="https://storage.example/1.jpg"),
            ReplyImage(url="https://storage.example/2.jpg"),
        ],
    )
    message = InboundMessage(message_id="media", phone="51911111111", text="carta")
    dispatcher = ReplyDispatcher(store, sender)
    with pytest.raises(RuntimeError):
        await dispatcher.deliver(message, record)
    assert record.part_index == 3
    assert await dispatcher.deliver(message, record) == DeliveryOutcome.SENT
    assert sender.sent == ["text", *[image.url for image in record.media]]
    assert await dispatcher.deliver(message, record) == DeliveryOutcome.SENT
    assert len(sender.sent) == 4


async def test_meta_image_uses_signed_link_and_callback_without_exposing_token() -> None:
    import json

    captured: list[dict[str, object]] = []

    async def handle(request: httpx.Request) -> httpx.Response:
        captured.append(json.loads(request.content))
        return httpx.Response(200, json={"messages": [{"id": "wamid"}]})

    async with httpx.AsyncClient(transport=httpx.MockTransport(handle)) as client:
        adapter = MetaWhatsAppAdapter("private-token", "phone", "v26.0", client)
        await adapter.send_image(
            "51911111111", "https://storage.example/menu.jpg?signature=test", "Carta", "media-ref"
        )
    assert captured[0]["type"] == "image"
    assert captured[0]["biz_opaque_callback_data"] == "media-ref"
    assert "private-token" not in str(captured)
