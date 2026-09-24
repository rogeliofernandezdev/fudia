import httpx
import pytest

from src.infrastructure.whatsapp_adapter import MetaWhatsAppAdapter


@pytest.mark.asyncio
async def test_send_text_uses_inbound_sender_phone_id() -> None:
    requested_urls: list[str] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        requested_urls.append(str(request.url))
        return httpx.Response(200, json={"messages": [{"id": "wamid.1"}]})

    client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    adapter = MetaWhatsAppAdapter(
        token="token",
        phone_id="global-phone-id",
        graph_version="v99.0",
        client=client,
    )

    await adapter.send_text(
        "51999999999",
        "Hola",
        sender_phone_id="inbound-phone-id",
    )

    assert requested_urls == [
        "https://graph.facebook.com/v99.0/inbound-phone-id/messages"
    ]
    await client.aclose()


@pytest.mark.asyncio
async def test_send_text_falls_back_to_configured_phone_id() -> None:
    requested_urls: list[str] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        requested_urls.append(str(request.url))
        return httpx.Response(200, json={"messages": [{"id": "wamid.2"}]})

    client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    adapter = MetaWhatsAppAdapter(
        token="token",
        phone_id="global-phone-id",
        graph_version="v99.0",
        client=client,
    )

    await adapter.send_text("51999999999", "Hola")

    assert requested_urls == [
        "https://graph.facebook.com/v99.0/global-phone-id/messages"
    ]
    await client.aclose()
