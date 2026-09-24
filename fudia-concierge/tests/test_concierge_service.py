from decimal import Decimal
from typing import Any

import pytest

from src.domain.models import (
    ComboDetail,
    ComboGroup,
    ComboOption,
    ConversationSession,
    MenuItem,
    MenuResponse,
    OrderResult,
    TableContext,
)
from src.infrastructure.state_store import MemoryConversationStore
from src.services.concierge_service import ConciergeService, extract_qr_token
from src.services.tools import ConciergeTools, explicit_confirmation


class FakeFudia:
    def __init__(self, concierge_enabled: bool = True) -> None:
        self.order_calls: list[dict[str, Any]] = []
        self.concierge_enabled = concierge_enabled

    async def resolve_table(self, token: str) -> TableContext:
        assert token == "a" * 32
        return TableContext(
            name="M1",
            seats=4,
            zone="Principal",
            organizationName="Restaurante Demo",
            locationName="Local principal",
            conciergeEnabled=self.concierge_enabled,
            whatsappPhone="+51987654321" if self.concierge_enabled else "",
        )

    async def search_menu(
        self, token: str, query: str = "", product_id: str = ""
    ) -> MenuResponse:
        item = MenuItem(
            productId="p1",
            name="Lomo saltado",
            description="",
            price=Decimal("32.50"),
            categoryName="Platos",
            status="available",
            isCombo=False,
        )
        if product_id and product_id != "p1":
            return MenuResponse(items=[], currencySymbol="S/")
        return MenuResponse(items=[item], currencySymbol="S/")

    async def get_product(self, token: str, product_id: str) -> MenuItem | None:
        response = await self.search_menu(token, product_id=product_id)
        return response.items[0] if response.items else None

    async def get_combo(self, token: str, product_id: str) -> ComboDetail:
        assert token == "a" * 32
        assert product_id == "combo1"
        return ComboDetail(
            id="combo1",
            name="Combo criollo",
            price=Decimal("20.00"),
            groups=[
                ComboGroup(
                    id="g1",
                    name="Acompañamiento",
                    required=True,
                    minSelections=1,
                    maxSelections=1,
                    options=[
                        ComboOption(
                            productId="opt1",
                            name="Papas",
                            surcharge=Decimal("0"),
                            available=True,
                        ),
                        ComboOption(
                            productId="opt2",
                            name="Yuca",
                            surcharge=Decimal("3.00"),
                            available=True,
                        ),
                    ],
                )
            ],
        )

    async def create_order(
        self,
        token: str,
        phone: str,
        lines: list[dict[str, Any]],
        conversation_id: str,
    ) -> OrderResult:
        self.order_calls.append(
            {
                "token": token,
                "phone": phone,
                "lines": lines,
                "conversation_id": conversation_id,
            }
        )
        return OrderResult(
            id="o1",
            code="PED-001",
            status="confirmado",
            total=Decimal("32.50"),
        )


class QuietEngine:
    async def reply(self, session, user_message, tool_handler) -> str:
        return "¿Qué deseas agregar?"


def test_extract_qr_token() -> None:
    assert extract_qr_token("Hola FUDIA:" + "A" * 32) == "a" * 32
    assert extract_qr_token("hola") is None


def test_explicit_confirmation() -> None:
    assert explicit_confirmation("Sí")
    assert explicit_confirmation("Confirmo")
    assert explicit_confirmation("Sí, confirmar")
    assert not explicit_confirmation("No confirmar")
    assert not explicit_confirmation("tal vez")


@pytest.mark.asyncio
async def test_bootstrap_qr_resolves_table_and_persists_session() -> None:
    store = MemoryConversationStore()
    service = ConciergeService(store, FakeFudia(), QuietEngine())

    reply = await service.handle_message("51999999999", "FUDIA:" + "a" * 32)

    assert "Restaurante Demo" in reply
    assert "mesa M1" in reply
    session = await store.get("51999999999")
    assert session is not None
    assert session.qr_token == "a" * 32
    assert session.table is not None
    assert session.table.name == "M1"


@pytest.mark.asyncio
async def test_disabled_concierge_rejects_qr() -> None:
    store = MemoryConversationStore()
    service = ConciergeService(store, FakeFudia(concierge_enabled=False), QuietEngine())

    reply = await service.handle_message("51999999999", "FUDIA:" + "a" * 32)

    assert "no está disponible" in reply
    assert await store.get("51999999999") is None


@pytest.mark.asyncio
async def test_service_requires_qr_before_conversation() -> None:
    service = ConciergeService(MemoryConversationStore(), FakeFudia(), QuietEngine())
    reply = await service.handle_message("51999999999", "Quiero un lomo")
    assert "escanea el QR" in reply


@pytest.mark.asyncio
async def test_cart_requires_explicit_confirmation_before_order() -> None:
    fudia = FakeFudia()
    session = ConversationSession(phone="51999999999", qr_token="a" * 32)
    tools = ConciergeTools(fudia, session, "quiero uno")

    added = await tools.execute("add_item", {"productId": "p1", "quantity": 1, "note": ""})
    assert added["ok"] is True

    prepared = await tools.execute("prepare_confirmation", {})
    assert prepared["ok"] is True
    assert session.awaiting_confirmation is True

    blocked = await tools.execute("confirm_order", {})
    assert blocked["code"] == "confirmation_required"
    assert fudia.order_calls == []

    confirmed_tools = ConciergeTools(fudia, session, "Sí")
    confirmed = await confirmed_tools.execute("confirm_order", {})
    assert confirmed["ok"] is True
    assert confirmed["order"]["code"] == "PED-001"
    assert len(fudia.order_calls) == 1
    assert session.cart == []


@pytest.mark.asyncio
async def test_combo_selection_is_validated_and_forwarded() -> None:
    fudia = FakeFudia()
    session = ConversationSession(phone="51999999999", qr_token="a" * 32)
    tools = ConciergeTools(fudia, session, "quiero el combo con yuca")

    incomplete = await tools.execute(
        "add_combo_item",
        {
            "productId": "combo1",
            "quantity": 1,
            "note": "",
            "selections": [],
        },
    )
    assert incomplete["code"] == "combo_group_incomplete"
    assert session.cart == []

    added = await tools.execute(
        "add_combo_item",
        {
            "productId": "combo1",
            "quantity": 1,
            "note": "",
            "selections": [{"groupId": "g1", "productId": "opt2"}],
        },
    )
    assert added["ok"] is True
    assert added["cart"]["total"] == "23.00"
    assert session.cart[0].item_type == "combo"
    assert session.cart[0].selections[0].name == "Yuca"

    prepared = await tools.execute("prepare_confirmation", {})
    assert prepared["ok"] is True
    confirmed = await ConciergeTools(fudia, session, "Sí").execute(
        "confirm_order", {}
    )
    assert confirmed["ok"] is True
    assert fudia.order_calls[0]["lines"][0]["productId"] == "combo1"
    assert fudia.order_calls[0]["lines"][0]["selections"] == [
        {"groupId": "g1", "productId": "opt2"}
    ]
