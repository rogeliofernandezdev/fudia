from __future__ import annotations

import asyncio
import hashlib
import logging
import socket
import time
from dataclasses import dataclass
from typing import Any

import redis.asyncio as redis
from redis.exceptions import ResponseError

from src.domain.models import InboundMessage
from src.metrics import QUEUE_ORDER_GAPS, QUEUE_TRIMMED
from src.observability import fingerprint, log_event
from src.services.ports import QueueDelivery as QueueDelivery
from src.services.ports import QueueStats, conversation_identity

logger = logging.getLogger(__name__)


def _conversation(message: InboundMessage) -> str:
    return conversation_identity(
        message.phone,
        message.sender_phone_id.strip() or "default",
    )


def _stream_id(value: str) -> tuple[int, int]:
    milliseconds, _, sequence = value.partition("-")
    return int(milliseconds or 0), int(sequence or 0)


def retention_floor(
    cutoff_ms: int,
    last_delivered_id: str,
    oldest_pending_id: str | None,
) -> str:
    """Lowest stream id that must survive trimming (`XTRIM MINID`).

    Entries older than the cutoff go away only if they were delivered to the
    group and acknowledged: anything after the last delivered id was never
    read, and anything from the oldest pending id on may still be retried.
    """
    candidates = [f"{cutoff_ms}-0", last_delivered_id]
    if oldest_pending_id:
        candidates.append(oldest_pending_id)
    return min(candidates, key=_stream_id)


@dataclass
class DeadLetter:
    entry_id: str
    message: InboundMessage
    attempt: int
    sequence: int
    failed_at_ms: int
    error: str


class RedisInboundQueue:
    # Dedup and per-conversation order are assigned atomically with the
    # XADD: the order is the order in which the webhook accepted messages.
    _ENQUEUE_SCRIPT = """
    local status = redis.call('GET', KEYS[1])
    if status and status ~= 'failed' then
      return 0
    end
    local seq = redis.call('INCR', KEYS[3])
    redis.call('EXPIRE', KEYS[3], ARGV[3])
    redis.call('SET', KEYS[4], '0', 'NX')
    redis.call('EXPIRE', KEYS[4], ARGV[3])
    redis.call(
      'XADD', KEYS[2], '*',
      'payload', ARGV[1],
      'attempt', '0',
      'seq', seq,
      'conversation', ARGV[4]
    )
    redis.call('SET', KEYS[1], 'queued', 'EX', ARGV[2])
    return seq
    """

    _ACK_SCRIPT = """
    redis.call('XACK', KEYS[1], ARGV[1], ARGV[2])
    redis.call('SET', KEYS[2], 'completed', 'EX', ARGV[3])
    local seq = tonumber(ARGV[4])
    if seq > 0 then
      local done = tonumber(redis.call('GET', KEYS[3]) or '0')
      if seq > done then
        redis.call('SET', KEYS[3], seq, 'EX', ARGV[5])
      end
    end
    return 1
    """

    # Retries keep message id, sequence and conversation, so a retried
    # message still blocks the next one of the same conversation. Dead
    # letters release the sequence to avoid blocking the conversation.
    _RETRY_SCRIPT = """
    if ARGV[7] == '1' then
      redis.call(
        'XADD', KEYS[2], '*',
        'payload', ARGV[3],
        'attempt', ARGV[4],
        'seq', ARGV[5],
        'conversation', ARGV[6],
        'failed_at', ARGV[10],
        'error', ARGV[11]
      )
      redis.call('XACK', KEYS[1], ARGV[1], ARGV[2])
      redis.call('SET', KEYS[3], 'failed', 'EX', 300)
      local seq = tonumber(ARGV[5])
      if seq > 0 then
        local done = tonumber(redis.call('GET', KEYS[4]) or '0')
        if seq > done then
          redis.call('SET', KEYS[4], seq, 'EX', ARGV[9])
        end
      end
      return 0
    end
    redis.call(
      'XADD', KEYS[1], '*',
      'payload', ARGV[3],
      'attempt', ARGV[4],
      'seq', ARGV[5],
      'conversation', ARGV[6]
    )
    redis.call('XACK', KEYS[1], ARGV[1], ARGV[2])
    redis.call('SET', KEYS[3], 'queued', 'EX', ARGV[8])
    return 1
    """

    # Renew the idle time only while this consumer still owns the entry.
    # An unconditional XCLAIM would steal it back from a worker that already
    # recovered it, and both would process the same delivery.
    _TOUCH_SCRIPT = """
    local owned = redis.call(
      'XPENDING', KEYS[1], ARGV[1], ARGV[3], ARGV[3], 1, ARGV[2]
    )
    if #owned == 0 then
      return 0
    end
    redis.call('XCLAIM', KEYS[1], ARGV[1], ARGV[2], 0, ARGV[3], 'JUSTID')
    return 1
    """

    _REQUEUE_SCRIPT = """
    local entries = redis.call('XRANGE', KEYS[1], ARGV[1], ARGV[1])
    if #entries == 0 then
      return 0
    end
    local fields = entries[1][2]
    local values = {}
    for index = 1, #fields, 2 do
      values[fields[index]] = fields[index + 1]
    end
    redis.call(
      'XADD', KEYS[2], '*',
      'payload', values['payload'],
      'attempt', '0',
      'seq', values['seq'] or '0',
      'conversation', values['conversation'] or '',
      'requeued', '1'
    )
    redis.call('SET', KEYS[3], 'queued', 'EX', ARGV[2])
    redis.call('XDEL', KEYS[1], ARGV[1])
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
        order_ttl_seconds: int = 604_800,
        order_max_wait_seconds: float = 120.0,
        retention_seconds: int = 86_400,
        dead_letter_retention_seconds: int = 1_209_600,
    ) -> None:
        self.client = redis.from_url(
            redis_url,
            decode_responses=True,
        )
        self.stream = stream
        self.dead_stream = f"{stream}:dead"
        self.group = group
        self.consumer = (
            consumer.strip()
            or f"{socket.gethostname()}-{id(self)}"
        )
        self.dedup_ttl_seconds = dedup_ttl_seconds
        self.max_attempts = max_attempts
        self.visibility_timeout_ms = visibility_timeout_ms
        self.order_ttl_seconds = order_ttl_seconds
        self.order_max_wait_seconds = order_max_wait_seconds
        self.retention_seconds = retention_seconds
        self.dead_letter_retention_seconds = dead_letter_retention_seconds
        self._started = False

    def _dedup_key(self, message_id: str) -> str:
        digest = hashlib.sha256(
            message_id.encode("utf-8")
        ).hexdigest()
        return f"fudia:concierge:message:{digest}"

    def _sequence_key(self, conversation: str) -> str:
        return f"fudia:concierge:seq:{conversation}"

    def _done_key(self, conversation: str) -> str:
        return f"fudia:concierge:seqdone:{conversation}"

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
        if not message.received_at:
            message = message.model_copy(
                update={"received_at": time.time()}
            )
        conversation = _conversation(message)
        result = await self.client.eval(
            self._ENQUEUE_SCRIPT,
            4,
            self._dedup_key(message.message_id),
            self.stream,
            self._sequence_key(conversation),
            self._done_key(conversation),
            message.model_dump_json(),
            self.dedup_ttl_seconds,
            self.order_ttl_seconds,
            conversation,
        )
        return bool(result)

    def _delivery(
        self,
        entry_id: str,
        values: dict[str, str],
    ) -> QueueDelivery:
        payload = str(values.get("payload", "{}"))
        sequence = int(values.get("seq", "0") or 0)
        message = InboundMessage.model_validate_json(payload)
        return QueueDelivery(
            entry_id=entry_id,
            message=message.model_copy(update={"sequence": sequence}),
            attempt=int(values.get("attempt", "0")),
            sequence=sequence,
            conversation=str(values.get("conversation", "")),
        )

    def _deliveries(self, raw_entries: Any) -> list[QueueDelivery]:
        if not isinstance(raw_entries, list):
            return []
        deliveries: list[QueueDelivery] = []
        for raw_entry in raw_entries:
            if (
                not isinstance(raw_entry, (list, tuple))
                or len(raw_entry) != 2
            ):
                continue
            raw_values: Any = raw_entry[1]
            if not isinstance(raw_values, dict):
                continue
            values = {
                str(key): str(value)
                for key, value in raw_values.items()
            }
            deliveries.append(
                self._delivery(str(raw_entry[0]), values)
            )
        return deliveries

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
        return self._deliveries(claimed[1])

    async def read(
        self,
        limit: int = 10,
    ) -> list[QueueDelivery]:
        if limit <= 0:
            return []
        await self.start()
        stale = await self._claim_stale(limit)
        if stale:
            return stale

        batches: Any = await self.client.xreadgroup(
            self.group,
            self.consumer,
            {self.stream: ">"},
            count=limit,
            block=1000,
        )
        if not isinstance(batches, list):
            return []

        deliveries: list[QueueDelivery] = []
        for raw_batch in batches:
            if (
                not isinstance(raw_batch, (list, tuple))
                or len(raw_batch) != 2
            ):
                continue
            deliveries.extend(self._deliveries(raw_batch[1]))
        return deliveries

    async def wait_turn(
        self,
        delivery: QueueDelivery,
        timeout: float,
    ) -> bool:
        """True when every earlier message of the conversation finished
        (acknowledged or dead-lettered)."""
        if delivery.sequence <= 0 or not delivery.conversation:
            return True
        loop = asyncio.get_running_loop()
        deadline = loop.time() + timeout
        key = self._done_key(delivery.conversation)
        while True:
            done = int(await self.client.get(key) or 0)
            if done >= delivery.sequence - 1:
                return True
            if _order_wait_exceeded(
                delivery, self.order_max_wait_seconds
            ):
                return True
            if loop.time() >= deadline:
                return False
            await asyncio.sleep(0.05)

    async def ack(
        self,
        delivery: QueueDelivery,
    ) -> None:
        await self.client.eval(
            self._ACK_SCRIPT,
            3,
            self.stream,
            self._dedup_key(delivery.message.message_id),
            self._done_key(delivery.conversation or "none"),
            self.group,
            delivery.entry_id,
            self.dedup_ttl_seconds,
            delivery.sequence if delivery.conversation else 0,
            self.order_ttl_seconds,
        )

    async def defer(
        self,
        delivery: QueueDelivery,
    ) -> None:
        # Leave the entry pending without a new copy: once its idle time
        # passes the visibility timeout, XAUTOCLAIM delivers it again with
        # the same entry id, attempt and sequence.
        return None

    async def touch(
        self,
        delivery: QueueDelivery,
    ) -> bool:
        owned = await self.client.eval(
            self._TOUCH_SCRIPT,
            1,
            self.stream,
            self.group,
            self.consumer,
            delivery.entry_id,
        )
        return bool(owned)

    async def retry(
        self,
        delivery: QueueDelivery,
        error: str = "",
    ) -> bool:
        next_attempt = delivery.attempt + 1
        dead = next_attempt >= self.max_attempts
        result = await self.client.eval(
            self._RETRY_SCRIPT,
            4,
            self.stream,
            self.dead_stream,
            self._dedup_key(delivery.message.message_id),
            self._done_key(delivery.conversation or "none"),
            self.group,
            delivery.entry_id,
            delivery.message.model_dump_json(),
            next_attempt,
            delivery.sequence if delivery.conversation else 0,
            delivery.conversation,
            "1" if dead else "0",
            self.dedup_ttl_seconds,
            self.order_ttl_seconds,
            int(time.time() * 1000),
            error[:80],
        )
        return bool(result)

    async def trim(self) -> int:
        """Apply retention without losing work: entries are kept while they
        are pending, undelivered or younger than the retention window."""
        await self.start()
        now_ms = int(time.time() * 1000)
        last_delivered = "0-0"
        groups: Any = await self.client.xinfo_groups(self.stream)
        for group in groups if isinstance(groups, list) else []:
            if str(group.get("name")) == self.group:
                last_delivered = str(group.get("last-delivered-id", "0-0"))
        pending: Any = await self.client.xpending(
            self.stream, self.group
        )
        oldest_pending = (
            str(pending.get("min"))
            if isinstance(pending, dict) and int(pending.get("pending", 0))
            else None
        )
        min_id = retention_floor(
            now_ms - self.retention_seconds * 1000,
            last_delivered,
            oldest_pending,
        )
        trimmed = int(
            await self.client.xtrim(
                self.stream, minid=min_id, approximate=False
            )
            or 0
        )
        dead_trimmed = int(
            await self.client.xtrim(
                self.dead_stream,
                minid=(
                    f"{now_ms - self.dead_letter_retention_seconds * 1000}-0"
                ),
                approximate=False,
            )
            or 0
        )
        if trimmed:
            QUEUE_TRIMMED.labels(stream="inbound").inc(trimmed)
        if dead_trimmed:
            QUEUE_TRIMMED.labels(stream="dead_letter").inc(dead_trimmed)
        return trimmed + dead_trimmed

    async def stats(self) -> QueueStats:
        await self.start()
        pending: Any = await self.client.xpending(
            self.stream, self.group
        )
        return QueueStats(
            pending=(
                int(pending.get("pending", 0))
                if isinstance(pending, dict)
                else 0
            ),
            dead_letter=int(await self.client.xlen(self.dead_stream)),
            stream_length=int(await self.client.xlen(self.stream)),
        )

    async def dead_letters(
        self,
        count: int = 50,
    ) -> list[DeadLetter]:
        entries: Any = await self.client.xrange(
            self.dead_stream, count=count
        )
        letters: list[DeadLetter] = []
        for entry_id, values in entries or []:
            letters.append(
                DeadLetter(
                    entry_id=str(entry_id),
                    message=InboundMessage.model_validate_json(
                        str(values.get("payload", "{}"))
                    ),
                    attempt=int(values.get("attempt", "0")),
                    sequence=int(values.get("seq", "0") or 0),
                    failed_at_ms=int(values.get("failed_at", "0") or 0),
                    error=str(values.get("error", "")),
                )
            )
        return letters

    async def requeue_dead(self, entry_id: str) -> bool:
        """Move one dead letter back to the stream with its original message
        id. Persisted processing records keep the replay from repeating
        effects that already happened."""
        entries: Any = await self.client.xrange(
            self.dead_stream, min=entry_id, max=entry_id
        )
        if not entries:
            return False
        message = InboundMessage.model_validate_json(
            str(entries[0][1].get("payload", "{}"))
        )
        moved = await self.client.eval(
            self._REQUEUE_SCRIPT,
            3,
            self.dead_stream,
            self.stream,
            self._dedup_key(message.message_id),
            entry_id,
            self.dedup_ttl_seconds,
        )
        return bool(moved)

    async def discard_dead(self, entry_id: str) -> bool:
        return bool(await self.client.xdel(self.dead_stream, entry_id))

    async def ping(self) -> bool:
        return bool(await self.client.ping())

    async def close(self) -> None:
        await self.client.aclose()


def _order_wait_exceeded(
    delivery: QueueDelivery,
    limit_seconds: float,
) -> bool:
    received_at = delivery.message.received_at
    if not received_at or time.time() - received_at <= limit_seconds:
        return False
    # The predecessor is lost or stuck: process anyway so one message
    # cannot block the conversation forever.
    QUEUE_ORDER_GAPS.inc()
    log_event(
        logger,
        "concierge.queue.order_gap",
        message=fingerprint(delivery.message.message_id),
        sequence=delivery.sequence,
    )
    return True


class MemoryInboundQueue:
    def __init__(
        self,
        max_attempts: int = 5,
        order_max_wait_seconds: float = 120.0,
        defer_delay_seconds: float = 0.05,
    ) -> None:
        self.max_attempts = max_attempts
        self.order_max_wait_seconds = order_max_wait_seconds
        self.defer_delay_seconds = defer_delay_seconds
        self.pending: list[QueueDelivery] = []
        self.dead: list[QueueDelivery] = []
        self.status: dict[str, str] = {}
        self.sequences: dict[str, int] = {}
        self.done: dict[str, int] = {}
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
        if not message.received_at:
            message = message.model_copy(
                update={"received_at": time.time()}
            )
        conversation = _conversation(message)
        self.sequences[conversation] = (
            self.sequences.get(conversation, 0) + 1
        )
        self.sequence += 1
        self.pending.append(
            QueueDelivery(
                entry_id=str(self.sequence),
                message=message.model_copy(
                    update={"sequence": self.sequences[conversation]}
                ),
                attempt=0,
                sequence=self.sequences[conversation],
                conversation=conversation,
            )
        )
        self.status[message.message_id] = "queued"
        return True

    async def read(
        self,
        limit: int = 10,
    ) -> list[QueueDelivery]:
        if limit <= 0 or not self.pending:
            return []
        deliveries = self.pending[:limit]
        del self.pending[:limit]
        return deliveries

    async def wait_turn(
        self,
        delivery: QueueDelivery,
        timeout: float,
    ) -> bool:
        if delivery.sequence <= 0 or not delivery.conversation:
            return True
        loop = asyncio.get_running_loop()
        deadline = loop.time() + timeout
        while True:
            done = self.done.get(delivery.conversation, 0)
            if done >= delivery.sequence - 1:
                return True
            if _order_wait_exceeded(
                delivery, self.order_max_wait_seconds
            ):
                return True
            if loop.time() >= deadline:
                return False
            await asyncio.sleep(0.005)

    def _release_turn(self, delivery: QueueDelivery) -> None:
        if delivery.conversation and delivery.sequence > self.done.get(
            delivery.conversation, 0
        ):
            self.done[delivery.conversation] = delivery.sequence

    async def ack(
        self,
        delivery: QueueDelivery,
    ) -> None:
        self.status[
            delivery.message.message_id
        ] = "completed"
        self._release_turn(delivery)

    async def defer(
        self,
        delivery: QueueDelivery,
    ) -> None:
        asyncio.get_running_loop().call_later(
            self.defer_delay_seconds,
            self.pending.append,
            delivery,
        )

    async def touch(
        self,
        delivery: QueueDelivery,
    ) -> bool:
        return (
            self.status.get(
                delivery.message.message_id
            )
            == "queued"
        )

    async def retry(
        self,
        delivery: QueueDelivery,
        error: str = "",
    ) -> bool:
        next_attempt = delivery.attempt + 1
        if next_attempt >= self.max_attempts:
            self.status[
                delivery.message.message_id
            ] = "failed"
            self.dead.append(delivery)
            self._release_turn(delivery)
            return False
        self.sequence += 1
        self.pending.append(
            QueueDelivery(
                entry_id=str(self.sequence),
                message=delivery.message,
                attempt=next_attempt,
                sequence=delivery.sequence,
                conversation=delivery.conversation,
            )
        )
        self.status[
            delivery.message.message_id
        ] = "queued"
        return True

    async def trim(self) -> int:
        return 0

    async def stats(self) -> QueueStats:
        return QueueStats(
            pending=len(self.pending),
            dead_letter=len(self.dead),
            stream_length=len(self.pending),
        )

    async def ping(self) -> bool:
        return True
