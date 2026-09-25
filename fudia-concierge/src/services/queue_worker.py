from __future__ import annotations

import asyncio
import logging
import time
from typing import Protocol

from src.infrastructure.queue import InboundQueue, QueueDelivery
from src.metrics import QUEUE_DELIVERIES, QUEUE_DURATION, QUEUE_IN_FLIGHT
from src.observability import (
    begin_trace,
    end_trace,
    fingerprint,
    log_event,
)

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
    ) -> None: ...


class QueueWorker:
    def __init__(
        self,
        queue: InboundQueue,
        service: MessageService,
        whatsapp: WhatsAppSender,
        max_concurrency: int = 8,
        visibility_heartbeat_seconds: float = 10.0,
    ) -> None:
        if max_concurrency < 1:
            raise ValueError(
                "max_concurrency must be at least 1"
            )
        if visibility_heartbeat_seconds <= 0:
            raise ValueError(
                "visibility_heartbeat_seconds must be positive"
            )
        self.queue = queue
        self.service = service
        self.whatsapp = whatsapp
        self.max_concurrency = max_concurrency
        self.visibility_heartbeat_seconds = (
            visibility_heartbeat_seconds
        )
        self._stop = asyncio.Event()
        self._in_flight: set[
            asyncio.Future[None]
        ] = set()

    async def run(self) -> None:
        await self.queue.start()
        try:
            while not self._stop.is_set():
                capacity = (
                    self.max_concurrency
                    - len(self._in_flight)
                )
                if capacity <= 0:
                    await self._wait_for_capacity()
                    continue

                deliveries = await self.queue.read(
                    limit=capacity
                )
                if not deliveries:
                    await asyncio.sleep(0)
                    continue

                for delivery in deliveries:
                    self._start_delivery(delivery)
        except asyncio.CancelledError:
            await self._cancel_in_flight()
            raise
        finally:
            await self._drain()

    def _start_delivery(
        self,
        delivery: QueueDelivery,
    ) -> None:
        task = asyncio.create_task(
            self.process(delivery)
        )
        self._in_flight.add(task)
        task.add_done_callback(
            self._in_flight.discard
        )

    async def _wait_for_capacity(self) -> None:
        if not self._in_flight:
            return
        await asyncio.wait(
            tuple(self._in_flight),
            return_when=asyncio.FIRST_COMPLETED,
        )

    async def _drain(self) -> None:
        if not self._in_flight:
            return
        await asyncio.gather(
            *tuple(self._in_flight),
            return_exceptions=True,
        )

    async def _cancel_in_flight(self) -> None:
        tasks = tuple(self._in_flight)
        for task in tasks:
            task.cancel()
        if tasks:
            await asyncio.gather(
                *tasks,
                return_exceptions=True,
            )

    async def process(
        self,
        delivery: QueueDelivery,
    ) -> None:
        message = delivery.message
        started = time.perf_counter()
        trace_token = begin_trace(
            message.message_id
        )
        visibility_heartbeat = asyncio.create_task(
            self._keep_delivery_visible(delivery)
        )
        QUEUE_IN_FLIGHT.inc()
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
            )
            await self.queue.ack(delivery)
            duration = time.perf_counter() - started
            QUEUE_DELIVERIES.labels(
                result="completed"
            ).inc()
            QUEUE_DURATION.observe(duration)
            log_event(
                logger,
                "concierge.queue.completed",
                message=fingerprint(
                    message.message_id
                ),
                phone=fingerprint(message.phone),
                attempt=delivery.attempt,
                durationMs=round(
                    duration * 1000,
                    1,
                ),
            )
        except asyncio.CancelledError:
            raise
        except Exception as exc:
            will_retry = await self.queue.retry(
                delivery
            )
            QUEUE_DELIVERIES.labels(
                result=(
                    "retry"
                    if will_retry
                    else "dead_letter"
                )
            ).inc()
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
        finally:
            visibility_heartbeat.cancel()
            try:
                await visibility_heartbeat
            except asyncio.CancelledError:
                pass
            QUEUE_IN_FLIGHT.dec()
            end_trace(trace_token)

    async def _keep_delivery_visible(
        self,
        delivery: QueueDelivery,
    ) -> None:
        try:
            while True:
                await asyncio.sleep(
                    self.visibility_heartbeat_seconds
                )
                if not await self.queue.touch(delivery):
                    return
        except asyncio.CancelledError:
            raise
        except Exception as exc:
            log_event(
                logger,
                "concierge.queue.visibility_heartbeat_failed",
                message=fingerprint(
                    delivery.message.message_id
                ),
                errorType=type(exc).__name__,
            )
            logger.exception(
                "No se pudo renovar la visibilidad del mensaje %s",
                fingerprint(
                    delivery.message.message_id
                ),
            )

    async def stop(self) -> None:
        self._stop.set()
