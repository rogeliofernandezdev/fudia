from __future__ import annotations

import time

import redis.asyncio as redis

from src.domain.models import (
    ConversationSession,
    CustomerMemory,
    ProcessingRecord,
)
from src.infrastructure.concurrency import lock_key
from src.services.ports import ConversationLease, LockLostError, message_ref
from src.services.ports import ConversationStore as ConversationStore
from src.services.ports import conversation_identity as _session_identity

OUTBOUND_STATUSES = frozenset({"sent", "delivered", "read", "failed"})


def session_key(identity: str) -> str:
    return f"fudia:concierge:session:{identity}"


def record_key(message_id: str) -> str:
    return f"fudia:concierge:processing:{message_ref(message_id)}"


def outbound_key(ref: str) -> str:
    return f"fudia:concierge:outbound:{ref}"


def memory_key(identity: str) -> str:
    return f"fudia:concierge:customer:{identity}"


def _stamp(record: ProcessingRecord) -> ProcessingRecord:
    now = time.time()
    if not record.created_at:
        record.created_at = now
    record.updated_at = now
    return record


class RedisConversationStore:
    # Fenced write: session and record change together, and only while the
    # caller still owns the conversation lock. A worker whose lease expired
    # can no longer overwrite what the new owner persisted.
    _COMMIT_SCRIPT = """
    if redis.call('GET', KEYS[1]) ~= ARGV[1] then
      return 0
    end
    if ARGV[2] ~= '' then
      redis.call('SET', KEYS[2], ARGV[2], 'EX', ARGV[3])
    end
    if ARGV[4] ~= '' then
      redis.call('SET', KEYS[3], ARGV[4], 'EX', ARGV[5])
    end
    if ARGV[6] ~= '' then
      redis.call('SET', KEYS[4], ARGV[6], 'EX', ARGV[7])
    end
    return 1
    """

    def __init__(
        self,
        redis_url: str,
        ttl_seconds: int,
        record_ttl_seconds: int = 259_200,
        pending_record_ttl_seconds: int = 1_209_600,
        outbound_status_ttl_seconds: int = 86_400,
        memory_ttl_seconds: int = 7_776_000,
    ) -> None:
        self.client = redis.from_url(
            redis_url,
            decode_responses=True,
        )
        self.ttl_seconds = ttl_seconds
        self.record_ttl_seconds = record_ttl_seconds
        self.pending_record_ttl_seconds = pending_record_ttl_seconds
        self.outbound_status_ttl_seconds = outbound_status_ttl_seconds
        self.memory_ttl_seconds = memory_ttl_seconds

    def _key(
        self,
        phone: str,
        channel_key: str,
    ) -> str:
        return session_key(
            _session_identity(phone, channel_key)
        )

    def _record_ttl(self, record: ProcessingRecord) -> int:
        # Records that may still be needed for recovery outlive the stream
        # retries and the dead letter; finished ones expire sooner.
        if record.terminal:
            return self.record_ttl_seconds
        return self.pending_record_ttl_seconds

    async def get(
        self,
        phone: str,
        channel_key: str = "default",
    ) -> ConversationSession | None:
        raw = await self.client.get(
            self._key(phone, channel_key)
        )
        if not raw:
            return None
        return ConversationSession.model_validate_json(raw)

    async def get_memory(
        self,
        phone: str,
        channel_key: str = "default",
    ) -> CustomerMemory | None:
        raw = await self.client.get(
            memory_key(_session_identity(phone, channel_key))
        )
        if not raw:
            return None
        return CustomerMemory.model_validate_json(raw)

    async def save(
        self,
        session: ConversationSession,
    ) -> None:
        await self.client.set(
            self._key(
                session.phone,
                session.channel_key,
            ),
            session.model_dump_json(),
            ex=self.ttl_seconds,
        )

    async def commit(
        self,
        lease: ConversationLease,
        session: ConversationSession | None = None,
        record: ProcessingRecord | None = None,
    ) -> None:
        lease.ensure()
        if record is not None:
            _stamp(record)
        # The venue memory changes with the session, under the same fence.
        memory = session.memory() if session is not None else None
        committed = await self.client.eval(
            self._COMMIT_SCRIPT,
            4,
            lock_key(lease.identity),
            session_key(lease.identity),
            (
                record_key(record.message_id)
                if record is not None
                else session_key(lease.identity)
            ),
            memory_key(lease.identity),
            lease.token,
            session.model_dump_json() if session is not None else "",
            self.ttl_seconds,
            record.model_dump_json() if record is not None else "",
            self._record_ttl(record) if record is not None else 0,
            memory.model_dump_json() if memory is not None else "",
            self.memory_ttl_seconds,
        )
        if not committed:
            lease.mark_lost()
            raise LockLostError("La conversación pasó a otro proceso.")

    async def get_record(
        self,
        message_id: str,
    ) -> ProcessingRecord | None:
        raw = await self.client.get(record_key(message_id))
        if not raw:
            return None
        return ProcessingRecord.model_validate_json(raw)

    async def put_record(
        self,
        record: ProcessingRecord,
    ) -> None:
        _stamp(record)
        await self.client.set(
            record_key(record.message_id),
            record.model_dump_json(),
            ex=self._record_ttl(record),
        )

    async def record_outbound_status(
        self,
        ref: str,
        status: str,
    ) -> None:
        if status not in OUTBOUND_STATUSES:
            return
        await self.client.set(
            outbound_key(ref),
            status,
            ex=self.outbound_status_ttl_seconds,
        )

    async def outbound_status(self, ref: str) -> str:
        return str(await self.client.get(outbound_key(ref)) or "")

    async def ping(self) -> bool:
        return bool(await self.client.ping())

    async def close(self) -> None:
        await self.client.aclose()


class MemoryConversationStore:
    def __init__(self) -> None:
        self.items: dict[
            str,
            ConversationSession,
        ] = {}
        self.records: dict[str, ProcessingRecord] = {}
        self.outbound: dict[str, str] = {}
        self.memories: dict[str, CustomerMemory] = {}

    def _key(
        self,
        phone: str,
        channel_key: str,
    ) -> str:
        return _session_identity(
            phone,
            channel_key,
        )

    async def get(
        self,
        phone: str,
        channel_key: str = "default",
    ) -> ConversationSession | None:
        value = self.items.get(
            self._key(phone, channel_key)
        )
        if value is None:
            return None
        return value.model_copy(deep=True)

    async def get_memory(
        self,
        phone: str,
        channel_key: str = "default",
    ) -> CustomerMemory | None:
        value = self.memories.get(self._key(phone, channel_key))
        return value.model_copy(deep=True) if value else None

    async def save(
        self,
        session: ConversationSession,
    ) -> None:
        self.items[
            self._key(
                session.phone,
                session.channel_key,
            )
        ] = session.model_copy(deep=True)

    async def commit(
        self,
        lease: ConversationLease,
        session: ConversationSession | None = None,
        record: ProcessingRecord | None = None,
    ) -> None:
        # The ownership check and both writes run without yielding, which is
        # atomic within one event loop.
        lease.ensure()
        if not await lease.lock.owns(lease.identity, lease.token):
            lease.mark_lost()
            raise LockLostError("La conversación pasó a otro proceso.")
        if session is not None:
            self.items[lease.identity] = session.model_copy(deep=True)
            memory = session.memory()
            if memory is not None:
                self.memories[lease.identity] = memory
        if record is not None:
            self.records[message_ref(record.message_id)] = (
                _stamp(record).model_copy(deep=True)
            )

    async def get_record(
        self,
        message_id: str,
    ) -> ProcessingRecord | None:
        record = self.records.get(message_ref(message_id))
        return record.model_copy(deep=True) if record else None

    async def put_record(
        self,
        record: ProcessingRecord,
    ) -> None:
        self.records[message_ref(record.message_id)] = (
            _stamp(record).model_copy(deep=True)
        )

    async def record_outbound_status(
        self,
        ref: str,
        status: str,
    ) -> None:
        if status in OUTBOUND_STATUSES:
            self.outbound[ref] = status

    async def outbound_status(self, ref: str) -> str:
        return self.outbound.get(ref, "")

    async def ping(self) -> bool:
        return True
