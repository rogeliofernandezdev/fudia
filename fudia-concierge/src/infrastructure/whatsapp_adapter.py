from __future__ import annotations

import asyncio
from typing import Protocol

import httpx


class WhatsAppAdapter(Protocol):
    async def send_text(
        self,
        phone: str,
        message: str,
        sender_phone_id: str = "",
    ) -> None: ...


class MetaWhatsAppAdapter:
    def __init__(
        self,
        token: str,
        phone_id: str,
        graph_version: str,
        client: httpx.AsyncClient | None = None,
    ) -> None:
        self.token = token
        self.phone_id = phone_id
        self.graph_version = graph_version.strip("/")
        self._external_client = client
        self._client: httpx.AsyncClient | None = client

    def _http(self) -> httpx.AsyncClient:
        if self._client is None:
            self._client = httpx.AsyncClient(
                timeout=httpx.Timeout(
                    15.0,
                    connect=3.0,
                )
            )
        return self._client

    async def send_text(
        self,
        phone: str,
        message: str,
        sender_phone_id: str = "",
    ) -> None:
        phone_id = (
            sender_phone_id.strip()
            or self.phone_id
        )
        if (
            not self.token
            or not phone_id
            or not self.graph_version
        ):
            raise RuntimeError(
                "WhatsApp no está configurado."
            )

        url = (
            "https://graph.facebook.com/"
            f"{self.graph_version}/"
            f"{phone_id}/messages"
        )
        response: httpx.Response | None = None
        for attempt in range(3):
            try:
                response = await self._http().post(
                    url,
                    headers={
                        "Authorization": (
                            f"Bearer {self.token}"
                        )
                    },
                    json={
                        "messaging_product": "whatsapp",
                        "to": phone,
                        "type": "text",
                        "text": {"body": message},
                    },
                )
            except httpx.TransportError:
                if attempt == 2:
                    raise
                await asyncio.sleep(
                    0.2 * (2**attempt)
                )
                continue

            should_retry = (
                response.status_code == 429
                or response.status_code >= 500
            )
            if should_retry and attempt < 2:
                retry_after = response.headers.get(
                    "Retry-After",
                    "",
                )
                try:
                    delay = min(
                        float(retry_after),
                        5.0,
                    )
                except ValueError:
                    delay = 0.2 * (2**attempt)
                await asyncio.sleep(delay)
                continue
            break

        if response is None:
            raise RuntimeError(
                "WhatsApp no respondió."
            )
        response.raise_for_status()

    async def close(self) -> None:
        if (
            self._client is not None
            and self._external_client is None
        ):
            await self._client.aclose()
            self._client = None
