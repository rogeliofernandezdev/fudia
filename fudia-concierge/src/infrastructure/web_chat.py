"""Ephemeral browser capabilities and an idempotent outbound transcript.

Only the authenticated BFF can open/read a capability. Business state is still
written exclusively by ConversationStore's fenced worker commit.
"""

from __future__ import annotations

import hashlib
import json
import secrets
import time
from typing import Any

import redis.asyncio as redis


class WebChatStore:
    _APPEND = """
    if redis.call('EXISTS', KEYS[1]) == 0 then return -1 end
    if redis.call('SADD', KEYS[2], ARGV[1]) == 0 then return 0 end
    redis.call('RPUSH', KEYS[3], ARGV[2])
    redis.call('LTRIM', KEYS[3], -200, -1)
    redis.call('EXPIRE', KEYS[2], ARGV[3])
    redis.call('EXPIRE', KEYS[3], ARGV[3])
    return 1
    """

    def __init__(self, url: str, ttl: int) -> None:
        self.client = redis.from_url(url, decode_responses=True)
        self.ttl = ttl

    def key(self, ref: str) -> str:
        return "fudia:web:" + hashlib.sha256(ref.encode()).hexdigest()

    async def open(self, qr: str, visitor: str, channel: str, country: str, start: str = "start") -> str:
        ref = secrets.token_urlsafe(32)
        data = {
            "qr": qr,
            "visitor": visitor,
            "channel": channel,
            "country": country,
            # Request id of the message that opens the welcome turn.
            "start": start,
            "createdAt": time.time(),
        }
        pointer = self.key("visitor:" + visitor + ":" + qr + ":" + channel)
        # Concurrent refreshes share the same conversation and welcome turn.
        script = """
        local current = redis.call('GET', KEYS[1])
        if current then return current end
        redis.call('SET', KEYS[2], ARGV[2], 'EX', ARGV[3])
        redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[3])
        return ARGV[1]
        """
        result = await self.client.eval(script, 2, pointer, self.key(ref), ref, json.dumps(data), self.ttl)
        return str(result)

    async def resolve(self, ref: str) -> dict[str, Any] | None:
        raw = await self.client.get(self.key(ref))
        return dict(json.loads(raw)) if raw else None

    async def append(
        self, ref: str, message_id: str, role: str, text: str, image: str = "", *, hidden: bool = False
    ) -> None:
        key = self.key(ref)
        entry: dict[str, Any] = {
            "id": message_id,
            "role": role,
            "text": text,
            "imageUrl": image,
            "createdAt": time.time(),
        }
        if hidden:
            # Kept to follow its processing; never shown as a bubble.
            entry["hidden"] = True
        result = await self.client.eval(
            self._APPEND,
            3,
            key,
            key + ":seen",
            key + ":messages",
            message_id,
            json.dumps(entry),
            self.ttl,
        )
        if int(result) < 0:
            raise RuntimeError("La sesión del chat venció.")

    async def messages(self, ref: str) -> list[dict[str, Any]]:
        return [dict(json.loads(raw)) for raw in await self.client.lrange(self.key(ref) + ":messages", 0, -1)]

    async def close(self) -> None:
        await self.client.aclose()


class WebChatAdapter:
    supports_templates = False

    def __init__(self, store: WebChatStore, channel: str) -> None:
        self.store, self.channel = store, channel

    async def _append(self, recipient: str, text: str, image: str, callback: str) -> None:
        bound = await self.store.resolve(recipient)
        if not bound or bound["channel"] != self.channel:
            raise RuntimeError("El destinatario no pertenece al chat original.")
        if not callback:
            raise RuntimeError("El chat requiere una referencia idempotente.")
        await self.store.append(recipient, callback, "assistant", text, image)

    async def send_text(self, phone: str, message: str, callback_ref: str = "") -> None:
        await self._append(phone, message, "", callback_ref)

    async def notify_failure(self, phone: str, message_id: str, text: str) -> None:
        # An idempotent terminal transport marker also survives queue-key expiry.
        await self._append(phone, text, "", "failure:" + message_id)

    async def send_image(self, phone: str, url: str, caption: str = "", callback_ref: str = "") -> None:
        await self._append(phone, caption, url, callback_ref)

    async def send_buttons(self, phone: str, body: str, buttons: list[tuple[str, str]]) -> None:
        raise RuntimeError("Los botones de centrales requieren un transporte de WhatsApp.")

    async def send_template(
        self, phone: str, name: str, language: str, body: list[str], button_payloads: list[str]
    ) -> None:
        raise RuntimeError("El chat web no envía plantillas de WhatsApp.")
