from __future__ import annotations

import hashlib
import socket
from dataclasses import dataclass
from typing import Any, Protocol

import redis.asyncio as redis
from redis.exceptions import ResponseError

from src.domain.models import InboundMessage


@dataclass
class QueueDelivery:
    entry_id: str
    message: InboundMessage
    attempt: int


class InboundQueue(Protocol):
    async def start(self) -> None: ...
    async def enqueue(self, message: InboundMessage) -> bool: ...
    async def read(self) -> list[QueueDelivery]: ...
    async def ack(self, delivery: QueueDelivery) -> None: ...
    async def retry(self, delivery: QueueDelivery) -> bool: ...
    async def ping(self) -> bool: ...


class RedisInboundQueue:
    _ENQUEUE_SCRIPT = """
    local status = redis.call('GET', KEYS[1])
    if status and status ~= 'failed' then
      return 0
    end
    redis.call(
      'XADD', KEYS[2], '*',
      'payload', ARGV[1],
      'attempt', '0'
    )
    redis.call(
      'SET', KEYS[1], 'queued',
      'EX', ARGV[2]
    )
    return 1
    """

    def __init__(
        self,
        redis_url: str,
        stream: str = "fudia:concierge:inbound",
        group: str = "fudia-concierge",
        consumer: str = "",
        dedup_ttl_seconds: int = 86_400,
        max_attempts: int = 5,
        visibility_timeout_ms: int = 30_000,
    ) -> None:
        self.client = redis.from_url(
            redis_url,
            decode_responses=True,
        )
        self.stream = stream
        self.group = group
        self.consumer = (
            consumer.strip()
            or f"{socket.gethostname()}-{id(self)}"
        )
        self.dedup_ttl_seconds = dedup_ttl_seconds
        self.max_attempts = max_attempts
        self.visibility_timeout_ms = visibility_timeout_ms
        self._started = False

    def _dedup_key(self, message_id: str) -> str:
        digest = hashlib.sha256(
            message_id.encode("utf-8")
        ).hexdigest()
        return f"fudia:concierge:message:{digest}"

    async def start(self) -> None:
        if self._started:
            return
        try:
            await self.client.xgroup_create(
                self.stream,
                self.group,
                id="0-0",
                mkstream=True,
            )
        except ResponseError as exc:
            if "BUSYGROUP" not in str(exc):
                raise
        self._started = True

    async def enqueue(
        self,
        message: InboundMessage,
    ) -> bool:
        if not message.message_id:
            return False
        await self.start()
        payload = message.model_dump_json()
        result = await self.client.eval(
            self._ENQUEUE_SCRIPT,
            2,
            self._dedup_key(message.message_id),
            self.stream,
            payload,
            self.dedup_ttl_seconds,
        )
        return bool(result)

    def _delivery(
        self,
        entry_id: str,
        values: dict[str, str],
    ) -> QueueDelivery:
        payload = str(values.get("payload", "{}"))
        attempt = int(values.get("attempt", "0"))
        return QueueDelivery(
            entry_id=entry_id,
            message=InboundMessage.model_validate_json(
                payload
            ),
            attempt=attempt,
        )

    async def _claim_stale(
        self,
        count: int,
    ) -> list[QueueDelivery]:
        claimed: Any = await self.client.xautoclaim(
            self.stream,
            self.group,
            self.consumer,
            min_idle_time=self.visibility_timeout_ms,
            start_id="0-0",
            count=count,
        )
        if (
            not isinstance(claimed, (list, tuple))
            or len(claimed) < 2
        ):
            return []

        raw_entries: Any = claimed[1]
        if not isinstance(raw_entries, list):
            return []

        deliveries: list[QueueDelivery] = []
        for raw_entry in raw_entries:
            if (
                not isinstance(raw_entry, (list, tuple))
                or len(raw_entry) != 2
            ):
                continue
            entry_id = str(raw_entry[0])
            raw_values: Any = raw_entry[1]
            if not isinstance(raw_values, dict):
                continue
            values = {
                str(key): str(value)
                for key, value in raw_values.items()
            }
            deliveries.append(
                self._delivery(entry_id, values)
            )
        return deliveries

    async def read(
        self,
    ) -> list[QueueDelivery]:
        await self.start()
        stale = await self._claim_stale(10)
        if stale:
            return stale

        batches = await self.client.xreadgroup(
            self.group,
            self.consumer,
            {self.stream: ">"},
            count=10,
            block=1000,
        )
        deliveries: list[QueueDelivery] = []
        for _, entries in batches:
            deliveries.extend(
                self._delivery(entry_id, values)
                for entry_id, values in entries
            )
        return deliveries

    async def ack(
        self,
        delivery: QueueDelivery,
    ) -> None:
        pipeline = self.client.pipeline(
            transaction=True
        )
        pipeline.xack(
            self.stream,
            self.group,
            delivery.entry_id,
        )
        pipeline.set(
            self._dedup_key(
                delivery.message.message_id
            ),
            "completed",
            ex=self.dedup_ttl_seconds,
        )
        await pipeline.execute()

    async def retry(
        self,
        delivery: QueueDelivery,
    ) -> bool:
        next_attempt = delivery.attempt + 1
        key = self._dedup_key(
            delivery.message.message_id
        )
        pipeline = self.client.pipeline(
            transaction=True
        )
        if next_attempt >= self.max_attempts:
            pipeline.xadd(
                f"{self.stream}:dead",
                {
                    "payload": (
                        delivery.message.model_dump_json()
                    ),
                    "attempt": str(next_attempt),
                },
            )
            pipeline.xack(
                self.stream,
                self.group,
                delivery.entry_id,
            )
            pipeline.set(
                key,
                "failed",
                ex=300,
            )
            await pipeline.execute()
            return False

        pipeline.xadd(
            self.stream,
            {
                "payload": (
                    delivery.message.model_dump_json()
                ),
                "attempt": str(next_attempt),
            },
        )
        pipeline.xack(
            self.stream,
            self.group,
            delivery.entry_id,
        )
        pipeline.set(
            key,
            "queued",
            ex=self.dedup_ttl_seconds,
        )
        await pipeline.execute()
        return True

    async def ping(self) -> bool:
        return bool(await self.client.ping())

    async def close(self) -> None:
        await self.client.aclose()


class MemoryInboundQueue:
    def __init__(
        self,
        max_attempts: int = 5,
    ) -> None:
        self.max_attempts = max_attempts
        self.pending: list[QueueDelivery] = []
        self.status: dict[str, str] = {}
        self.sequence = 0

    async def start(self) -> None:
        return None

    async def enqueue(
        self,
        message: InboundMessage,
    ) -> bool:
        status = self.status.get(
            message.message_id
        )
        if status and status != "failed":
            return False
        self.sequence += 1
        self.pending.append(
            QueueDelivery(
                entry_id=str(self.sequence),
                message=message,
                attempt=0,
            )
        )
        self.status[message.message_id] = "queued"
        return True

    async def read(
        self,
    ) -> list[QueueDelivery]:
        if not self.pending:
            return []
        return [self.pending.pop(0)]

    async def ack(
        self,
        delivery: QueueDelivery,
    ) -> None:
        self.status[
            delivery.message.message_id
        ] = "completed"

    async def retry(
        self,
        delivery: QueueDelivery,
    ) -> bool:
        next_attempt = delivery.attempt + 1
        if next_attempt >= self.max_attempts:
            self.status[
                delivery.message.message_id
            ] = "failed"
            return False
        self.sequence += 1
        self.pending.append(
            QueueDelivery(
                entry_id=str(self.sequence),
                message=delivery.message,
                attempt=next_attempt,
            )
        )
        self.status[
            delivery.message.message_id
        ] = "queued"
        return True

    async def ping(self) -> bool:
        return True
