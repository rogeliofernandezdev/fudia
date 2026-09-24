from __future__ import annotations

import asyncio
import hashlib
from typing import Protocol
from uuid import uuid4

import redis.asyncio as redis


def conversation_identity(
    phone: str,
    channel_key: str,
) -> str:
    raw = f"{channel_key or 'default'}|{phone}"
    return hashlib.sha256(
        raw.encode("utf-8")
    ).hexdigest()


class ConversationBusyError(RuntimeError):
    pass


class ConversationLock(Protocol):
    async def acquire(
        self,
        identity: str,
    ) -> str: ...

    async def release(
        self,
        identity: str,
        token: str,
    ) -> None: ...


class RedisConversationLock:
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
        key = f"fudia:concierge:lock:{identity}"
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

    async def release(
        self,
        identity: str,
        token: str,
    ) -> None:
        key = f"fudia:concierge:lock:{identity}"
        await self.client.eval(
            self._RELEASE_SCRIPT,
            1,
            key,
            token,
        )

    async def close(self) -> None:
        await self.client.aclose()


class MemoryConversationLock:
    def __init__(self) -> None:
        self._locks: dict[str, asyncio.Lock] = {}

    async def acquire(
        self,
        identity: str,
    ) -> str:
        lock = self._locks.setdefault(
            identity,
            asyncio.Lock(),
        )
        await lock.acquire()
        return identity

    async def release(
        self,
        identity: str,
        token: str,
    ) -> None:
        lock = self._locks.get(identity)
        if lock is not None and lock.locked():
            lock.release()

    async def close(self) -> None:
        return None
