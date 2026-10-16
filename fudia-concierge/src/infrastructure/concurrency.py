from __future__ import annotations

import asyncio
from uuid import uuid4

import redis.asyncio as redis

from src.services.ports import ConversationBusyError as ConversationBusyError
from src.services.ports import ConversationLock as ConversationLock
from src.services.ports import LocalConversationLock
from src.services.ports import conversation_identity as conversation_identity


def lock_key(identity: str) -> str:
    return f"fudia:concierge:lock:{identity}"


class RedisConversationLock:
    _REFRESH_SCRIPT = """
    if redis.call('GET', KEYS[1]) == ARGV[1] then
      return redis.call('PEXPIRE', KEYS[1], ARGV[2])
    end
    return 0
    """

    _RELEASE_SCRIPT = """
    if redis.call('GET', KEYS[1]) == ARGV[1] then
      return redis.call('DEL', KEYS[1])
    end
    return 0
    """

    def __init__(
        self,
        redis_url: str,
        ttl_seconds: int = 30,
        wait_seconds: float = 8.0,
    ) -> None:
        self.client = redis.from_url(
            redis_url,
            decode_responses=True,
        )
        self.ttl_ms = ttl_seconds * 1000
        self.wait_seconds = wait_seconds

    async def acquire(
        self,
        identity: str,
    ) -> str:
        key = lock_key(identity)
        token = uuid4().hex
        deadline = (
            asyncio.get_running_loop().time()
            + self.wait_seconds
        )
        while True:
            acquired = await self.client.set(
                key,
                token,
                nx=True,
                px=self.ttl_ms,
            )
            if acquired:
                return token
            if (
                asyncio.get_running_loop().time()
                >= deadline
            ):
                raise ConversationBusyError(
                    "La conversación sigue ocupada."
                )
            await asyncio.sleep(0.05)

    async def refresh(
        self,
        identity: str,
        token: str,
    ) -> bool:
        result = await self.client.eval(
            self._REFRESH_SCRIPT,
            1,
            lock_key(identity),
            token,
            self.ttl_ms,
        )
        return bool(result)

    async def owns(
        self,
        identity: str,
        token: str,
    ) -> bool:
        return bool(
            await self.client.get(lock_key(identity))
            == token
        )

    async def release(
        self,
        identity: str,
        token: str,
    ) -> None:
        await self.client.eval(
            self._RELEASE_SCRIPT,
            1,
            lock_key(identity),
            token,
        )

    async def close(self) -> None:
        await self.client.aclose()


class MemoryConversationLock(LocalConversationLock):
    """In-process lock for tests and single-process development."""
