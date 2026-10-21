from fastapi import FastAPI
from fastapi.testclient import TestClient

from src.api.start import router
from src.domain.models import TableContext
from src.infrastructure.fudia_client import FudiaError


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

    async def start_url(
        self,
        token: str,
    ) -> str:
        self.tokens.append(token)
        return (
            "https://wa.me/51987654321"
            "?text=FUDIA%3A"
            + token
        )


def app_with(
    fudia: FakeFudia,
    whatsapp: FakeWhatsApp,
) -> FastAPI:
    app = FastAPI()
    app.state.fudia = fudia
    app.state.whatsapp = whatsapp
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
