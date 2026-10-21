from decimal import Decimal

import pytest

from src.domain.models import ConversationSession, MenuItem
from src.services.tools import ConciergeTools
from tests.test_concierge_service import FakeFudia


class StockFudia(FakeFudia):
    """Products with today's remaining units, looked up by id."""

    def __init__(self, *items: MenuItem) -> None:
        super().__init__()
        self.products = {item.productId: item for item in items}

    async def get_product(self, token: str, product_id: str) -> MenuItem | None:
        return self.products.get(product_id)


def water(remaining: str | None = "40") -> MenuItem:
    return MenuItem(
        productId="water",
        name="Agua mineral San Luis",
        price=Decimal("2"),
        remaining=None if remaining is None else Decimal(remaining),
    )


def session() -> ConversationSession:
    return ConversationSession(phone="51999999999", qr_token="a" * 32)


@pytest.mark.asyncio
async def test_more_than_the_remaining_units_is_refused_with_what_is_left() -> None:
    state = session()
    tools = ConciergeTools(StockFudia(water()), state, "100 aguas")

    result = await tools.execute("add_item", {"productId": "water", "quantity": 100})

    assert result == {
        "ok": False,
        "code": "insufficient_stock",
        "name": "Agua mineral San Luis",
        "requested": 100,
        "available": 40,
        "inCart": 0,
    }
    assert state.cart == []


@pytest.mark.asyncio
async def test_units_already_in_the_cart_count_against_what_is_left() -> None:
    state = session()
    fudia = StockFudia(water())

    first = await ConciergeTools(fudia, state, "30 aguas").execute(
        "add_item", {"productId": "water", "quantity": 30}
    )
    more = await ConciergeTools(fudia, state, "15 más").execute(
        "add_item", {"productId": "water", "quantity": 15}
    )
    rest = await ConciergeTools(fudia, state, "10 más").execute(
        "add_item", {"productId": "water", "quantity": 10}
    )

    assert first["ok"] is True
    assert (more["code"], more["available"], more["inCart"]) == ("insufficient_stock", 10, 30)
    assert rest["ok"] is True
    assert state.cart[0].quantity == 40


@pytest.mark.asyncio
async def test_products_without_quantity_control_have_no_cap() -> None:
    state = session()
    tools = ConciergeTools(StockFudia(water(remaining=None)), state, "100 aguas")

    result = await tools.execute("add_item", {"productId": "water", "quantity": 100})

    assert result["ok"] is True
    assert state.cart[0].quantity == 100


@pytest.mark.asyncio
async def test_a_combo_dish_draws_from_the_same_remaining_portions() -> None:
    state = session()
    fudia = StockFudia(
        MenuItem(productId="combo1", name="Combo criollo", price=Decimal("20"), isCombo=True),
        MenuItem(productId="opt1", name="Papas", price=Decimal("6"), remaining=Decimal("2")),
    )
    selection = {"selections": [{"groupId": "g1", "productId": "opt1"}]}

    two = await ConciergeTools(fudia, state, "dos combos").execute(
        "add_combo_item", {"productId": "combo1", "quantity": 2, **selection}
    )
    third = await ConciergeTools(fudia, state, "uno más").execute(
        "add_combo_item", {"productId": "combo1", "quantity": 1, **selection}
    )

    assert two["ok"] is True
    assert (third["code"], third["name"], third["available"]) == ("insufficient_stock", "Papas", 0)
    assert len(state.cart) == 1
