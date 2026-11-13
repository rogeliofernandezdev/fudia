from decimal import Decimal

import pytest
from fastapi import FastAPI

from src.api.web_chat import order_summary, router
from src.domain.models import CartLine, ConversationSession, GeoPoint


def session() -> ConversationSession:
    return ConversationSession(
        phone="browser-session", provider="FUDIA_WEB", mode="delivery", currency_symbol="S/",
        cart=[CartLine(product_id="p1", name="Lomo saltado", quantity=1, unit_price=Decimal("20.00"))],
    )


@pytest.mark.parametrize("fee", ["8.00", "0.00"])
def test_summary_includes_the_backend_quote_and_total(fee: str) -> None:
    current = session()
    current.delivery.fulfillment = "delivery"
    current.delivery.location = GeoPoint(latitude=-12.12, longitude=-77.03)
    current.delivery.covered = True
    current.delivery.fee = Decimal(fee)
    summary = order_summary(current, "delivery")
    assert summary.subtotal == "20.00" and summary.deliveryFee == fee
    assert Decimal(summary.total or "0") == Decimal("20.00") + Decimal(fee)
    assert summary.fulfillment == "delivery" and summary.currencySymbol == "S/"


@pytest.mark.parametrize("covered", [False, True])
def test_unquoted_or_out_of_range_delivery_does_not_look_free(covered: bool) -> None:
    current = session()
    current.delivery.fulfillment = "delivery"
    current.delivery.location = GeoPoint(latitude=-12.12, longitude=-77.03)
    current.delivery.covered = covered
    current.delivery.fee = Decimal("8.00") if not covered else None
    summary = order_summary(current, "delivery")
    assert summary.deliveryFee is None and summary.total is None


def test_pickup_and_table_totals_do_not_include_a_previous_delivery_fee() -> None:
    current = session()
    current.delivery.fulfillment = "pickup"
    current.delivery.fee = Decimal("8.00")
    for mode in ("delivery", "table", "counter"):
        current.mode = mode  # type: ignore[assignment]
        summary = order_summary(current, mode)
        assert summary.subtotal == summary.total == "20.00" and summary.deliveryFee == "0.00"


def test_summary_changes_with_the_cart_and_never_reuses_its_previous_total() -> None:
    current = session()
    current.delivery.fulfillment = "pickup"
    assert order_summary(current, "delivery").total == "20.00"
    current.cart[0].quantity = 2
    assert order_summary(current, "delivery").total == "40.00"
    current.cart = []
    assert order_summary(current, "delivery").subtotal == "0.00"


def test_initial_delivery_state_does_not_invent_a_quote() -> None:
    assert order_summary(None, "delivery").deliveryFee is None
    assert order_summary(None, "delivery").total is None


def test_state_openapi_exposes_the_order_summary_contract() -> None:
    app = FastAPI()
    app.include_router(router)
    schema = app.openapi()
    response = schema["paths"]["/internal/web-chat/state"]["post"]["responses"]["200"]
    assert response["content"]["application/json"]["schema"]["$ref"].endswith("/WebChatState")
    fields = schema["components"]["schemas"]["WebOrderSummary"]["properties"]
    assert {"subtotal", "deliveryFee", "total", "currencySymbol", "fulfillment"} <= fields.keys()
