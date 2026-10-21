from __future__ import annotations

import redis.asyncio as redis

from src.services.ports import TableStartCodes as TableStartCodes
from src.services.start_codes import new_start_code


class RedisTableStartCodes:
    """Start codes shared by the HTTP ingress (issues them at scan time) and
    the worker (resolves them when the first WhatsApp message arrives)."""

    _ATTEMPTS = 8

    def __init__(
        self,
        redis_url: str,
        ttl_seconds: int,
        client: redis.Redis | None = None,
    ) -> None:
        self.client = client or redis.from_url(
            redis_url,
            decode_responses=True,
        )
        self.ttl_seconds = ttl_seconds

    @staticmethod
    def _key(code: str) -> str:
        return f"fudia:concierge:startcode:{code.upper()}"

    async def issue(self, qr_token: str) -> str:
        # NX: a live code is never reassigned to another table.
        for _ in range(self._ATTEMPTS):
            code = new_start_code()
            if await self.client.set(
                self._key(code),
                qr_token,
                nx=True,
                ex=self.ttl_seconds,
            ):
                return code
        raise RuntimeError("No pudimos generar un código de mesa libre.")

    async def resolve(self, code: str) -> str | None:
        value = await self.client.get(self._key(code))
        return str(value) if value else None

    async def close(self) -> None:
        await self.client.aclose()
