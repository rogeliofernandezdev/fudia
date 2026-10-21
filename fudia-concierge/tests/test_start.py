from urllib.parse import parse_qs, quote, urlsplit

from fastapi import FastAPI
from fastapi.testclient import TestClient

from src.api.start import router
from src.domain.models import TableContext
from src.infrastructure.fudia_client import FudiaError
from src.services.ports import LocalTableStartCodes


class FakeFudia:
    def __init__(
        self,
        *,
        enabled: bool = True,
        error: FudiaError | None = None,
    ) -> None:
        self.enabled = enabled
        self.error = error

    async def resolve_table(
        self,
        token: str,
    ) -> TableContext:
        if self.error is not None:
            raise self.error
        return TableContext(
            name="M1",
            seats=4,
            zone="Principal",
            organizationName="Restaurante Demo",
            locationName="Local principal",
            conciergeEnabled=self.enabled,
        )


class FakeWhatsApp:
    def __init__(self) -> None:
        self.tokens: list[str] = []
        self.texts: list[str] = []

    async def start_url(
        self,
        token: str,
        text: str = "",
    ) -> str:
        self.tokens.append(token)
        self.texts.append(text)
        if text:
            return "https://wa.me/51987654321?text=" + quote(text, safe="")
        return (
            "https://wa.me/51987654321"
            "?text=FUDIA%3A"
            + token
        )


class BrokenStartCodes:
    async def issue(self, qr_token: str) -> str:
        raise ConnectionError("redis down")

    async def resolve(self, code: str) -> str | None:
        return None


def app_with(
    fudia: FakeFudia,
    whatsapp: FakeWhatsApp,
    start_codes: object | None = None,
) -> FastAPI:
    app = FastAPI()
    app.state.fudia = fudia
    app.state.whatsapp = whatsapp
    if start_codes is not None:
        app.state.start_codes = start_codes
    app.include_router(router)
    return app


def test_start_redirects_only_after_qr_entitlement_validation() -> None:
    whatsapp = FakeWhatsApp()
    app = app_with(
        FakeFudia(enabled=True),
        whatsapp,
    )
    client = TestClient(
        app,
        follow_redirects=False,
    )

    response = client.get(
        "/start/" + "a" * 32
    )

    assert response.status_code == 307
    assert response.headers["location"] == (
        "https://wa.me/51987654321"
        "?text=FUDIA%3A"
        + "a" * 32
    )
    assert whatsapp.tokens == ["a" * 32]


def test_start_blocks_company_without_concierge_module() -> None:
    whatsapp = FakeWhatsApp()
    app = app_with(
        FakeFudia(enabled=False),
        whatsapp,
    )
    client = TestClient(
        app,
        follow_redirects=False,
    )

    response = client.get(
        "/start/" + "a" * 32
    )

    assert response.status_code == 404
    assert whatsapp.tokens == []


def test_start_returns_service_unavailable_when_backend_cannot_validate_qr() -> None:
    whatsapp = FakeWhatsApp()
    app = app_with(
        FakeFudia(
            error=FudiaError(
                503,
                "fudia_unreachable",
                "unavailable",
            )
        ),
        whatsapp,
    )
    client = TestClient(
        app,
        follow_redirects=False,
    )

    response = client.get(
        "/start/" + "a" * 32
    )

    assert response.status_code == 503
    assert whatsapp.tokens == []


def test_start_message_is_friendly_with_a_short_table_code() -> None:
    whatsapp = FakeWhatsApp()
    codes = LocalTableStartCodes()
    client = TestClient(
        app_with(FakeFudia(enabled=True), whatsapp, codes),
        follow_redirects=False,
    )

    response = client.get("/start/" + "a" * 32)

    assert response.status_code == 307
    text = parse_qs(urlsplit(response.headers["location"]).query)["text"][0]
    first, second = text.split("\n")
    assert first == "¡Hola! 👋 Quiero ver la carta."
    label, code = second.split(" · ")
    assert label == "Mesa M1"
    assert "FUDIA" not in text and "a" * 32 not in text
    assert codes.codes[code][0] == "a" * 32


def test_start_falls_back_to_legacy_message_when_codes_are_unavailable() -> None:
    whatsapp = FakeWhatsApp()
    client = TestClient(
        app_with(FakeFudia(enabled=True), whatsapp, BrokenStartCodes()),
        follow_redirects=False,
    )

    response = client.get("/start/" + "a" * 32)

    assert response.status_code == 307
    assert response.headers["location"].endswith("FUDIA%3A" + "a" * 32)
    assert whatsapp.texts == [""]
