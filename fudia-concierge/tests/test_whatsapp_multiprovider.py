from __future__ import annotations

import time
from typing import Any

import httpx
import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient
from pydantic import SecretStr

from config.settings import Settings
from src.api.whatsapp_providers import router
from src.domain.models import CartLine, InboundMessage, TableContext
from src.domain.whatsapp import ChannelRoute, RestaurantOption, RestaurantPage
from src.infrastructure.queue import MemoryInboundQueue
from src.infrastructure.state_store import MemoryConversationStore
from src.infrastructure.whatsapp_providers import GatewayWhatsAppAdapter, WhatsAppProviders
from src.services.concierge_service import ConciergeService
from tests.test_concierge_service import FakeFudia

REF = "871e4e24-d031-42fc-82bc-1b7961dcf2cc"
PHONE = "51912345678"
KEY = "test-private-gateway-key-32-characters"


class Registry:
    def __init__(self) -> None:
        self.routes = [
            ChannelRoute(
                routingKey="original",
                countryCode="PE",
                providerKey="12345",
                phoneNumber="+51911111111",
                active=True,
            )
        ]

    async def whatsapp_routes(self) -> list[ChannelRoute]:
        return self.routes


def config(**kwargs: Any) -> Settings:
    return Settings(
        _env_file=None,
        whatsapp_phone_id="12345",
        whatsapp_token="private-token",
        whatsapp_graph_version="v25.0",
        **kwargs,
    )


@pytest.mark.asyncio
async def test_printed_token_resolves_current_country_number_without_replacement() -> None:
    registry = Registry()
    phones = {"12345": "51911111111", "67890": "51922222222"}

    async def graph(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"display_phone_number": phones[request.url.path.rsplit("/", 1)[-1]]})

    providers = WhatsAppProviders(
        config(
            whatsapp_meta_accounts=SecretStr(
                '[{"countryCode":"PE","phoneId":"67890","token":"second-secret"}]'
            )
        ),
        registry,
        meta_client=httpx.AsyncClient(transport=httpx.MockTransport(graph)),
    )
    before = await providers.start_url_for_country("PE", "a" * 32, "Mesa 5 · K7P2")
    registry.routes[0] = registry.routes[0].model_copy(update={"active": False})
    registry.routes.append(
        ChannelRoute(
            routingKey="new", countryCode="PE", providerKey="67890", phoneNumber="+51922222222", active=True
        )
    )
    after = await providers.start_url_for_country("PE", "a" * 32, "Mesa 5 · K7P2")
    assert before.startswith("https://wa.me/51911111111?")
    assert after.startswith("https://wa.me/51922222222?")
    original = await providers.for_channel("original")
    assert original.phone_id == "12345"
    with pytest.raises(RuntimeError):
        await providers.start_url_for_country("CL", "a" * 32)
    phones["12345"] = "51933333333"
    with pytest.raises(RuntimeError, match="cambió"):
        await providers.for_channel("original")
    await providers.close()


@pytest.mark.asyncio
async def test_meta_activation_checks_authorized_actual_registered_asset() -> None:
    registered = True

    async def graph(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            json={
                "id": "12345",
                "display_phone_number": "+51 911 111 111",
                "code_verification_status": "VERIFIED",
                "platform_type": "CLOUD_API",
                "status": "CONNECTED" if registered else "DISCONNECTED",
            },
        )

    providers = WhatsAppProviders(
        config(), Registry(), meta_client=httpx.AsyncClient(transport=httpx.MockTransport(graph))
    )
    assert (await providers.verify("PE", "META_CLOUD_API", "+51911111111"))["state"] == "connected"
    assert (await providers.verify("PE", "META_CLOUD_API", "+51922222222"))["state"] != "connected"
    assert (await providers.verify("CL", "META_CLOUD_API", "+51911111111"))["state"] != "connected"
    registered = False
    assert (await providers.verify("PE", "META_CLOUD_API", "+51911111111"))["state"] != "connected"
    await providers.close()


def gateway_app() -> tuple[FastAPI, MemoryInboundQueue]:
    backend = Registry()
    backend.routes = [
        ChannelRoute(
            routingKey="gateway-original",
            countryCode="PE",
            provider="WHATSMEOW",
            providerKey=REF,
            phoneNumber="+51911111111",
            active=True,
        )
    ]
    providers = WhatsAppProviders(config(whatsapp_gateway_ingress_key=SecretStr(KEY)), backend)
    app = FastAPI()
    app.include_router(router)
    app.state.settings, app.state.whatsapp = providers.cfg, providers
    queue = MemoryInboundQueue()
    app.state.inbound_queue = queue
    return app, queue


def event(**kwargs: Any) -> dict[str, Any]:
    return {
        "type": "message",
        "provider": "WHATSMEOW",
        "sessionRef": REF,
        "countryCode": "PE",
        "externalId": "external-1",
        "sender": "123456789012345@lid",
        "text": "hola",
        "timestamp": int(time.time()),
        **kwargs,
    }


def test_gateway_auth_validation_location_and_namespaced_dedup() -> None:
    app, queue = gateway_app()
    with TestClient(app) as client:
        path = "/internal/whatsapp/gateway/events"
        assert client.post(path, json=event()).status_code == 401
        headers = {"X-Fudia-Gateway-Key": KEY}
        assert client.post(path, json=event(countryCode="CL"), headers=headers).json()["status"] == "ignored"
        assert client.post(path, json=event(sender="123@g.us"), headers=headers).status_code == 400
        assert (
            client.post(path, json=event(sender=PHONE + "@s.whatsapp.net"), headers=headers).status_code
            == 400
        )
        assert (
            client.post(path, json=event(timestamp=int(time.time()) + 1000), headers=headers).status_code
            == 400
        )
        first = client.post(
            path, json=event(text="", location={"latitude": -12.1, "longitude": -77.1}), headers=headers
        )
        assert first.json()["accepted"] == 1
        assert client.post(path, json=event(), headers=headers).json()["accepted"] == 0
    message = queue.pending[0].message
    assert message.message_id == f"WHATSMEOW:{REF}:external-1"
    assert message.sender_phone_id == "gateway-original"
    assert message.identity().verifiedPhone == ""
    assert message.phone == "123456789012345@lid"
    assert message.location and message.location.latitude == -12.1


@pytest.mark.asyncio
async def test_legacy_meta_queue_alias_prevents_upgrade_duplicate() -> None:
    queue = MemoryInboundQueue()
    assert await queue.enqueue(InboundMessage(message_id="wamid.old", phone=PHONE, text="confirmo"))
    assert not await queue.enqueue(
        InboundMessage(
            message_id="META_CLOUD_API:12345:wamid.old",
            legacy_message_id="wamid.old",
            phone=PHONE,
            text="confirmo",
        )
    )
    assert len(queue.pending) == 1


class DirectoryFudia(FakeFudia):
    def __init__(self, visits: int) -> None:
        super().__init__()
        self.options = [
            RestaurantOption(
                organizationId=f"org-{i}",
                locationId=f"loc-{i}",
                organizationName=f"Restaurante {i}",
                locationName="Principal",
                visitCount=1 if visits else 0,
                qrToken=str(i) * 32,
            )
            for i in (1, 2)
        ]
        self.visits = visits
        self.origins: list[dict[str, Any]] = []
        self.disabled = False

    async def restaurants(
        self, channel: str, identity: Any, only_visited: bool, offset: int
    ) -> RestaurantPage:
        assert channel == "original" and identity.verifiedPhone == PHONE
        return RestaurantPage(items=self.options[: self.visits] if only_visited else self.options)

    async def restaurant_context(self, channel: str, location: str) -> RestaurantOption:
        assert not self.disabled
        return next(item for item in self.options if item.locationId == location)

    async def resolve_table(self, token: str) -> TableContext:
        selected = next(item for item in self.options if item.qrToken == token)
        return TableContext(
            name="Delivery",
            seats=0,
            serviceType="delivery",
            conciergeEnabled=True,
            countryCode="PE",
            organizationId=selected.organizationId,
            locationId=selected.locationId,
            organizationName=selected.organizationName,
            locationName=selected.locationName,
        )

    async def open_conversation(
        self, token: str, conversation_id: str, kind: str, origin: dict[str, Any] | None = None
    ) -> None:
        self.origins.append(origin or {})
        await super().open_conversation(token, conversation_id, kind)


class NoAI:
    async def reply(self, *_: Any) -> str:
        raise AssertionError("Restaurant selection must not invoke AI")


async def turn(service: ConciergeService, index: int, text: str, **kwargs: Any):
    return await service.process_inbound(
        InboundMessage(
            message_id=f"turn-{index}",
            phone=PHONE,
            text=text,
            sender_phone_id="original",
            sent_at=1000 + index,
            **kwargs,
        )
    )


@pytest.mark.asyncio
@pytest.mark.parametrize("visits", [0, 1, 2])
async def test_restaurant_discovery_single_known_and_multiple_selector(visits: int) -> None:
    backend, store = DirectoryFudia(visits), MemoryConversationStore()
    service = ConciergeService(store, backend, NoAI(), directory=backend)
    result = await turn(service, 1, "hola")
    session = await store.get(PHONE, "original")
    assert session
    if visits == 1:
        assert "Hola de nuevo" in result.reply and session.venue.organizationId == "org-1"
    else:
        assert session.selection.state == "AWAITING_RESTAURANT_SELECTION"
        assert session.selection.options[1].locationId == "loc-2"
        invalid = await turn(service, 2, "99")
        assert "Elige un número" in invalid.reply and not backend.opened
        selected = await turn(service, 3, "2")
        assert "Restaurante 2" in selected.reply
        session = await store.get(PHONE, "original")
        assert session.selection is None and session.venue.organizationId == "org-2"
        assert session.qr_token == "2" * 32 and session.table is None
        assert backend.origins[-1]["channelKey"] == "original"
        duplicate = await turn(service, 3, "2")
        assert duplicate.reply == selected.reply and len(backend.opened) == 1


@pytest.mark.asyncio
async def test_restaurant_change_clears_temporary_cart_and_fences_delayed_message() -> None:
    backend, store = DirectoryFudia(2), MemoryConversationStore()
    service = ConciergeService(store, backend, NoAI(), directory=backend)
    await turn(service, 1, "hola")
    await turn(service, 2, "1")
    old = await store.get(PHONE, "original")
    old.cart = [CartLine(product_id="p1", name="Lomo", quantity=1, unit_price="20")]
    await store.save(old)
    await turn(service, 3, "restaurantes")
    await turn(service, 4, "2")
    current = await store.get(PHONE, "original")
    assert current.conversation_id != old.conversation_id and current.cart == []
    stale = await service.process_inbound(
        InboundMessage(
            message_id="delayed", phone=PHONE, text="confirmo", sender_phone_id="original", sent_at=1001
        )
    )
    assert stale.stage == "superseded" and not backend.order_calls
    assert (await store.get(PHONE, "original")).venue.organizationId == "org-2"


@pytest.mark.asyncio
async def test_gateway_outbound_keeps_original_session_and_stable_callback() -> None:
    class Client:
        async def request(self, method, path, payload):
            self.payload = payload
            return {"state": "sent"}

    client = Client()
    adapter = GatewayWhatsAppAdapter(client, REF)
    await adapter.send_text("123@lid", "hola", callback_ref="a" * 64)
    assert client.payload["sessionRef"] == REF and client.payload["recipient"] == "123@lid"
    assert client.payload["callbackRef"] == "a" * 64
    assert adapter.supports_templates is False


@pytest.mark.asyncio
async def test_namespaced_ids_do_not_collide_across_meta_assets() -> None:
    queue = MemoryInboundQueue()
    for asset in ("12345", "67890"):
        assert await queue.enqueue(
            InboundMessage(
                message_id=f"META_CLOUD_API:{asset}:same-id",
                legacy_message_id="same-id",
                phone=PHONE,
                text="hola",
                sender_phone_id=asset,
            )
        )
    assert len(queue.pending) == 2


@pytest.mark.asyncio
async def test_live_table_stays_and_three_hour_idle_enters_known_restaurant() -> None:
    from src.domain.models import ConversationSession

    backend, store = DirectoryFudia(1), MemoryConversationStore()
    service = ConciergeService(store, backend, NoAI(), directory=backend, clock=lambda: 20_000)
    table = TableContext(
        name="Mesa 5",
        seats=4,
        organizationId="org-1",
        locationId="loc-1",
        organizationName="Restaurante 1",
        locationName="Principal",
        conciergeEnabled=True,
    )
    session = ConversationSession(
        phone=PHONE, channel_key="original", qr_token="1" * 32, table=table, last_activity_at=19_000
    )
    await store.save(session)
    active = await turn(service, 1, "cambiar restaurante")
    assert "sigue vigente" in active.reply and not backend.opened
    session = await store.get(PHONE, "original")
    session.last_activity_at = 20_000 - 10_801
    await store.save(session)
    ended = await turn(service, 2, "hola")
    assert "Hola de nuevo" in ended.reply
    current = await store.get(PHONE, "original")
    assert current.table is None and current.mode == "delivery"


@pytest.mark.asyncio
async def test_verified_legacy_import_uses_actual_meta_asset_only() -> None:
    class Backend(Registry):
        def __init__(self):
            super().__init__()
            self.imported = []

        async def register_legacy_whatsapp_route(self, route):
            self.imported.append(route)

    backend = Backend()

    async def graph(request):
        return httpx.Response(
            200,
            json={
                "id": "12345",
                "display_phone_number": "+51911111111",
                "code_verification_status": "VERIFIED",
                "platform_type": "CLOUD_API",
                "status": "CONNECTED",
            },
        )

    providers = WhatsAppProviders(
        config(), backend, meta_client=httpx.AsyncClient(transport=httpx.MockTransport(graph))
    )
    await providers.initialize()
    assert len(backend.imported) == 1 and backend.imported[0].routingKey == "12345"
    assert backend.imported[0].phoneNumber == "+51911111111"
    await providers.close()


@pytest.mark.asyncio
async def test_legacy_customer_memory_is_revalidated_before_persistent_visit() -> None:
    backend, store = DirectoryFudia(0), MemoryConversationStore()
    legacy = ConciergeService(store, backend, NoAI())
    await legacy.process_inbound(
        InboundMessage(
            message_id="legacy-qr", phone=PHONE, text="FUDIA:" + "1" * 32, sender_phone_id="original"
        )
    )
    assert await store.get_memory(PHONE, "original") is not None
    store.items.clear()
    backend.opened.clear()
    current = ConciergeService(store, backend, NoAI(), directory=backend)
    reply = await turn(current, 1, "hola", country_code="PE")
    assert "Hola de nuevo" in reply.reply
    session = await store.get(PHONE, "original")
    assert session.selection is None and session.venue.organizationId == "org-1"
    assert backend.origins[-1]["identity"]["verifiedPhone"] == PHONE
