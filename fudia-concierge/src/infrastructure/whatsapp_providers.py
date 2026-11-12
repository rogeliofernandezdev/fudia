"""Resolve immutable channel references. Never select the current number to
send a reply or a notice that belongs to an earlier conversation.
"""

from __future__ import annotations

import hashlib
import json
import logging
import re
import secrets
import time
from typing import Any, Protocol
from urllib.parse import quote, urlsplit
from uuid import uuid4

import httpx

from config.settings import Settings
from src.domain.whatsapp import ChannelRoute
from src.infrastructure.web_chat import WebChatAdapter, WebChatStore
from src.infrastructure.whatsapp_adapter import MetaWhatsAppAdapter
from src.observability import log_event
from src.services.ports import WhatsAppDeliveryUncertain

logger = logging.getLogger(__name__)


def digits(value: str) -> str:
    return re.sub(r"\D", "", value)


class ChannelBackend(Protocol):
    async def whatsapp_routes(self) -> list[ChannelRoute]: ...
    async def register_legacy_whatsapp_route(self, route: ChannelRoute) -> None: ...


class GatewayClient:
    def __init__(self, base: str, key: str, client: httpx.AsyncClient | None = None) -> None:
        self.base = base.rstrip("/")
        self.key = key
        self.external = client is not None
        self.client = client or httpx.AsyncClient(
            timeout=httpx.Timeout(18, connect=3), follow_redirects=False
        )

    async def request(self, method: str, path: str, body: dict[str, Any] | None = None) -> dict[str, Any]:
        parsed = urlsplit(self.base)
        if (
            not self.key
            or len(self.key) < 32
            or not parsed.hostname
            or parsed.username
            or parsed.query
            or parsed.fragment
            or (
                parsed.scheme != "https"
                and not (parsed.scheme == "http" and parsed.hostname in {"localhost", "127.0.0.1"})
            )
        ):
            raise RuntimeError("El gateway de WhatsApp no está configurado.")
        try:
            response = await self.client.request(
                method,
                self.base + path,
                headers={"X-Fudia-Gateway-Key": self.key},
                json=body,
                timeout=httpx.Timeout(60 if path == "/v1/messages" else 12, connect=3),
            )
        except httpx.TransportError as exc:
            if method == "POST" and path == "/v1/messages":
                raise WhatsAppDeliveryUncertain("No pudimos confirmar el envío del gateway.") from exc
            raise RuntimeError("No pudimos consultar el gateway de WhatsApp.") from exc
        response.raise_for_status()
        data: dict[str, Any] = response.json()
        return data

    async def status(self, ref: str, country: str = "") -> dict[str, Any]:
        if not re.fullmatch(r"[0-9a-fA-F-]{36}", ref):
            raise RuntimeError("La vinculación no es válida.")
        return await self.request("GET", f"/v1/sessions/{ref}?countryCode={quote(country, safe='')}")

    async def close(self) -> None:
        if not self.external:
            await self.client.aclose()


class GatewayWhatsAppAdapter:
    supports_templates = False

    def __init__(self, gateway: GatewayClient, ref: str, callback: str = "") -> None:
        self.gateway, self.ref, self.callback = gateway, ref, callback

    def for_event(self, event_id: str) -> GatewayWhatsAppAdapter:
        callback = hashlib.sha256(f"partner:{event_id}".encode()).hexdigest()
        return GatewayWhatsAppAdapter(self.gateway, self.ref, callback)

    async def _send(self, phone: str, body: dict[str, Any], callback: str = "") -> None:
        ref = callback or self.callback or hashlib.sha256(uuid4().bytes).hexdigest()
        result = await self.gateway.request(
            "POST",
            "/v1/messages",
            {
                "sessionRef": self.ref,
                "recipient": phone,
                "callbackRef": ref,
                **body,
            },
        )
        if result.get("state") not in {"sent", "delivered", "read"}:
            raise WhatsAppDeliveryUncertain("El gateway todavía no confirmó el envío.")

    async def send_text(self, phone: str, message: str, callback_ref: str = "") -> None:
        await self._send(phone, {"kind": "text", "text": message}, callback_ref)

    async def send_image(self, phone: str, url: str, caption: str = "", callback_ref: str = "") -> None:
        await self._send(phone, {"kind": "image", "imageUrl": url, "caption": caption}, callback_ref)

    async def send_buttons(self, phone: str, body: str, buttons: list[tuple[str, str]]) -> None:
        await self._send(
            phone,
            {
                "kind": "text",
                "text": body,
                "buttons": [{"id": key, "title": title} for key, title in buttons],
            },
        )

    async def send_template(
        self,
        phone: str,
        name: str,
        language: str,
        body: list[str],
        button_payloads: list[str],
    ) -> None:
        raise RuntimeError("Este proveedor no usa plantillas de Meta.")


class WhatsAppProviders:
    def __init__(
        self,
        cfg: Settings,
        backend: ChannelBackend,
        *,
        gateway: GatewayClient | None = None,
        meta_client: httpx.AsyncClient | None = None,
    ) -> None:
        self.cfg, self.backend = cfg, backend
        self.web = WebChatStore(cfg.redis_connection_url, cfg.session_ttl_seconds)
        self.gateway = gateway or GatewayClient(
            cfg.whatsapp_gateway_url, cfg.whatsapp_gateway_key.get_secret_value()
        )
        self.accounts: dict[str, tuple[str, MetaWhatsAppAdapter]] = {}
        raw = cfg.whatsapp_meta_accounts.get_secret_value()
        try:
            entries: Any = json.loads(raw) if raw else []
            if not isinstance(entries, list):
                raise ValueError
            if cfg.whatsapp_phone_id and cfg.whatsapp_token:
                entries = [
                    {
                        "countryCode": cfg.whatsapp_country_code,
                        "phoneId": cfg.whatsapp_phone_id,
                        "token": cfg.whatsapp_token,
                    },
                    *entries,
                ]
            for entry in entries:
                country, phone_id, token = entry["countryCode"], entry["phoneId"], entry["token"]
                if (
                    not re.fullmatch(r"[A-Z]{2}", country)
                    or not re.fullmatch(
                        r"[0-9]{4,80}" if cfg.env.lower() == "production" else r"[A-Za-z0-9_-]{1,80}",
                        phone_id,
                    )
                    or not isinstance(token, str)
                    or not token
                    or phone_id in self.accounts
                ):
                    raise ValueError
                self.accounts[phone_id] = (
                    country,
                    MetaWhatsAppAdapter(
                        token,
                        phone_id,
                        cfg.whatsapp_graph_version,
                        client=meta_client,
                    ),
                )
        except (ValueError, TypeError, KeyError) as exc:
            raise ValueError("WHATSAPP_META_ACCOUNTS contiene una configuración inválida.") from exc
        self._routes: list[ChannelRoute] = []
        self._expires = 0.0

    async def routes(self, *, fresh: bool = False) -> list[ChannelRoute]:
        if fresh or time.monotonic() >= self._expires:
            self._routes = await self.backend.whatsapp_routes()
            self._expires = time.monotonic() + 3
        return self._routes

    async def current_meta(self, phone_id: str) -> ChannelRoute | None:
        routes = await self.routes()
        matching = [
            route
            for route in routes
            if route.active
            and route.provider == "META_CLOUD_API"
            and route.providerKey == phone_id
            and phone_id in self.accounts
        ]
        return matching[0] if len(matching) == 1 else None

    async def historical_meta(self, phone_id: str) -> ChannelRoute | None:
        # Only delivery action tokens may enter through an inactive account.
        # Ordinary customer messages still require current_meta.
        if phone_id not in self.accounts:
            return None
        for route in await self.routes():
            if route.provider == "META_CLOUD_API" and route.providerKey == phone_id:
                if route.countryCode == self.accounts[phone_id][0]:
                    return route
        return None

    async def current_gateway(self, session_ref: str, *, historical: bool = False) -> ChannelRoute | None:
        routes = await self.routes()
        matching = [
            route
            for route in routes
            if route.provider == "WHATSMEOW"
            and route.providerKey == session_ref
            and (historical or route.active)
        ]
        return matching[0] if matching else None

    async def for_channel(
        self, channel_key: str
    ) -> MetaWhatsAppAdapter | GatewayWhatsAppAdapter | WebChatAdapter:
        routes = [route for route in await self.routes() if route.routingKey == channel_key]
        if len(routes) != 1:
            raise RuntimeError("No pudimos resolver el canal original de la conversación.")
        route = routes[0]
        if route.provider == "FUDIA_WEB":
            self.web_origin()
            return WebChatAdapter(self.web, route.routingKey)
        if route.provider == "META_CLOUD_API":
            account = self.accounts.get(route.providerKey)
            if account is None or account[0] != route.countryCode:
                raise RuntimeError("El activo de Meta original no está configurado.")
            # A reassigned asset must never send an old reply from its new number.
            actual = await account[1].display_phone_digits(force_refresh=True)
            if actual != digits(route.phoneNumber):
                raise RuntimeError("El número del canal original cambió en Meta.")
            return account[1]
        status = await self.gateway.status(route.providerKey, route.countryCode)
        if status.get("state") != "connected" or digits(str(status.get("phoneNumber", ""))) != digits(
            route.phoneNumber
        ):
            raise RuntimeError("La sesión de WhatsApp original no está conectada.")
        return GatewayWhatsAppAdapter(self.gateway, route.providerKey)

    async def start_url_for_country(self, country: str, token: str, text: str = "") -> str:
        routes = [
            route for route in await self.routes(fresh=True) if route.active and route.countryCode == country
        ]
        if len(routes) != 1:
            raise RuntimeError("El país no tiene un canal de WhatsApp activo.")
        route = routes[0]
        if route.provider == "FUDIA_WEB":
            # Every scan carries its own reference; reloading the page repeats
            # it. It lets a new scan reopen a table whose attention ended.
            scan = secrets.token_urlsafe(12)
            return f"{self.web_origin()}/chat/{quote(token, safe='')}?s={scan}"
        await self.for_channel(route.routingKey)
        return f"https://wa.me/{digits(route.phoneNumber)}?text={quote(text or f'FUDIA:{token}', safe='')}"

    def web_origin(self) -> str:
        parsed = urlsplit(self.cfg.web_chat_url)
        if (not parsed.hostname or parsed.username or parsed.password or parsed.query or parsed.fragment
            or parsed.path not in {"", "/"}
            or (parsed.scheme != "https" and not (
                parsed.scheme == "http" and parsed.hostname in {"localhost", "127.0.0.1"}
            ))):
            raise RuntimeError("El origen público del chat web no está configurado.")
        return self.cfg.web_chat_url.rstrip("/")

    async def verify(self, country: str, provider: str, phone: str, session_ref: str = "") -> dict[str, Any]:
        out: dict[str, Any] = {
            "provider": provider,
            "countryCode": country,
            "state": "unconfigured",
            "phoneNumber": "",
        }
        if provider == "FUDIA_WEB":
            self.web_origin()
            out.update(state="connected", providerKey="web:" + country)
            return out
        if provider == "META_CLOUD_API":
            for phone_id, (account_country, account) in self.accounts.items():
                if account_country != country:
                    continue
                actual, connected = await account.connection_info()
                if actual == digits(phone):
                    out.update(
                        phoneNumber="+" + actual,
                        state="connected" if connected else "disconnected",
                        providerKey=phone_id,
                    )
                    return out
            return out
        if provider != "WHATSMEOW":
            raise RuntimeError("Proveedor no válido.")
        if not session_ref:
            return out
        status = await self.gateway.status(session_ref, country)
        if status.get("countryCode") != country:
            raise RuntimeError("La sesión pertenece a otro país.")
        out.update(status)
        out["providerKey"] = session_ref
        if digits(str(status.get("phoneNumber", ""))) != digits(phone):
            out["state"] = "number_mismatch"
        out.pop("qrCode", None)
        return out

    async def link(self, country: str, session_ref: str = "") -> dict[str, Any]:
        if not re.fullmatch(r"[A-Z]{2}", country):
            raise RuntimeError("Selecciona un país válido.")
        if session_ref:
            return await self.gateway.status(session_ref, country)
        return await self.gateway.request("POST", "/v1/link", {"countryCode": country})

    async def initialize(self) -> None:
        account = self.accounts.get(self.cfg.whatsapp_phone_id)
        if account is None:
            return
        try:
            phone, connected = await account[1].connection_info()
            if not connected:
                return
            await self.backend.register_legacy_whatsapp_route(
                ChannelRoute(
                    routingKey=self.cfg.whatsapp_phone_id,
                    countryCode=account[0],
                    provider="META_CLOUD_API",
                    providerKey=self.cfg.whatsapp_phone_id,
                    phoneNumber="+" + phone,
                    legacy=True,
                )
            )
        except Exception as exc:
            log_event(logger, "concierge.legacy_route.unavailable", errorType=type(exc).__name__)

    async def send_text(self, phone: str, message: str, callback_ref: str = "") -> None:
        raise RuntimeError("Se requiere el canal original antes de enviar.")

    async def send_image(self, phone: str, url: str, caption: str = "", callback_ref: str = "") -> None:
        raise RuntimeError("Se requiere el canal original antes de enviar.")

    async def send_template(
        self,
        phone: str,
        name: str,
        language: str,
        body: list[str],
        button_payloads: list[str],
    ) -> None:
        raise RuntimeError("Se requiere el canal original antes de enviar.")

    async def send_buttons(self, phone: str, body: str, buttons: list[tuple[str, str]]) -> None:
        raise RuntimeError("Se requiere el canal original antes de enviar.")

    async def close(self) -> None:
        for _, account in self.accounts.values():
            await account.close()
        await self.gateway.close()
        await self.web.close()
