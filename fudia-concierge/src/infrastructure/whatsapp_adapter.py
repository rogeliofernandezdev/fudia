from __future__ import annotations

import asyncio
import re
import time
from typing import Protocol
from urllib.parse import quote

import httpx

from src.metrics import WHATSAPP_OUTBOUND


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
        self._display_phone_digits = ""
        self._display_phone_expires_at = 0.0

    def _http(self) -> httpx.AsyncClient:
        if self._client is None:
            self._client = httpx.AsyncClient(
                timeout=httpx.Timeout(
                    15.0,
                    connect=3.0,
                )
            )
        return self._client

    def _ensure_configured(self) -> None:
        if (
            not self.token
            or not self.phone_id
            or not self.graph_version
        ):
            raise RuntimeError(
                "WhatsApp no está configurado."
            )

    async def display_phone_digits(self) -> str:
        now = time.monotonic()
        if (
            self._display_phone_digits
            and now < self._display_phone_expires_at
        ):
            return self._display_phone_digits

        self._ensure_configured()
        url = (
            "https://graph.facebook.com/"
            f"{self.graph_version}/"
            f"{self.phone_id}"
        )
        response: httpx.Response | None = None
        for attempt in range(3):
            try:
                response = await self._http().get(
                    url,
                    headers={
                        "Authorization": (
                            f"Bearer {self.token}"
                        )
                    },
                    params={
                        "fields": "display_phone_number"
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
                "Meta no respondió."
            )
        response.raise_for_status()
        body = response.json()
        digits = re.sub(
            r"\D",
            "",
            str(
                body.get(
                    "display_phone_number",
                    "",
                )
            ),
        )
        if not re.fullmatch(
            r"[1-9][0-9]{7,14}",
            digits,
        ):
            raise RuntimeError(
                "Meta no devolvió un número de WhatsApp válido."
            )

        self._display_phone_digits = digits
        self._display_phone_expires_at = now + 300.0
        return digits

    async def start_url(
        self,
        qr_token: str,
    ) -> str:
        digits = await self.display_phone_digits()
        message = quote(
            f"FUDIA:{qr_token}",
            safe="",
        )
        return (
            f"https://wa.me/{digits}"
            f"?text={message}"
        )

    async def send_text(
        self,
        phone: str,
        message: str,
        sender_phone_id: str = "",
    ) -> None:
        self._ensure_configured()
        url = (
            "https://graph.facebook.com/"
            f"{self.graph_version}/"
            f"{self.phone_id}/messages"
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
                    WHATSAPP_OUTBOUND.labels(
                        result="transport_error"
                    ).inc()
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
            WHATSAPP_OUTBOUND.labels(
                result="no_response"
            ).inc()
            raise RuntimeError(
                "WhatsApp no respondió."
            )
        if response.is_success:
            WHATSAPP_OUTBOUND.labels(
                result="success"
            ).inc()
        else:
            WHATSAPP_OUTBOUND.labels(
                result="http_error"
            ).inc()
        response.raise_for_status()

    async def close(self) -> None:
        if (
            self._client is not None
            and self._external_client is None
        ):
            await self._client.aclose()
            self._client = None
