from decimal import Decimal
from typing import Any

import pytest

from src.domain.models import (
    BillItem,
    BillSummary,
    CartLine,
    CartModifier,
    CartSelection,
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
from src.services.concierge_service import (
    ConciergeService,
    agent_message,
    extract_qr_token,
)
from src.services.ports import FudiaError, LocalTableStartCodes
from src.services.tools import ConciergeTools, explicit_confirmation


class FakeFudia:
    def __init__(self, concierge_enabled: bool = True) -> None:
        self.order_calls: list[dict[str, Any]] = []
        self.bill_calls: list[dict[str, Any]] = []
        self.handoff_calls: list[dict[str, Any]] = []
        self.handoff_status = "none"
        self.concierge_enabled = concierge_enabled
        self.opened: list[dict[str, str]] = []
        self.usage_reports: list[dict[str, Any]] = []
        # foods-backend in enforce mode at 100 %: only new conversations fail.
        self.conversation_limit_reached = False

    async def open_conversation(
        self, token: str, conversation_id: str, kind: str
    ) -> None:
        known = any(o["conversation_id"] == conversation_id for o in self.opened)
        if self.conversation_limit_reached and not known:
            raise FudiaError(
                409,
                "concierge_conversation_limit_reached",
                "Fudia Concierge no puede iniciar conversaciones nuevas.",
            )
        if not known:
            self.opened.append(
                {"token": token, "conversation_id": conversation_id, "kind": kind}
            )

    async def report_usage(
        self,
        token: str,
        conversation_id: str,
        request_id: str,
        tiers: list[dict[str, Any]],
    ) -> None:
        self.usage_reports.append(
            {
                "token": token,
                "conversation_id": conversation_id,
                "request_id": request_id,
                "tiers": tiers,
            }
        )

    async def resolve_table(self, token: str) -> TableContext:
        assert token == "a" * 32
        return TableContext(
            name="M1",
            seats=4,
            zone="Principal",
            organizationName="Restaurante Demo",
            locationName="Local principal",
            conciergeEnabled=self.concierge_enabled,
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
        request_id: str = "",
    ) -> BillSummary:
        self.bill_calls.append(
            {
                "token": token,
                "phone": phone,
                "conversation_id": conversation_id,
                "request_id": request_id,
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


class FixedRouter:
    def __init__(self, route: str) -> None:
        self.value = route
        self.calls = 0

    async def route(self, session, user_message) -> str:
        self.calls += 1
        return self.value



def test_table_turn_carries_the_state_of_the_attention() -> None:
    session = ConversationSession(
        phone="51999999999",
        qr_token="a" * 32,
        table=TableContext(
            name="MESA 7",
            seats=4,
            organizationName="La Panchita",
            locationName="Miraflores",
            conciergeEnabled=True,
        ),
        cart=[
            CartLine(
                product_id="p1",
                name="Lomo saltado",
                quantity=2.0,
                unit_price=Decimal("15"),
                modifiers=[
                    CartModifier(
                        group_id="g",
                        group_name="Extras",
                        option_id="o",
                        name="Sin cebolla",
                        surcharge=Decimal("0"),
                    )
                ],
            ),
            CartLine(
                product_id="m1",
                name="Menú",
                quantity=1.0,
                unit_price=Decimal("12"),
                item_type="combo",
                selections=[
                    CartSelection(
                        group_id="g1",
                        group_name="Segundo",
                        product_id="p2",
                        name="Ají de gallina",
                        surcharge=Decimal("0"),
                    )
                ],
            ),
        ],
        awaiting_confirmation=True,
        last_order_id="o1",
        handoff_pending=True,
    )

    note = agent_message(session, "otro igual al primero")

    assert note == (
        "[Estado de la atención (no lo copies textual)]\n"
        "Modo: mesa abierta, MESA 7 en La Panchita - Miraflores.\n"
        "Carrito de esta ronda (aún no enviado a cocina):\n"
        "- 2 × Lomo saltado (Sin cebolla): 30.00\n"
        "- 1 × Menú (Segundo: Ají de gallina): 12.00\n"
        "Total del carrito: 42.00\n"
        "Confirmación pendiente: ya se mostró el resumen; un «sí» del cliente "
        "lo envía y cualquier cambio exige mostrarlo de nuevo.\n"
        "Pedidos ya enviados en esta atención: sí.\n"
        "El personal del local ya fue avisado y está en camino.\n\n"
        "Mensaje del cliente:\notro igual al primero"
    )


def test_table_turn_with_empty_cart_says_so() -> None:
    session = ConversationSession(
        phone="51999999999",
        qr_token="a" * 32,
        table=TableContext(
            name="M1",
            seats=4,
            organizationName="Restaurante Demo",
            locationName="Local principal",
            conciergeEnabled=True,
        ),
    )

    note = agent_message(session, "hola")

    assert "Modo: mesa abierta, Mesa M1" in note
    assert "Carrito de esta ronda: vacío." in note
    assert "Pedidos ya enviados en esta atención: ninguno." in note
    assert "Confirmación pendiente" not in note


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
    assert "Mesa M1" in reply
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

    assert "no está habilitado" in reply
    assert await store.get("51999999999") is None


@pytest.mark.asyncio
async def test_service_requires_qr_before_conversation() -> None:
    service = ConciergeService(MemoryConversationStore(), FakeFudia(), QuietEngine())
    reply = await service.handle_message("51999999999", "Quiero un lomo")
    assert "vuelve a escanear el QR" in reply


@pytest.mark.asyncio
async def test_short_start_code_opens_the_table_like_the_qr_token() -> None:
    store = MemoryConversationStore()
    codes = LocalTableStartCodes()
    service = ConciergeService(store, FakeFudia(), QuietEngine(), start_codes=codes)
    code = await codes.issue("a" * 32)

    reply = await service.handle_message(
        "51999999999", f"¡Hola! 👋 Quiero ver la carta.\nMesa M1 · {code}"
    )

    assert "Restaurante Demo" in reply and "Mesa M1" in reply
    session = await store.get("51999999999")
    assert session is not None and session.qr_token == "a" * 32


@pytest.mark.asyncio
async def test_qr_start_welcomes_the_diner_with_the_menu() -> None:
    service = ConciergeService(MemoryConversationStore(), FakeFudia(), QuietEngine())

    reply = await service.handle_message("51999999999", "FUDIA:" + "a" * 32)

    assert reply.startswith("🍽️ Bienvenido a Restaurante Demo\n📍 Mesa M1\n\n")
    assert "🍛 Platos:\n- Lomo saltado (S/ 32.50)" in reply
    assert reply.endswith("¿Qué deseas pedir?")


@pytest.mark.asyncio
async def test_qr_start_still_welcomes_when_the_menu_fails() -> None:
    class MenuDownFudia(FakeFudia):
        async def search_menu(
            self, token: str, query: str = "", product_id: str = ""
        ) -> MenuResponse:
            raise FudiaError(503, "fudia_unreachable", "sin conexión")

    store = MemoryConversationStore()
    service = ConciergeService(store, MenuDownFudia(), QuietEngine())

    reply = await service.handle_message("51999999999", "FUDIA:" + "a" * 32)

    assert reply == (
        "🍽️ Bienvenido a Restaurante Demo\n📍 Mesa M1\n\n¿Qué deseas pedir?"
    )
    session = await store.get("51999999999")
    assert session is not None and session.table is not None


@pytest.mark.asyncio
async def test_start_code_alone_and_in_lowercase_still_opens_the_table() -> None:
    store = MemoryConversationStore()
    codes = LocalTableStartCodes()
    service = ConciergeService(store, FakeFudia(), QuietEngine(), start_codes=codes)
    code = await codes.issue("a" * 32)

    reply = await service.handle_message("51911111111", code.lower())

    assert "Mesa M1" in reply


@pytest.mark.asyncio
async def test_expired_start_code_asks_to_scan_again_without_seating_the_diner() -> None:
    now = [1_000.0]
    store = MemoryConversationStore()
    codes = LocalTableStartCodes(ttl_seconds=600, clock=lambda: now[0])
    service = ConciergeService(store, FakeFudia(), QuietEngine(), start_codes=codes)
    code = await codes.issue("a" * 32)
    now[0] += 601

    reply = await service.handle_message(
        "51999999999", f"¡Hola! 👋 Quiero ver la carta.\nMesa M1 · {code}"
    )

    assert "ya venció" in reply
    assert await store.get("51999999999") is None


@pytest.mark.asyncio
async def test_deleted_start_message_never_guesses_a_table() -> None:
    store = MemoryConversationStore()
    codes = LocalTableStartCodes()
    service = ConciergeService(store, FakeFudia(), QuietEngine(), start_codes=codes)
    await codes.issue("a" * 32)

    reply = await service.handle_message("51999999999", "Hola, quiero 2 lomos")

    assert "vuelve a escanear el QR" in reply
    assert await store.get("51999999999") is None


@pytest.mark.asyncio
async def test_math_question_is_blocked_before_the_main_model() -> None:
    store = MemoryConversationStore()
    engine = CountingEngine()
    service = ConciergeService(store, FakeFudia(), engine)

    await service.handle_message("51999999999", "FUDIA:" + "a" * 32)
    reply = await service.handle_message(
        "51999999999",
        "¿Cuánto es 3/4 + 2/5?",
    )

    assert "únicamente con el pedido" in reply
    assert engine.calls == 0


@pytest.mark.asyncio
async def test_general_question_is_rejected_by_unified_router() -> None:
    store = MemoryConversationStore()
    router = FixedRouter("out_of_scope")
    agent = CountingEngine()
    service = ConciergeService(
        store,
        FakeFudia(),
        router,  # type: ignore[arg-type]
        {
            "menu": agent,
            "order": agent,
            "service": agent,
        },
    )

    await service.handle_message("51999999999", "FUDIA:" + "a" * 32)
    reply = await service.handle_message(
        "51999999999",
        "¿Quién fue Napoleón?",
    )

    assert "únicamente con el pedido" in reply
    assert router.calls == 1
    assert agent.calls == 0


@pytest.mark.asyncio
async def test_valid_route_invokes_one_specialist() -> None:
    store = MemoryConversationStore()
    router = FixedRouter("menu")
    agent = CountingEngine()
    service = ConciergeService(
        store,
        FakeFudia(),
        router,  # type: ignore[arg-type]
        {
            "menu": agent,
            "order": agent,
            "service": agent,
        },
    )

    await service.handle_message("51999999999", "FUDIA:" + "a" * 32)
    reply = await service.handle_message(
        "51999999999",
        "Quiero ver las bebidas",
    )

    assert reply == "Continuemos con tu pedido."
    assert router.calls == 1
    assert agent.calls == 1


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


@pytest.mark.asyncio
async def test_search_menu_exposes_recommended_from_backend() -> None:
    class RecommendedFudia(FakeFudia):
        async def search_menu(
            self, token: str, query: str = "", product_id: str = ""
        ) -> MenuResponse:
            return MenuResponse(
                items=[
                    MenuItem(
                        productId="p2",
                        name="Ceviche",
                        price=Decimal("38"),
                        categoryName="Platos",
                        recommended=True,
                    ),
                    MenuItem(
                        productId="p1",
                        name="Lomo saltado",
                        price=Decimal("32.50"),
                        categoryName="Platos",
                    ),
                ],
                currencySymbol="S/",
            )

    session = ConversationSession(phone="51999999999", qr_token="a" * 32)
    tools = ConciergeTools(RecommendedFudia(), session, "qué me recomiendas")
    menu = await tools.execute("search_menu", {"query": ""})
    assert [item["recommended"] for item in menu["items"]] == [True, False]
    assert MenuItem(productId="x", name="Agua", price=Decimal("3")).recommended is False


class SearchingFudia(FakeFudia):
    """Matches queries by name like foods-backend, so a query can miss."""

    def __init__(self, items: list[MenuItem]) -> None:
        super().__init__()
        self.items = items
        self.queries: list[str] = []

    async def search_menu(
        self, token: str, query: str = "", product_id: str = ""
    ) -> MenuResponse:
        self.queries.append(query)
        found = [i for i in self.items if query.lower() in i.name.lower()]
        return MenuResponse(items=found, currencySymbol="S/")


@pytest.mark.asyncio
async def test_search_without_match_returns_the_full_carta_not_an_empty_one() -> None:
    lomo = MenuItem(productId="p1", name="Lomo saltado", price=Decimal("15"))
    fudia = SearchingFudia([lomo])
    session = ConversationSession(phone="51999999999", qr_token="a" * 32)
    tools = ConciergeTools(fudia, session, "¿Qué tienes para hoy?")

    menu = await tools.execute("search_menu", {"query": "para hoy"})

    assert fudia.queries == ["para hoy", ""]
    assert menu["matched"] is False and menu["menuEmpty"] is False
    assert [item["name"] for item in menu["items"]] == ["Lomo saltado"]
    assert menu["unavailable"] is True


@pytest.mark.asyncio
@pytest.mark.parametrize("intent", ["menu", "order"])
@pytest.mark.parametrize("missing_status", [None, "sold_out", "out_of_hours", "unavailable"])
async def test_unavailable_request_returns_canonical_carta_not_model_summary(
    intent: str, missing_status: str | None,
) -> None:
    items = [
        MenuItem(productId="combo1", name="Combo criollo", price=Decimal("20"), isCombo=True),
        MenuItem(productId="opt1", name="Papas", price=Decimal("6")),
        MenuItem(productId="p9", name="Ceviche", price=Decimal("25"), status="low"),
    ]
    if missing_status:
        items.append(MenuItem(productId="chaufa", name="Chaufa", price=Decimal("18"), status=missing_status))
    fudia = SearchingFudia(items)
    store = MemoryConversationStore()

    class SummarizingAgent:
        async def reply(self, session, user_message, tool_handler) -> str:
            await tool_handler("search_menu", {"query": "chaufa"})
            return "Te puedo ofrecer un delicioso ceviche. ¿Te provoca?"

    agent = SummarizingAgent()
    service = ConciergeService(
        store, fudia, FixedRouter(intent), agents={"menu": agent, "order": agent, "service": agent},
    )
    await service.handle_message("51999999999", "FUDIA:" + "a" * 32)
    reply = await service.handle_message("51999999999", "Quiero pedir un chaufa")

    assert reply == (
        "Lo que buscas no está disponible hoy. Te comparto nuestra carta:\n\n"
        "🍱 Combo criollo (S/ 20.00)\n"
        "🍴 Otros:\n- Papas\n"
        "🍴 Acompañamiento:\n- Yuca (+S/ 3.00)\n\n"
        "➕ Extras\n🍴 Otros:\n- Ceviche (S/ 25.00)\n\n¿Qué deseas pedir?"
    )
    assert "Chaufa" not in reply and "delicioso" not in reply
    assert fudia.queries[-2:] == ["chaufa", ""]
    session = await store.get("51999999999")
    assert session is not None and session.cart == []
    assert session.messages[-1].content == reply
    assert not fudia.order_calls


@pytest.mark.asyncio
@pytest.mark.parametrize("items", [[], [MenuItem(
    productId="p1", name="Chaufa", price=Decimal("18"), status="sold_out",
)]])
async def test_unavailable_request_with_no_alternatives_does_not_offer_an_empty_carta(items) -> None:
    class SearchingAgent:
        async def reply(self, session, user_message, tool_handler) -> str:
            await tool_handler("search_menu", {"query": "chaufa"})
            return "Prueba otro plato."

    service = ConciergeService(MemoryConversationStore(), SearchingFudia(items), SearchingAgent())
    await service.handle_message("51999999999", "FUDIA:" + "a" * 32)
    reply = await service.handle_message("51999999999", "Quiero un chaufa")
    assert reply == (
        "Hoy no tenemos platos disponibles.\n\n"
        "Si necesitas ayuda, puedo avisar al personal del local."
    )


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("already_in_reply", "long_acknowledgement"),
    [(False, False), (True, False), (False, True)],
)
async def test_unavailable_carta_preserves_other_actions_and_does_not_repeat_them(
    already_in_reply, long_acknowledgement,
) -> None:
    from src.services.menu_tools import unavailable_carta_reply

    class MixedAgent:
        async def reply(self, session, user_message, tool_handler) -> str:
            missing = await tool_handler("search_menu", {"query": "chaufa"})
            added = await tool_handler("add_item", {"productId": "p1", "quantity": 1})
            assert added["ok"] is True
            reply = "Agregué un lomo saltado a tu pedido."
            if long_acknowledgement:
                reply += "\n" + "Detalle de pedido. " * 210
            return reply + ("\n\n" + unavailable_carta_reply(missing) if already_in_reply else "")

    service = ConciergeService(
        MemoryConversationStore(),
        SearchingFudia([MenuItem(productId="p1", name="Lomo saltado", price=Decimal("15"))]),
        MixedAgent(),
    )
    await service.handle_message("51999999999", "FUDIA:" + "a" * 32)
    reply = await service.handle_message("51999999999", "Un lomo y un chaufa")
    assert reply.startswith("Agregué un lomo saltado a tu pedido.")
    if long_acknowledgement and not already_in_reply:
        assert "Te comparto nuestra carta:" not in reply
    else:
        assert reply.count("Te comparto nuestra carta:") == 1
        assert "- Lomo saltado (S/ 15.00)" in reply
    assert len(reply) <= 4096
    session = await service.store.get("51999999999")
    assert session is not None and len(session.cart) == 1
    assert session.cart[0].product_id == "p1" and session.cart[0].quantity == 1


@pytest.mark.asyncio
async def test_full_carta_details_combos_and_flags_their_dishes() -> None:
    fudia = SearchingFudia(
        [
            MenuItem(productId="combo1", name="Combo criollo", price=Decimal("20"), isCombo=True),
            MenuItem(productId="opt1", name="Papas", price=Decimal("6")),
            MenuItem(productId="p9", name="Ceviche", price=Decimal("25")),
        ]
    )
    session = ConversationSession(phone="51999999999", qr_token="a" * 32)

    menu = await ConciergeTools(fudia, session, "la carta").execute(
        "search_menu", {"query": ""}
    )

    assert [combo["name"] for combo in menu["combos"]] == ["Combo criollo"]
    group = menu["combos"][0]["groups"][0]
    assert group["name"] == "Acompañamiento"
    assert [option["name"] for option in group["options"]] == ["Papas", "Yuca"]
    flags = {item["name"]: item["includedInCombo"] for item in menu["items"]}
    assert flags == {"Combo criollo": False, "Papas": True, "Ceviche": False}

    assert menu["noneAvailable"] is False
    assert menu["cartaText"] == (
        "🍱 Combo criollo (S/ 20.00)\n"
        "🍴 Otros:\n- Papas\n"
        "🍴 Acompañamiento:\n- Yuca (+S/ 3.00)\n\n"
        "➕ Extras\n"
        "🍴 Otros:\n- Ceviche (S/ 25.00)"
    )

    lookup = await ConciergeTools(fudia, session, "ceviche").execute(
        "search_menu", {"query": "ceviche"}
    )
    assert lookup["combos"] == [] and lookup["cartaText"] == ""


@pytest.mark.asyncio
async def test_full_carta_reports_when_everything_is_sold_out() -> None:
    fudia = SearchingFudia(
        [
            MenuItem(productId="p1", name="Ceviche", price=Decimal("25"), status="sold_out"),
            MenuItem(productId="p2", name="Causa", price=Decimal("20"), status="sold_out"),
        ]
    )
    session = ConversationSession(phone="51999999999", qr_token="a" * 32)

    menu = await ConciergeTools(fudia, session, "que tienes para hoy").execute(
        "search_menu", {"query": "para hoy"}
    )

    assert menu["matched"] is False and menu["menuEmpty"] is False
    assert menu["noneAvailable"] is True
    assert menu["cartaText"] == (
        "Hoy no tenemos platos disponibles."
    )


@pytest.mark.asyncio
async def test_search_with_match_or_empty_carta_is_reported_as_such() -> None:
    lomo = MenuItem(productId="p1", name="Lomo saltado", price=Decimal("15"))
    session = ConversationSession(phone="51999999999", qr_token="a" * 32)

    fudia = SearchingFudia([lomo])
    found = await ConciergeTools(fudia, session, "lomo").execute(
        "search_menu", {"query": "lomo"}
    )
    assert fudia.queries == ["lomo"]
    assert found["matched"] is True and found["menuEmpty"] is False

    empty = await ConciergeTools(SearchingFudia([]), session, "carta").execute(
        "search_menu", {"query": "ceviche"}
    )
    assert empty["matched"] is False and empty["menuEmpty"] is True
    assert empty["items"] == []
