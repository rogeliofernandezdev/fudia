from __future__ import annotations

import time
from typing import Protocol

import redis.asyncio as redis


class RateLimiter(Protocol):
    async def allow(
        self,
        identity: str,
    ) -> bool: ...


class RedisRateLimiter:
    _SCRIPT = """
    local current = redis.call('INCR', KEYS[1])
    if current == 1 then
      redis.call('EXPIRE', KEYS[1], ARGV[1])
    end
    if current > tonumber(ARGV[2]) then
      return 0
    end
    return 1
    """

    def __init__(
        self,
        redis_url: str,
        limit: int,
        window_seconds: int,
    ) -> None:
        self.client = redis.from_url(
            redis_url,
            decode_responses=True,
        )
        self.limit = limit
        self.window_seconds = window_seconds

    async def allow(
        self,
        identity: str,
    ) -> bool:
        bucket = int(
            time.time() // self.window_seconds
        )
        key = (
            "fudia:concierge:rate:"
            f"{identity}:{bucket}"
        )
        result = await self.client.eval(
            self._SCRIPT,
            1,
            key,
            self.window_seconds + 1,
            self.limit,
        )
        return bool(result)

    async def close(self) -> None:
        await self.client.aclose()


class AllowAllRateLimiter:
    async def allow(
        self,
        identity: str,
    ) -> bool:
        return True

    async def close(self) -> None:
        return None
