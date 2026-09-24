from decimal import Decimal
from typing import Any

import pytest

from src.domain.models import (
    BillItem,
    BillSummary,
    ComboDetail,
    ComboGroup,
    ComboOption,
    ConversationSession,
    MenuItem,
    MenuResponse,
    ModifierConfig,
    ModifierGroup,
    ModifierOption,
    OrderResult,
    TableContext,
)
from src.infrastructure.state_store import MemoryConversationStore
from src.services.concierge_service import ConciergeService, extract_qr_token
from src.services.tools import ConciergeTools, explicit_confirmation


class FakeFudia:
    def __init__(self, concierge_enabled: bool = True) -> None:
        self.order_calls: list[dict[str, Any]] = []
        self.bill_calls: list[dict[str, Any]] = []
        self.handoff_calls: list[dict[str, Any]] = []
        self.handoff_status = "none"
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

    async def request_handoff(
        self,
        token: str,
        phone: str,
        conversation_id: str,
        reason: str,
    ) -> dict[str, Any]:
        self.handoff_status = "pending"
        self.handoff_calls.append(
            {
                "token": token,
                "phone": phone,
                "conversation_id": conversation_id,
                "reason": reason,
            }
        )
        return {"id": "h1", "status": "pending", "tableName": "M1"}

    async def get_handoff_status(
        self,
        token: str,
        conversation_id: str,
    ) -> dict[str, Any]:
        return {"status": self.handoff_status}

    async def create_order(
        self,
        token: str,
        phone: str,
        lines: list[dict[str, Any]],
        conversation_id: str,
        request_id: str,
    ) -> OrderResult:
        self.order_calls.append(
            {
                "token": token,
                "phone": phone,
                "lines": lines,
                "conversation_id": conversation_id,
                "request_id": request_id,
            }
        )
        return OrderResult(
            id="o1",
            code="PED-001",
            status="confirmado",
            total=Decimal("32.50"),
        )

    async def request_bill(
        self,
        token: str,
        phone: str,
        conversation_id: str,
    ) -> BillSummary:
        self.bill_calls.append(
            {
                "token": token,
                "phone": phone,
                "conversation_id": conversation_id,
            }
        )
        return BillSummary(
            orderId="o1",
            code="PED-001",
            tableName="M1",
            status="listo",
            currencySymbol="S/",
            items=[
                BillItem(
                    id="i1",
                    productId="p1",
                    name="Lomo saltado",
                    qty=Decimal("2"),
                    unitPrice=Decimal("32.50"),
                )
            ],
            total=Decimal("65.00"),
            paidAmount=Decimal("10.00"),
            remainingAmount=Decimal("55.00"),
            paymentStatus="partial",
        )


class QuietEngine:
    async def reply(self, session, user_message, tool_handler) -> str:
        return "¿Qué deseas agregar?"

class CountingEngine:
    def __init__(self) -> None:
        self.calls = 0

    async def reply(self, session, user_message, tool_handler) -> str:
        self.calls += 1
        return "Continuemos con tu pedido."



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
async def test_qr_session_rejects_wrong_whatsapp_recipient() -> None:
    store = MemoryConversationStore()
    service = ConciergeService(store, FakeFudia(), QuietEngine())

    reply = await service.handle_message(
        "51999999999",
        "FUDIA:" + "a" * 32,
        "+51 900 000 000",
    )

    assert "no corresponde al número de WhatsApp" in reply
    assert await store.get("51999999999") is None


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
    assert fudia.order_calls[0]["conversation_id"] == session.conversation_id
    assert fudia.order_calls[0]["request_id"]
    assert session.cart == []
    assert session.pending_order_request_id is None


@pytest.mark.asyncio
async def test_each_confirmed_round_uses_a_new_request_id() -> None:
    fudia = FakeFudia()
    session = ConversationSession(phone="51999999999", qr_token="a" * 32)

    first_tools = ConciergeTools(fudia, session, "quiero uno")
    await first_tools.execute(
        "add_item", {"productId": "p1", "quantity": 1, "note": ""}
    )
    await first_tools.execute("prepare_confirmation", {})
    first_request_id = session.pending_order_request_id
    assert first_request_id

    await ConciergeTools(fudia, session, "Sí").execute("confirm_order", {})

    second_tools = ConciergeTools(fudia, session, "otro más")
    await second_tools.execute(
        "add_item", {"productId": "p1", "quantity": 1, "note": ""}
    )
    await second_tools.execute("prepare_confirmation", {})
    second_request_id = session.pending_order_request_id
    assert second_request_id
    assert second_request_id != first_request_id

    await ConciergeTools(fudia, session, "Sí").execute("confirm_order", {})

    assert len(fudia.order_calls) == 2
    assert fudia.order_calls[0]["conversation_id"] == session.conversation_id
    assert fudia.order_calls[1]["conversation_id"] == session.conversation_id
    assert fudia.order_calls[0]["request_id"] != fudia.order_calls[1]["request_id"]


@pytest.mark.asyncio
async def test_bill_uses_backend_total_and_blocks_unconfirmed_cart() -> None:
    fudia = FakeFudia()
    session = ConversationSession(phone="51999999999", qr_token="a" * 32)

    tools = ConciergeTools(fudia, session, "la cuenta")
    bill = await tools.execute("request_bill", {})
    assert bill["ok"] is True
    assert bill["bill"]["total"] == "65.00"
    assert bill["bill"]["paidAmount"] == "10.00"
    assert bill["bill"]["remainingAmount"] == "55.00"
    assert len(fudia.bill_calls) == 1

    await tools.execute(
        "add_item", {"productId": "p1", "quantity": 1, "note": ""}
    )
    blocked = await tools.execute("request_bill", {})
    assert blocked["code"] == "unconfirmed_cart"
    assert len(fudia.bill_calls) == 1


@pytest.mark.asyncio
async def test_request_human_persists_handoff_and_marks_session() -> None:
    fudia = FakeFudia()
    session = ConversationSession(phone="51999999999", qr_token="a" * 32)
    tools = ConciergeTools(fudia, session, "quiero hablar con un mozo")

    result = await tools.execute(
        "request_human",
        {"reason": "El cliente quiere hablar con un mozo."},
    )

    assert result["ok"] is True
    assert session.handoff_pending is True
    assert len(fudia.handoff_calls) == 1
    assert fudia.handoff_calls[0]["conversation_id"] == session.conversation_id


@pytest.mark.asyncio
async def test_pending_handoff_pauses_llm_until_staff_resolves_it() -> None:
    store = MemoryConversationStore()
    fudia = FakeFudia()
    engine = CountingEngine()
    service = ConciergeService(store, fudia, engine)

    await service.handle_message("51999999999", "FUDIA:" + "a" * 32)
    session = await store.get("51999999999")
    assert session is not None
    session.handoff_pending = True
    await store.save(session)
    fudia.handoff_status = "pending"

    waiting = await service.handle_message("51999999999", "quiero otra bebida")
    assert "personal del local ya fue avisado" in waiting
    assert engine.calls == 0

    fudia.handoff_status = "resolved"
    resumed = await service.handle_message("51999999999", "quiero otra bebida")
    assert resumed == "Continuemos con tu pedido."
    assert engine.calls == 1
    updated = await store.get("51999999999")
    assert updated is not None
    assert updated.handoff_pending is False


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


class FakeModifierFudia(FakeFudia):
    async def search_menu(
        self, token: str, query: str = "", product_id: str = ""
    ) -> MenuResponse:
        item = MenuItem(
            productId="p-mod",
            name="Hamburguesa",
            price=Decimal("20.00"),
            categoryName="Platos",
            status="available",
            isCombo=False,
            hasModifiers=True,
        )
        if product_id and product_id != "p-mod":
            return MenuResponse(items=[], currencySymbol="S/")
        return MenuResponse(items=[item], currencySymbol="S/")

    async def get_modifiers(
        self, token: str, product_id: str
    ) -> ModifierConfig:
        assert product_id == "p-mod"
        return ModifierConfig(
            productId=product_id,
            groups=[
                ModifierGroup(
                    id="g1",
                    name="Tamaño",
                    required=True,
                    minSelections=1,
                    maxSelections=1,
                    options=[
                        ModifierOption(
                            id="o1",
                            name="Grande",
                            surcharge=Decimal("5.00"),
                        ),
                        ModifierOption(
                            id="o2",
                            name="Mediana",
                            surcharge=Decimal("0"),
                        ),
                    ],
                )
            ],
        )


@pytest.mark.asyncio
async def test_modified_item_requires_real_modifier_ids() -> None:
    fudia = FakeModifierFudia()
    from src.domain.models import ConversationSession

    session = ConversationSession(
        phone="51999999999",
        qr_token="a" * 32,
    )
    tools = ConciergeTools(fudia, session, "quiero una grande")

    plain = await tools.execute(
        "add_item",
        {"productId": "p-mod", "quantity": 1, "note": ""},
    )
    assert plain["code"] == "modifier_review_required"

    incomplete = await tools.execute(
        "add_modified_item",
        {
            "productId": "p-mod",
            "quantity": 1,
            "note": "",
            "modifiers": [],
        },
    )
    assert incomplete["code"] == "modifier_group_incomplete"

    added = await tools.execute(
        "add_modified_item",
        {
            "productId": "p-mod",
            "quantity": 1,
            "note": "",
            "modifiers": [{"groupId": "g1", "optionId": "o1"}],
        },
    )
    assert added["ok"] is True
    assert added["cart"]["total"] == "25.00"
    assert added["cart"]["items"][0]["modifiers"][0]["name"] == "Grande"

    prepared = await tools.execute("prepare_confirmation", {})
    assert prepared["ok"] is True
    confirmed = await ConciergeTools(
        fudia, session, "Sí"
    ).execute("confirm_order", {})
    assert confirmed["ok"] is True
    assert fudia.order_calls[0]["lines"][0]["modifiers"] == [
        {"groupId": "g1", "optionId": "o1"}
    ]
