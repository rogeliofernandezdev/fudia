"""The full carta sends its virtual menu pages whichever agent asked for it."""

from __future__ import annotations

from decimal import Decimal

import pytest

from src.domain.intents import AgentIntent
from src.domain.models import ConversationSession, MenuItem, MenuResponse
from src.services.tool_registry import ToolRegistry
from tests.test_concierge_service import FakeFudia

PAGES = ["https://storage.example/carta-1.jpg", "https://storage.example/carta-2.jpg"]


class CartaFudia(FakeFudia):
    async def search_menu(self, token: str, query: str = "", product_id: str = "") -> MenuResponse:
        item = MenuItem(
            productId="p1",
            name="Lomo saltado",
            description="",
            price=Decimal("32.50"),
            categoryName="Platos",
            status="available",
            isCombo=False,
        )
        return MenuResponse(items=[item], currencySymbol="S/", virtualMenuUrls=PAGES)


@pytest.mark.parametrize("intent", ["menu", "order"])
async def test_full_carta_pages_reach_the_reply(intent: AgentIntent) -> None:
    session = ConversationSession(phone="51999999999", qr_token="a" * 32)
    registry = ToolRegistry(CartaFudia(), session, "quiero ver la carta")
    result = await registry.execute_for(intent, "search_menu", {"query": ""})
    assert result["ok"] is True
    # ConciergeService attaches registry.menu.media to the reply record.
    assert [image.url for image in registry.menu.media] == PAGES
