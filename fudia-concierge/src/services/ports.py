"""Application ports: the chatbot owns interfaces, not adapter implementations."""
from __future__ import annotations
import asyncio
import hashlib
from dataclasses import dataclass
from typing import Any, Protocol
from src.domain.models import (
    BillSummary, ComboDetail, ConversationSession, InboundMessage,
    MenuItem, MenuResponse, ModifierConfig, OrderResult, TableContext,
)


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
        self, token: str, phone: str, conversation_id: str
    ) -> BillSummary: ...


class ConversationStore(Protocol):
    async def get(
        self, phone: str, channel_key: str = "default"
    ) -> ConversationSession | None: ...
    async def save(self, session: ConversationSession) -> None: ...
    async def ping(self) -> bool: ...


class ConversationLock(Protocol):
    async def acquire(self, identity: str) -> str: ...
    async def refresh(self, identity: str, token: str) -> bool: ...
    async def release(self, identity: str, token: str) -> None: ...


class RateLimiter(Protocol):
    async def allow(self, identity: str) -> bool: ...


@dataclass
class QueueDelivery:
    entry_id: str
    message: InboundMessage
    attempt: int


class InboundQueue(Protocol):
    async def start(self) -> None: ...
    async def enqueue(self, message: InboundMessage) -> bool: ...
    async def read(self, limit: int = 10) -> list[QueueDelivery]: ...
    async def ack(self, delivery: QueueDelivery) -> None: ...
    async def touch(self, delivery: QueueDelivery) -> bool: ...
    async def retry(self, delivery: QueueDelivery) -> bool: ...
    async def ping(self) -> bool: ...


def conversation_identity(phone: str, channel_key: str) -> str:
    raw = f"{channel_key or 'default'}|{phone}"
    return hashlib.sha256(raw.encode("utf-8")).hexdigest()


class LocalConversationLock:
    """Safe only in one process; production injects RedisConversationLock."""
    def __init__(self) -> None:
        self._locks: dict[str, asyncio.Lock] = {}

    async def acquire(self, identity: str) -> str:
        lock = self._locks.setdefault(identity, asyncio.Lock())
        await lock.acquire()
        return identity

    async def refresh(self, identity: str, token: str) -> bool:
        lock = self._locks.get(identity)
        return bool(lock and lock.locked())

    async def release(self, identity: str, token: str) -> None:
        lock = self._locks.get(identity)
        if lock is not None and lock.locked():
            lock.release()


class AllowAllRateLimiter:
    async def allow(self, identity: str) -> bool:
        return True

    async def close(self) -> None:
        return None
