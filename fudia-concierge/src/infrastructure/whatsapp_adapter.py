from __future__ import annotations

from typing import Protocol

import httpx


class WhatsAppAdapter(Protocol):
    async def send_text(
        self, phone: str, message: str, sender_phone_id: str = ""
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
            self._client = httpx.AsyncClient(timeout=15.0)
        return self._client

    async def send_text(
        self, phone: str, message: str, sender_phone_id: str = ""
    ) -> None:
        phone_id = sender_phone_id.strip() or self.phone_id
        if not self.token or not phone_id or not self.graph_version:
            raise RuntimeError("WhatsApp no está configurado.")
        url = (
            f"https://graph.facebook.com/{self.graph_version}/"
            f"{phone_id}/messages"
        )
        response = await self._http().post(
            url,
            headers={"Authorization": f"Bearer {self.token}"},
            json={
                "messaging_product": "whatsapp",
                "to": phone,
                "type": "text",
                "text": {"body": message},
            },
        )
        response.raise_for_status()

    async def close(self) -> None:
        if self._client is not None and self._external_client is None:
            await self._client.aclose()
            self._client = None
