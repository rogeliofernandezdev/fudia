"""Application ports: the chatbot owns interfaces, not adapter implementations."""

from __future__ import annotations

import asyncio
import hashlib
import time
from collections.abc import Callable
from dataclasses import dataclass
from typing import Any, Protocol
from uuid import uuid4

from src.domain.models import (
    BillSummary,
    ComboDetail,
    ConversationSession,
    CustomerMemory,
    DeliveryOptions,
    DeliveryQuote,
    InboundMessage,
    MenuItem,
    MenuResponse,
    ModifierConfig,
    OrderResult,
    OrderStatus,
    ProcessingRecord,
    TableContext,
)
from src.services.start_codes import new_start_code


class FudiaError(RuntimeError):
    def __init__(self, status_code: int, code: str, message: str) -> None:
        super().__init__(message)
        self.status_code = status_code
        self.code = code
        self.message = message


class FudiaBackend(Protocol):
    async def resolve_table(self, token: str) -> TableContext: ...
    async def search_menu(
        self, token: str, query: str = "", product_id: str = ""
    ) -> MenuResponse: ...
    async def get_product(self, token: str, product_id: str) -> MenuItem | None: ...
    async def get_combo(self, token: str, product_id: str) -> ComboDetail: ...
    async def get_modifiers(self, token: str, product_id: str) -> ModifierConfig: ...
    async def request_handoff(
        self, token: str, phone: str, conversation_id: str, reason: str
    ) -> dict[str, Any]: ...
    async def get_handoff_status(
        self, token: str, conversation_id: str
    ) -> dict[str, Any]: ...
    async def create_order(
        self, token: str, phone: str, lines: list[dict[str, Any]],
        conversation_id: str, request_id: str
    ) -> OrderResult: ...
    async def request_bill(
        self, token: str, phone: str, conversation_id: str, request_id: str = ""
    ) -> BillSummary: ...
    async def get_order_status(
        self, token: str, order_id: str, conversation_id: str
    ) -> OrderStatus: ...
    async def get_delivery_options(self, token: str) -> DeliveryOptions: ...
    async def quote_delivery(
        self, token: str, latitude: float, longitude: float
    ) -> DeliveryQuote: ...
    async def create_delivery_order(
        self, token: str, order: dict[str, Any]
    ) -> OrderResult: ...
    async def open_conversation(
        self, token: str, conversation_id: str, kind: str
    ) -> None:
        """Count a new attention; idempotent by `conversation_id`. Raises
        `FudiaError` with `CONVERSATION_LIMIT_CODE` when no new conversation
        may start this month."""
    async def report_usage(
        self,
        token: str,
        conversation_id: str,
        request_id: str,
        tiers: list[dict[str, Any]],
    ) -> None: ...


# foods-backend rejects opening a conversation (never one in progress).
CONVERSATION_LIMIT_CODE = "concierge_conversation_limit_reached"


class LockLostError(RuntimeError):
    """The conversation lease expired or moved to another worker.

    Raised before any further effect or state write so a stale worker never
    overwrites the state produced by the new owner.
    """


class ConversationBusyError(RuntimeError):
    pass


class WhatsAppDeliveryUncertain(RuntimeError):
    """The request may have reached Meta but no answer came back.

    The message could already be on the diner's phone, so it must be
    reconciled instead of blindly sent again.
    """


def conversation_identity(phone: str, channel_key: str) -> str:
    raw = f"{channel_key or 'default'}|{phone}"
    return hashlib.sha256(raw.encode("utf-8")).hexdigest()


def message_ref(message_id: str) -> str:
    """Opaque, stable reference of an inbound message (record key and the
    `biz_opaque_callback_data` echoed by Meta status webhooks)."""
    return hashlib.sha256(message_id.encode("utf-8")).hexdigest()


class ConversationLock(Protocol):
    async def acquire(self, identity: str) -> str: ...
    async def refresh(self, identity: str, token: str) -> bool: ...
    async def owns(self, identity: str, token: str) -> bool: ...
    async def release(self, identity: str, token: str) -> None: ...


class ConversationLease:
    """Ownership of one conversation for the duration of a message."""

    def __init__(
        self,
        lock: ConversationLock,
        identity: str,
        token: str,
    ) -> None:
        self.lock = lock
        self.identity = identity
        self.token = token
        self.lost = False

    def mark_lost(self) -> None:
        self.lost = True

    def ensure(self) -> None:
        if self.lost:
            raise LockLostError("La conversación pasó a otro proceso.")

    async def verify(self) -> None:
        self.ensure()
        if not await self.lock.owns(self.identity, self.token):
            self.mark_lost()
            raise LockLostError("La conversación pasó a otro proceso.")


class ProcessingRecords(Protocol):
    async def get(
        self, phone: str, channel_key: str = "default"
    ) -> ConversationSession | None: ...
    async def get_memory(
        self, phone: str, channel_key: str = "default"
    ) -> CustomerMemory | None: ...
    async def get_record(self, message_id: str) -> ProcessingRecord | None: ...
    async def put_record(self, record: ProcessingRecord) -> None: ...
    async def outbound_status(self, ref: str) -> str: ...


class ConversationStore(ProcessingRecords, Protocol):
    async def save(self, session: ConversationSession) -> None: ...
    async def commit(
        self,
        lease: ConversationLease,
        session: ConversationSession | None = None,
        record: ProcessingRecord | None = None,
    ) -> None:
        """Atomically persist session and record only while `lease` still
        owns the conversation; otherwise raise `LockLostError`."""
        ...
    async def record_outbound_status(self, ref: str, status: str) -> None: ...
    async def ping(self) -> bool: ...


class RateLimiter(Protocol):
    async def allow(self, identity: str) -> bool: ...


@dataclass
class QueueDelivery:
    entry_id: str
    message: InboundMessage
    attempt: int
    # Per-conversation order assigned at enqueue; 0 means unordered.
    sequence: int = 0
    conversation: str = ""


@dataclass
class QueueStats:
    pending: int = 0
    dead_letter: int = 0
    stream_length: int = 0


class InboundQueue(Protocol):
    async def start(self) -> None: ...
    async def enqueue(self, message: InboundMessage) -> bool: ...
    async def read(self, limit: int = 10) -> list[QueueDelivery]: ...
    async def wait_turn(self, delivery: QueueDelivery, timeout: float) -> bool: ...
    async def ack(self, delivery: QueueDelivery) -> None: ...
    async def defer(self, delivery: QueueDelivery) -> None: ...
    async def touch(self, delivery: QueueDelivery) -> bool: ...
    async def retry(self, delivery: QueueDelivery, error: str = "") -> bool: ...
    async def trim(self) -> int: ...
    async def stats(self) -> QueueStats: ...
    async def ping(self) -> bool: ...


class LocalConversationLock:
    """FIFO lock with real tokens. Safe only in one process; production
    injects RedisConversationLock."""

    def __init__(self) -> None:
        self._owners: dict[str, str] = {}
        self._waiters: dict[str, list[asyncio.Future[None]]] = {}

    async def acquire(self, identity: str) -> str:
        while identity in self._owners:
            waiter: asyncio.Future[None] = (
                asyncio.get_running_loop().create_future()
            )
            self._waiters.setdefault(identity, []).append(waiter)
            try:
                await waiter
            except asyncio.CancelledError:
                self._remove(identity, waiter)
                if identity not in self._owners:
                    self._wake(identity)
                raise
            self._remove(identity, waiter)
        token = uuid4().hex
        self._owners[identity] = token
        return token

    async def refresh(self, identity: str, token: str) -> bool:
        return self._owners.get(identity) == token

    async def owns(self, identity: str, token: str) -> bool:
        return self._owners.get(identity) == token

    async def release(self, identity: str, token: str) -> None:
        if self._owners.get(identity) == token:
            self._free(identity)

    def expire(self, identity: str) -> None:
        """Simulate a lease that expired in Redis (tests and local runs)."""
        if identity in self._owners:
            self._free(identity)

    def _free(self, identity: str) -> None:
        del self._owners[identity]
        self._wake(identity)

    def _wake(self, identity: str) -> None:
        for waiter in self._waiters.get(identity, []):
            if not waiter.done():
                waiter.set_result(None)
                return

    def _remove(self, identity: str, waiter: asyncio.Future[None]) -> None:
        waiters = self._waiters.get(identity, [])
        if waiter in waiters:
            waiters.remove(waiter)
        if not waiters:
            self._waiters.pop(identity, None)

    async def close(self) -> None:
        return None


class LocalProcessingRecords:
    """In-process record store used when no shared store is injected."""

    def __init__(self) -> None:
        self.records: dict[str, ProcessingRecord] = {}
        self.outbound: dict[str, str] = {}

    async def get(
        self, phone: str, channel_key: str = "default"
    ) -> ConversationSession | None:
        return None

    async def get_memory(
        self, phone: str, channel_key: str = "default"
    ) -> CustomerMemory | None:
        return None

    async def get_record(self, message_id: str) -> ProcessingRecord | None:
        record = self.records.get(message_ref(message_id))
        return record.model_copy(deep=True) if record else None

    async def put_record(self, record: ProcessingRecord) -> None:
        self.records[message_ref(record.message_id)] = record.model_copy(
            deep=True
        )

    async def outbound_status(self, ref: str) -> str:
        return self.outbound.get(ref, "")


class AllowAllRateLimiter:
    async def allow(self, identity: str) -> bool:
        return True

    async def close(self) -> None:
        return None


class TableStartCodes(Protocol):
    """Short-lived codes that stand for a QR token in the start message."""

    async def issue(self, qr_token: str) -> str: ...
    async def resolve(self, code: str) -> str | None: ...


class LocalTableStartCodes:
    """In-process codes; production injects RedisTableStartCodes so the HTTP
    ingress that issues a code and the worker that reads it share them."""

    def __init__(
        self,
        ttl_seconds: float = 600.0,
        clock: Callable[[], float] = time.time,
    ) -> None:
        self.ttl_seconds = ttl_seconds
        self.clock = clock
        self.codes: dict[str, tuple[str, float]] = {}

    async def issue(self, qr_token: str) -> str:
        now = self.clock()
        while True:
            code = new_start_code()
            current = self.codes.get(code)
            if current is None or current[1] <= now:
                self.codes[code] = (qr_token, now + self.ttl_seconds)
                return code

    async def resolve(self, code: str) -> str | None:
        entry = self.codes.get(code.upper())
        if entry is None or entry[1] <= self.clock():
            return None
        return entry[0]
