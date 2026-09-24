from __future__ import annotations

import asyncio
import logging
import time
from typing import Protocol

from src.infrastructure.queue import InboundQueue, QueueDelivery
from src.observability import fingerprint, log_event

logger = logging.getLogger(__name__)


class MessageService(Protocol):
    async def handle_message(
        self,
        phone: str,
        text: str,
        recipient_phone: str = "",
        channel_id: str = "",
    ) -> str: ...


class WhatsAppSender(Protocol):
    async def send_text(
        self,
        phone: str,
        message: str,
        sender_phone_id: str = "",
    ) -> None: ...


class QueueWorker:
    def __init__(
        self,
        queue: InboundQueue,
        service: MessageService,
        whatsapp: WhatsAppSender,
    ) -> None:
        self.queue = queue
        self.service = service
        self.whatsapp = whatsapp
        self._stop = asyncio.Event()

    async def run(self) -> None:
        await self.queue.start()
        while not self._stop.is_set():
            deliveries = await self.queue.read()
            for delivery in deliveries:
                await self.process(delivery)

    async def process(
        self,
        delivery: QueueDelivery,
    ) -> None:
        message = delivery.message
        started = time.perf_counter()
        try:
            reply = await self.service.handle_message(
                message.phone,
                message.text,
                message.recipient_phone,
                message.sender_phone_id,
            )
            await self.whatsapp.send_text(
                message.phone,
                reply,
                message.sender_phone_id,
            )
            await self.queue.ack(delivery)
            log_event(
                logger,
                "concierge.queue.completed",
                message=fingerprint(
                    message.message_id
                ),
                phone=fingerprint(message.phone),
                attempt=delivery.attempt,
                durationMs=round(
                    (time.perf_counter() - started)
                    * 1000,
                    1,
                ),
            )
        except asyncio.CancelledError:
            raise
        except Exception as exc:
            will_retry = await self.queue.retry(
                delivery
            )
            log_event(
                logger,
                "concierge.queue.failed",
                message=fingerprint(
                    message.message_id
                ),
                phone=fingerprint(message.phone),
                attempt=delivery.attempt,
                retry=will_retry,
                errorType=type(exc).__name__,
            )
            logger.exception(
                "No se pudo procesar mensaje %s",
                fingerprint(message.message_id),
            )

    async def stop(self) -> None:
        self._stop.set()
