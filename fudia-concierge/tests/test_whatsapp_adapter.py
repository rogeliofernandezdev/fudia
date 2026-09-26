import httpx
import pytest

from src.infrastructure.whatsapp_adapter import MetaWhatsAppAdapter


@pytest.mark.asyncio
async def test_send_text_uses_global_phone_id() -> None:
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
    )

    assert requested_urls == [
        "https://graph.facebook.com/v99.0/global-phone-id/messages"
    ]
    await client.aclose()


@pytest.mark.asyncio
async def test_start_url_resolves_display_number_from_meta_and_caches_it() -> None:
    requested_urls: list[str] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        requested_urls.append(str(request.url))
        assert request.headers["Authorization"] == "Bearer token"
        assert request.url.params["fields"] == "display_phone_number"
        return httpx.Response(
            200,
            json={
                "id": "global-phone-id",
                "display_phone_number": "+51 987 654 321",
            },
        )

    client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    adapter = MetaWhatsAppAdapter(
        token="token",
        phone_id="global-phone-id",
        graph_version="v99.0",
        client=client,
    )

    first = await adapter.start_url("a" * 32)
    second = await adapter.start_url("b" * 32)

    assert first == (
        "https://wa.me/51987654321"
        "?text=FUDIA%3A" + "a" * 32
    )
    assert second == (
        "https://wa.me/51987654321"
        "?text=FUDIA%3A" + "b" * 32
    )
    assert len(requested_urls) == 1
    assert requested_urls[0].startswith(
        "https://graph.facebook.com/v99.0/global-phone-id"
    )
    await client.aclose()


@pytest.mark.asyncio
async def test_start_url_rejects_invalid_meta_display_number() -> None:
    async def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            json={"display_phone_number": "invalid"},
        )

    client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    adapter = MetaWhatsAppAdapter(
        token="token",
        phone_id="global-phone-id",
        graph_version="v99.0",
        client=client,
    )

    with pytest.raises(
        RuntimeError,
        match="número de WhatsApp válido",
    ):
        await adapter.start_url("a" * 32)

    await client.aclose()
