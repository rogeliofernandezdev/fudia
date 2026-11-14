from __future__ import annotations

from decimal import Decimal
from types import SimpleNamespace
from unittest.mock import AsyncMock

import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from src.api import web_chat
from src.domain.models import ConversationSession, DeliveryTracking, OrderStatus, TrackingPoint
from src.services.ports import FudiaError


def tracking_app(
    monkeypatch: pytest.MonkeyPatch, session: ConversationSession | None, backend: SimpleNamespace
) -> FastAPI:
    monkeypatch.setattr(web_chat, "context", AsyncMock(return_value={"channel": "web-route"}))
    monkeypatch.setattr(
        web_chat,
        "body",
        AsyncMock(return_value=web_chat.WebRequest(qr="a" * 32, visitor="v" * 43, capability="cap")),
    )
    app = FastAPI()
    app.include_router(web_chat.router)
    app.state.fudia = backend
    app.state.conversation_store = SimpleNamespace(get=AsyncMock(return_value=session))
    return app


def test_tracking_reads_only_the_order_bound_to_the_session(monkeypatch: pytest.MonkeyPatch) -> None:
    session = ConversationSession(
        phone="browser-session", qr_token="a" * 32, mode="delivery", last_order_id="o1",
        conversation_id="conversation-1",
    )
    order = OrderStatus(
        orderId="o1", code="PED-7", channel="delivery", status="en_camino", total=Decimal("65"), active=True,
        delivery=DeliveryTracking(
            dispatchStatus="picked_up", partnerName="Reparto Norte", originName="Local principal",
            origin=TrackingPoint(latitude=-12.04, longitude=-77.04),
            destination=TrackingPoint(latitude=-12.12, longitude=-77.03),
            courier=TrackingPoint(latitude=-12.08, longitude=-77.035),
            courierLocatedAt="2026-11-14T10:00:00Z",
        ),
    )
    backend = SimpleNamespace(get_order_status=AsyncMock(return_value=order))
    with TestClient(tracking_app(monkeypatch, session, backend)) as client:
        # A forged order id in the body is ignored: the session decides the order.
        response = client.post("/internal/web-chat/tracking", json={"orderId": "another-order"})
        assert response.status_code == 200, response.text
        data = response.json()
        assert data["available"] is True
        assert data["code"] == "PED-7" and data["status"] == "en_camino"
        assert data["delivery"]["courier"] == {"latitude": -12.08, "longitude": -77.035}
        assert data["delivery"]["partnerName"] == "Reparto Norte"
        backend.get_order_status.assert_awaited_once_with("a" * 32, "o1", "conversation-1")

        # A pickup order has a status but no map.
        order.channel, order.delivery = "recojo", None
        data = client.post("/internal/web-chat/tracking").json()
        assert data["available"] is True and data["delivery"] is None

        backend.get_order_status.side_effect = FudiaError(503, "unavailable", "No disponible")
        assert client.post("/internal/web-chat/tracking").status_code == 503


@pytest.mark.parametrize(
    "session",
    [
        None,
        ConversationSession(phone="browser-session", qr_token="a" * 32, mode="delivery"),
        ConversationSession(phone="browser-session", qr_token="a" * 32, mode="table", last_order_id="o1"),
    ],
)
def test_tracking_is_unavailable_without_a_delivery_order(
    monkeypatch: pytest.MonkeyPatch, session: ConversationSession | None
) -> None:
    backend = SimpleNamespace(get_order_status=AsyncMock())
    with TestClient(tracking_app(monkeypatch, session, backend)) as client:
        response = client.post("/internal/web-chat/tracking")
        assert response.status_code == 200, response.text
        assert response.json() == {
            "available": False, "code": "", "status": "", "channel": "", "delivery": None,
        }
    backend.get_order_status.assert_not_awaited()
    assert web_chat.trackable(session) is False
