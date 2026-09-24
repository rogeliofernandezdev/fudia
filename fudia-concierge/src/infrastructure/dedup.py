from __future__ import annotations

from typing import Protocol

import redis.asyncio as redis


class MessageDeduplicator(Protocol):
    async def claim(self, message_id: str) -> bool: ...


class RedisMessageDeduplicator:
    def __init__(self, redis_url: str) -> None:
        self.client = redis.from_url(redis_url, decode_responses=True)

    async def claim(self, message_id: str) -> bool:
        if not message_id:
            return True
        result = await self.client.set(
            f"fudia:concierge:message:{message_id}",
            "1",
            ex=86_400,
            nx=True,
        )
        return bool(result)

    async def close(self) -> None:
        await self.client.aclose()


class MemoryMessageDeduplicator:
    def __init__(self) -> None:
        self.ids: set[str] = set()

    async def claim(self, message_id: str) -> bool:
        if not message_id:
            return True
        if message_id in self.ids:
            return False
        self.ids.add(message_id)
        return True
