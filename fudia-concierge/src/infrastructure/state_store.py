from __future__ import annotations

import hashlib
from typing import Protocol

import redis.asyncio as redis

from src.domain.models import ConversationSession


def _session_identity(
    phone: str,
    channel_key: str,
) -> str:
    raw = f"{channel_key or 'default'}|{phone}"
    return hashlib.sha256(
        raw.encode("utf-8")
    ).hexdigest()


class ConversationStore(Protocol):
    async def get(
        self,
        phone: str,
        channel_key: str = "default",
    ) -> ConversationSession | None: ...

    async def save(
        self,
        session: ConversationSession,
    ) -> None: ...

    async def ping(self) -> bool: ...


class RedisConversationStore:
    def __init__(
        self,
        redis_url: str,
        ttl_seconds: int,
    ) -> None:
        self.client = redis.from_url(
            redis_url,
            decode_responses=True,
        )
        self.ttl_seconds = ttl_seconds

    def _key(
        self,
        phone: str,
        channel_key: str,
    ) -> str:
        identity = _session_identity(
            phone,
            channel_key,
        )
        return f"fudia:concierge:session:{identity}"

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
        return ConversationSession.model_validate_json(
            value.model_dump_json()
        )

    async def save(
        self,
        session: ConversationSession,
    ) -> None:
        self.items[
            self._key(
                session.phone,
                session.channel_key,
            )
        ] = ConversationSession.model_validate_json(
            session.model_dump_json()
        )

    async def ping(self) -> bool:
        return True
