from __future__ import annotations

import asyncio
import os
import re
import secrets
from typing import Any

import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from config.settings import Settings
from src.api.web_chat import router
from src.domain.models import InboundMessage, ProcessingRecord
from src.domain.web_chat import WebAction
from src.domain.whatsapp import ChannelRoute
from src.infrastructure.queue import MemoryInboundQueue
from src.infrastructure.state_store import MemoryConversationStore
from src.infrastructure.web_chat import WebChatAdapter, WebChatStore
from src.infrastructure.whatsapp_providers import WhatsAppProviders
from src.services.concierge_service import ConciergeService
from tests.test_concierge_service import FakeFudia

QR = "a" * 32
KEY = "private-test-integration-key-at-least-32-chars"


class NoModel:
    async def reply(self, *args: Any) -> str:
        raise AssertionError("Button actions must never call a language model")


def message(ref: str, text: str = "", action: WebAction | None = None) -> InboundMessage:
    return InboundMessage(
        provider="FUDIA_WEB",
        country_code="PE",
        subject="browser-visitor",
        phone="browser-session",
        sender_phone_id="web-route",
        bound_qr=QR,
        message_id="web:" + ref,
        text=text,
        web_action=action,
    )


@pytest.mark.asyncio
async def test_web_buttons_use_fenced_tools_explicit_confirmation_and_idempotency() -> None:
    store, backend = MemoryConversationStore(), FakeFudia()
    service = ConciergeService(store, backend, NoModel())
    await service.process_inbound(message("start", "FUDIA:" + QR))
    add = message(
        "add",
        "Agregar un plato",
        WebAction(name="add_item", args={"productId": "p1", "quantity": 1, "price": 0}),
    )
    await service.process_inbound(add)
    await service.process_inbound(add)
    session = await store.get("browser-session", "web-route")
    assert session and len(session.cart) == 1 and session.cart[0].quantity == 1
    assert str(session.cart[0].unit_price) == "32.50"
    assert not backend.order_calls
    await service.process_inbound(
        message("review", "Revisar mi pedido", WebAction(name="prepare_confirmation"))
    )
    session = await store.get("browser-session", "web-route")
    assert session and session.pending_order_request_id
    ref = session.pending_order_request_id
    # A stale tab cannot confirm another prepared cart.
    rejected = await service.process_inbound(
        message("stale", "Sí, confirmo mi pedido", WebAction(name="confirm_order", confirmation_ref="old"))
    )
    assert "cambió" in rejected.reply and not backend.order_calls
    confirm = message(
        "confirm", "Sí, confirmo mi pedido", WebAction(name="confirm_order", confirmation_ref=ref)
    )
    await service.process_inbound(confirm)
    await service.process_inbound(confirm)
    assert len(backend.order_calls) == 1
    assert backend.order_calls[0]["phone"] == ""
    session = await store.get("browser-session", "web-route")
    assert session and not session.cart


@pytest.mark.asyncio
async def test_web_text_cannot_change_the_qr_scope() -> None:
    store, backend = MemoryConversationStore(), FakeFudia()
    service = ConciergeService(store, backend, NoModel())
    await service.process_inbound(message("start", "FUDIA:" + QR))
    rejected = await service.process_inbound(message("other", "FUDIA:" + "b" * 32))
    assert "otro local" in rejected.reply
    session = await store.get("browser-session", "web-route")
    assert session and session.qr_token == QR and not backend.opened


def add_item(ref: str) -> InboundMessage:
    action = WebAction(name="add_item", args={"productId": "p1", "quantity": 1})
    return message(ref, "Agregar un plato", action)


@pytest.mark.asyncio
async def test_opening_the_page_counts_no_conversation_until_the_first_interaction() -> None:
    store, backend = MemoryConversationStore(), FakeFudia()
    service = ConciergeService(store, backend, NoModel())
    welcome = await service.process_inbound(message("start", "FUDIA:" + QR))
    assert "Bienvenido" in welcome.reply and not backend.opened
    # At the cap the diner is refused in the words of this channel and nothing
    # is added; the next interaction asks foods-backend again.
    backend.conversation_limit_reached = True
    refused = await service.process_inbound(add_item("add-1"))
    assert "por este chat" in refused.reply and "WhatsApp" not in refused.reply
    session = await store.get("browser-session", "web-route")
    assert session and not session.cart and not session.conversation_counted
    backend.conversation_limit_reached = False
    await service.process_inbound(add_item("add-2"))
    await service.process_inbound(add_item("add-3"))
    session = await store.get("browser-session", "web-route")
    assert session and session.conversation_counted and session.cart[0].quantity == 2
    assert backend.opened == [{"token": QR, "conversation_id": session.conversation_id, "kind": "table"}]


@pytest.mark.asyncio
async def test_cart_buttons_answer_with_the_dish_and_what_the_order_holds() -> None:
    # The tap is not echoed as a bubble, so the reply alone must be clear.
    store, backend = MemoryConversationStore(), FakeFudia()
    service = ConciergeService(store, backend, NoModel())
    await service.process_inbound(message("start", "FUDIA:" + QR))
    first = await service.process_inbound(add_item("add-1"))
    assert first.reply == "Agregué 1 × Lomo saltado.\nLlevas 1 producto · S/ 32.50."
    second = await service.process_inbound(add_item("add-2"))
    assert second.reply == "Agregué 1 × Lomo saltado.\nLlevas 2 productos · S/ 65.00."
    remove = WebAction(name="remove_item", args={"productId": "p1"})
    removed = await service.process_inbound(message("remove", "Actualizar mi pedido", remove))
    assert removed.reply == "Quité Lomo saltado de tu pedido.\nTu pedido quedó vacío."


@pytest.mark.asyncio
async def test_reopening_the_qr_keeps_a_live_table_and_reopens_an_ended_one() -> None:
    now = [1_000_000.0]
    store, backend = MemoryConversationStore(), FakeFudia()
    service = ConciergeService(store, backend, NoModel(), clock=lambda: now[0])
    await service.process_inbound(message("start:scan-1", "FUDIA:" + QR))
    await service.process_inbound(add_item("add"))
    # Another scan during the meal: same cart, same conversation, no welcome.
    again = await service.process_inbound(message("start:scan-2", "FUDIA:" + QR))
    assert again.stage == "superseded" and not again.reply
    session = await store.get("browser-session", "web-route")
    assert session and session.mode == "table" and len(session.cart) == 1 and len(backend.opened) == 1
    # Hours later the table attention is over: the chat takes delivery orders.
    now[0] += 4 * 3600
    moved = await service.process_inbound(add_item("add-later"))
    assert "ya terminó" in moved.reply
    session = await store.get("browser-session", "web-route")
    assert session and session.mode == "delivery"
    # Reloading the page repeats a scan already processed: nothing changes.
    await service.process_inbound(message("start:scan-2", "FUDIA:" + QR))
    session = await store.get("browser-session", "web-route")
    assert session and session.mode == "delivery"
    # Scanning the table QR again seats the diner at the table once more.
    seated = await service.process_inbound(message("start:scan-3", "FUDIA:" + QR))
    assert "Bienvenido" in seated.reply
    session = await store.get("browser-session", "web-route")
    assert session and session.mode == "table" and not session.cart and not session.conversation_counted


class Registry(FakeFudia):
    async def resolve_table(self, token: str):
        return (await super().resolve_table(token)).model_copy(update={"countryCode": "PE"})

    async def whatsapp_routes(self) -> list[ChannelRoute]:
        return [
            ChannelRoute(
                routingKey="web-route",
                countryCode="PE",
                provider="FUDIA_WEB",
                providerKey="web:PE",
                phoneNumber="",
                active=True,
            )
        ]


@pytest.mark.asyncio
async def test_same_qr_redirects_directly_to_web_without_meta_or_gateway() -> None:
    cfg = Settings(_env_file=None, web_chat_url="https://pedidos.example.com")
    providers = WhatsAppProviders(cfg, Registry())
    first = await providers.start_url_for_country("PE", QR)
    second = await providers.start_url_for_country("PE", QR)
    # Each scan has its own reference, so a new scan differs from a reload.
    assert re.fullmatch(re.escape("https://pedidos.example.com/chat/" + QR) + r"\?s=[A-Za-z0-9_-]{16}", first)
    assert first != second
    assert (await providers.verify("PE", "FUDIA_WEB", ""))["state"] == "connected"
    await providers.close()


@pytest.mark.asyncio
async def test_redis_transcript_deduplicates_refresh_outbox_and_refuses_other_channel() -> None:
    url = os.environ.get("WEB_CHAT_TEST_REDIS_URL")
    if not url:
        pytest.skip("WEB_CHAT_TEST_REDIS_URL not configured")
    store = WebChatStore(url, 600)
    visitor = secrets.token_urlsafe(32)
    refs = await asyncio.gather(*(store.open(QR, visitor, "web-route", "PE") for _ in range(4)))
    assert len(set(refs)) == 1
    sender = WebChatAdapter(store, "web-route")
    await sender.send_text(refs[0], "Hola", "callback")
    await sender.send_text(refs[0], "Hola", "callback")
    assert len(await store.messages(refs[0])) == 1
    with pytest.raises(RuntimeError, match="original"):
        await WebChatAdapter(store, "other-route").send_text(refs[0], "No enviar", "other")
    await store.close()


def test_private_web_api_caps_are_bound_to_visitor_qr_and_original_route() -> None:
    url = os.environ.get("WEB_CHAT_TEST_REDIS_URL")
    if not url:
        pytest.skip("WEB_CHAT_TEST_REDIS_URL not configured")
    app = FastAPI()
    app.include_router(router)
    cfg = Settings(
        _env_file=None, web_chat_url="http://localhost:5175", redis_url=url, fudia_concierge_api_key=KEY
    )
    app.state.settings = cfg
    app.state.whatsapp = WhatsAppProviders(cfg, Registry())
    app.state.fudia = Registry()
    app.state.inbound_queue = MemoryInboundQueue()
    app.state.conversation_store = MemoryConversationStore()
    visitor = secrets.token_urlsafe(32)
    headers = {"X-Fudia-Concierge-Key": KEY}
    payload = {"qr": QR, "visitor": visitor}
    with TestClient(app) as client:
        assert client.post("/internal/web-chat/session", json=payload).status_code == 401
        opened = client.post("/internal/web-chat/session", json=payload, headers=headers)
        assert opened.status_code == 200, opened.text
        capability = opened.json()["capability"]
        bound = {**payload, "capability": capability}
        assert (
            client.post(
                "/internal/web-chat/state",
                json={**bound, "visitor": secrets.token_urlsafe(32)},
                headers=headers,
            ).status_code
            == 410
        )
        assert (
            client.post(
                "/internal/web-chat/state", json={**bound, "qr": "b" * 32}, headers=headers
            ).status_code
            == 410
        )
        outgoing = {
            **bound,
            "requestId": secrets.token_hex(16),
            "action": {"name": "add_item", "args": {"productId": "p1", "quantity": 1}},
        }
        assert client.post("/internal/web-chat/messages", json=outgoing, headers=headers).status_code == 200
        assert client.post("/internal/web-chat/messages", json=outgoing, headers=headers).status_code == 200
        assert len(app.state.inbound_queue.pending) == 2  # one welcome, one add
        assert client.portal is not None
        for message in app.state.inbound_queue.pending:
            client.portal.call(
                app.state.whatsapp.web.append,
                capability,
                "failure:" + message.message.message_id,
                "assistant",
                "No pudimos procesar el mensaje.",
            )
        state = client.post("/internal/web-chat/state", json=bound, headers=headers)
        assert state.status_code == 200 and state.json()["failed"] and not state.json()["pending"]
        assert capability not in state.text
        invalid = {
            **outgoing,
            "requestId": secrets.token_hex(16),
            "action": {"name": "add_item", "args": {"quantity": -1}},
        }
        assert client.post("/internal/web-chat/messages", json=invalid, headers=headers).status_code == 400
        location = {
            **bound,
            "requestId": secrets.token_hex(16),
            "location": {"latitude": -12.04, "longitude": -77.03},
        }
        assert client.post("/internal/web-chat/messages", json=location, headers=headers).status_code == 200
        assert app.state.inbound_queue.pending[-1].message.location.latitude == -12.04
        location["location"]["latitude"] = 100
        assert client.post("/internal/web-chat/messages", json=location, headers=headers).status_code == 400


def test_polling_reuses_what_cannot_change_and_hides_the_echo_of_cart_buttons() -> None:
    url = os.environ.get("WEB_CHAT_TEST_REDIS_URL")
    if not url:
        pytest.skip("WEB_CHAT_TEST_REDIS_URL not configured")

    class CountingRegistry(Registry):
        resolved = 0

        async def resolve_table(self, token: str) -> Any:
            self.resolved += 1
            return await super().resolve_table(token)

    class CountingStore(MemoryConversationStore):
        lookups = 0

        async def get_record(self, message_id: str) -> ProcessingRecord | None:
            self.lookups += 1
            return await super().get_record(message_id)

    app = FastAPI()
    app.include_router(router)
    cfg = Settings(
        _env_file=None, web_chat_url="http://localhost:5175", redis_url=url, fudia_concierge_api_key=KEY
    )
    backend, store = CountingRegistry(), CountingStore()
    app.state.settings = cfg
    app.state.whatsapp = WhatsAppProviders(cfg, Registry())
    app.state.fudia = backend
    app.state.inbound_queue = MemoryInboundQueue()
    app.state.conversation_store = store
    headers = {"X-Fudia-Concierge-Key": KEY}
    payload = {"qr": QR, "visitor": secrets.token_urlsafe(32)}
    with TestClient(app) as client:
        opened = client.post("/internal/web-chat/session", json=payload, headers=headers)
        bound = {**payload, "capability": opened.json()["capability"]}
        add = {"name": "add_item", "args": {"productId": "p1", "quantity": 1}}
        for body in ({"action": add}, {"text": "hola"}):
            sent = client.post(
                "/internal/web-chat/messages",
                json={**bound, "requestId": secrets.token_hex(16), **body},
                headers=headers,
            )
            assert sent.status_code == 200, sent.text
        assert client.portal is not None
        for entry in app.state.inbound_queue.pending[1:]:
            record = ProcessingRecord(message_id=entry.message.message_id, stage="sent")
            client.portal.call(store.put_record, record)

        def poll() -> Any:
            state = client.post("/internal/web-chat/state", json=bound, headers=headers)
            assert state.status_code == 200, state.text
            return state.json()

        resolved, lookups = backend.resolved, store.lookups
        first = poll()
        # Both messages finished; the welcome was never processed in this test.
        assert backend.resolved == resolved + 1 and store.lookups == lookups + 3
        second = poll()
        assert backend.resolved == resolved + 1 and store.lookups == lookups + 4
        # The button tap is followed (it settled) but only the text is a bubble.
        assert [(entry["text"], entry["status"]) for entry in second["messages"]] == [("hola", "sent")]
        assert first["messages"] == second["messages"]


def test_each_scan_is_one_start_message_and_new_sessions_are_limited_per_qr() -> None:
    url = os.environ.get("WEB_CHAT_TEST_REDIS_URL")
    if not url:
        pytest.skip("WEB_CHAT_TEST_REDIS_URL not configured")

    class Limiter:
        def __init__(self) -> None:
            self.calls: list[str] = []
            self.allowed = True

        async def allow(self, identity: str) -> bool:
            self.calls.append(identity)
            return self.allowed

    app = FastAPI()
    app.include_router(router)
    cfg = Settings(
        _env_file=None, web_chat_url="http://localhost:5175", redis_url=url, fudia_concierge_api_key=KEY
    )
    limiter = Limiter()
    app.state.settings = cfg
    app.state.whatsapp = WhatsAppProviders(cfg, Registry())
    app.state.fudia = Registry()
    app.state.inbound_queue = MemoryInboundQueue()
    app.state.conversation_store = MemoryConversationStore()
    app.state.web_session_limiter = limiter
    headers = {"X-Fudia-Concierge-Key": KEY}
    payload = {"qr": QR, "visitor": secrets.token_urlsafe(32)}
    with TestClient(app) as client:

        def open_chat(body: dict[str, str]) -> Any:
            return client.post("/internal/web-chat/session", json=body, headers=headers)

        opened = open_chat({**payload, "scan": "scan-reference-1"})
        assert opened.status_code == 200, opened.text
        bound = {**payload, "capability": opened.json()["capability"]}
        start = f"web:{bound['capability']}:start:scan-reference-1"
        queued = app.state.inbound_queue.pending
        assert [entry.message.message_id for entry in queued] == [start]
        # A reload repeats its scan; a new scan is one more start message.
        assert open_chat({**bound, "scan": "scan-reference-1"}).status_code == 200
        assert len(queued) == 1
        assert open_chat({**bound, "scan": "scan-reference-2"}).status_code == 200
        assert len(queued) == 2 and limiter.calls == ["web-session:" + QR]
        # The welcome of the scan that created the session is still pending.
        state = client.post("/internal/web-chat/state", json=bound, headers=headers)
        assert state.status_code == 200 and state.json()["pending"]
        assert open_chat({**payload, "scan": "not a reference"}).status_code == 400
        limiter.allowed = False
        refused = open_chat({"qr": QR, "visitor": secrets.token_urlsafe(32)})
        assert refused.status_code == 429 and len(queued) == 2
        # Sessions that already exist are never cut by the limit.
        assert open_chat({**bound, "scan": "scan-reference-2"}).status_code == 200


def test_historical_gateway_accepts_only_authenticated_partner_actions() -> None:
    import time

    from pydantic import SecretStr

    from src.api.whatsapp_providers import router as gateway_router
    from src.services.delivery_partners import button_payload

    class HistoricalRegistry(Registry):
        async def whatsapp_routes(self):
            return [
                ChannelRoute(
                    routingKey="historical-device",
                    countryCode="PE",
                    provider="WHATSMEOW",
                    providerKey="871e4e24-d031-42fc-82bc-1b7961dcf2cc",
                    phoneNumber="+51987654321",
                )
            ]

    app = FastAPI()
    app.include_router(gateway_router)
    cfg = Settings(_env_file=None, whatsapp_gateway_ingress_key=SecretStr(KEY))
    app.state.settings = cfg
    app.state.whatsapp = WhatsAppProviders(cfg, HistoricalRegistry())
    app.state.inbound_queue = MemoryInboundQueue()
    event = {
        "type": "message",
        "provider": "WHATSMEOW",
        "sessionRef": "871e4e24-d031-42fc-82bc-1b7961dcf2cc",
        "countryCode": "PE",
        "externalId": "old-device-action",
        "sender": "51912345678@s.whatsapp.net",
        "verifiedPhone": "51912345678",
        "text": "Aceptar",
        "timestamp": int(time.time()),
    }
    with TestClient(app) as client:
        headers = {"X-Fudia-Gateway-Key": KEY}
        assert (
            client.post("/internal/whatsapp/gateway/events", json=event, headers=headers).json()["status"]
            == "ignored"
        )
        event["buttonPayload"] = button_payload("accept", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
        result = client.post("/internal/whatsapp/gateway/events", json=event, headers=headers)
        assert result.status_code == 200 and result.json()["accepted"] == 1
        assert app.state.inbound_queue.pending[0].message.partner_action is not None
