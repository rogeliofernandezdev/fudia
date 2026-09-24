from __future__ import annotations

from typing import Protocol

import redis.asyncio as redis

from src.domain.models import ConversationSession


class ConversationStore(Protocol):
    async def get(self, phone: str) -> ConversationSession | None: ...
    async def save(self, session: ConversationSession) -> None: ...


class RedisConversationStore:
    def __init__(self, redis_url: str, ttl_seconds: int) -> None:
        self.client = redis.from_url(redis_url, decode_responses=True)
        self.ttl_seconds = ttl_seconds

    def _key(self, phone: str) -> str:
        return f"fudia:concierge:session:{phone}"

    async def get(self, phone: str) -> ConversationSession | None:
        raw = await self.client.get(self._key(phone))
        if not raw:
            return None
        return ConversationSession.model_validate_json(raw)

    async def save(self, session: ConversationSession) -> None:
        await self.client.set(
            self._key(session.phone),
            session.model_dump_json(),
            ex=self.ttl_seconds,
        )

    async def close(self) -> None:
        await self.client.aclose()


class MemoryConversationStore:
    def __init__(self) -> None:
        self.items: dict[str, ConversationSession] = {}

    async def get(self, phone: str) -> ConversationSession | None:
        value = self.items.get(phone)
        if value is None:
            return None
        return ConversationSession.model_validate_json(value.model_dump_json())

    async def save(self, session: ConversationSession) -> None:
        self.items[session.phone] = ConversationSession.model_validate_json(
            session.model_dump_json()
        )
