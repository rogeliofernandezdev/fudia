from __future__ import annotations

from decimal import Decimal
from types import SimpleNamespace
from unittest.mock import AsyncMock

import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from src.api import web_chat
from src.domain.models import ConversationSession, OrderStatus, TableContext
from src.services.ports import FudiaError
from src.services.tools import ConciergeTools
from src.services.web_chat import action_reply
from tests.test_concierge_service import FakeFudia


def test_bill_shortcut_refreshes_delivery_from_backend_and_fails_closed(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    session = ConversationSession(phone="browser-session", qr_token="a" * 32, last_order_id="o1")
    order = OrderStatus(
        orderId="o1", code="PED-1", channel="salon", status="listo", total=Decimal("25"), active=True
    )
    backend = SimpleNamespace(get_order_status=AsyncMock(return_value=order))
    providers = SimpleNamespace(
        web=SimpleNamespace(messages=AsyncMock(return_value=[])),
        routes=AsyncMock(return_value=[SimpleNamespace(active=True, routingKey="web-route")]),
    )
    monkeypatch.setattr(web_chat, "context", AsyncMock(return_value={"channel": "web-route"}))
    monkeypatch.setattr(
        web_chat,
        "polled_table",
        AsyncMock(
            return_value=TableContext(
                tableId="table",
                organizationId="org",
                locationId="local",
                name="Mesa 1",
                conciergeEnabled=True,
                seats=4,
                organizationName="Restaurante de prueba",
                locationName="Local principal",
            )
        ),
    )
    monkeypatch.setattr(web_chat, "manager", lambda _: providers)
    app = FastAPI()
    app.include_router(web_chat.router)
    app.state.fudia = backend
    app.state.conversation_store = SimpleNamespace(get=AsyncMock(return_value=session))
    app.state.settings = SimpleNamespace(
        fudia_concierge_api_key=SimpleNamespace(get_secret_value=lambda: "test-key")
    )
    # The authenticated body is orthogonal to this regression; its existing
    # isolation tests cover visitor/QR bindings and the private service key.
    monkeypatch.setattr(
        web_chat,
        "body",
        AsyncMock(return_value=web_chat.WebRequest(qr="a" * 32, visitor="v" * 43, capability="cap")),
    )
    with TestClient(app) as client:

        def poll() -> bool:
            response = client.post("/internal/web-chat/state")
            assert response.status_code == 200, response.text
            return bool(response.json()["canRequestBill"])

        assert not poll()  # Ready at kitchen is still not delivered.
        order.canRequestBill = True
        order.status = "entregado"
        assert poll()  # The waiter records delivery without a chat message.
        order.canRequestBill = False
        order.status = "nuevo"
        assert not poll()  # A new round hides it again.
        backend.get_order_status.side_effect = FudiaError(503, "unavailable", "No disponible")
        assert not poll()
        assert backend.get_order_status.await_count == 4


@pytest.mark.asyncio
async def test_forged_bill_button_returns_delivery_message_without_claiming_staff_were_notified() -> None:
    class UndeliveredFudia(FakeFudia):
        async def request_bill(self, token: str, phone: str, conversation_id: str, request_id: str = ""):
            raise FudiaError(
                409,
                "products_not_delivered",
                "Podrás pedir la cuenta cuando el mozo haya entregado todos los productos de tu pedido.",
            )

    backend = UndeliveredFudia()
    session = ConversationSession(phone="browser-session", qr_token="a" * 32)
    result = await ConciergeTools(backend, session, "La cuenta, por favor").execute("request_bill", {})
    assert result["code"] == "products_not_delivered" and not result["ok"]
    reply = action_reply("request_bill", result, "S/")
    assert "mozo haya entregado" in reply
    assert "Total:" not in reply and "recibió tu solicitud" not in reply
