"""Re-scan -> keep ordering -> wait -> card -> append once to the active order."""

from decimal import Decimal
from typing import Any

import pytest

from src.api.web_chat import order_summary, registered_items
from src.domain.models import MenuItem, MenuResponse, OrderResult, OrderStatus, RegisteredOrderItem
from src.domain.web_chat import WebAction
from src.infrastructure.state_store import MemoryConversationStore
from src.services.concierge_service import ConciergeService
from tests.test_contact_phone import ContactFudia
from tests.test_web_chat import NoModel, message


class AppendBackend(ContactFudia):
    created = 0
    supports = True
    status = "preparando"

    def __init__(self):
        super().__init__()
        self.lines: list[dict[str, Any]] = []
        self.total = Decimal("0")
        self.id = "11111111-1111-4111-8111-111111111111"

    async def search_menu(self, token, query="", product_id=""):
        items = [
            MenuItem(productId="p1", name="Lomo saltado", price=Decimal("20.00")),
            MenuItem(productId="p2", name="Tallarín saltado", price=Decimal("14.00")),
        ]
        return MenuResponse(
            currencySymbol="S/", items=[i for i in items if not product_id or i.productId == product_id]
        )

    async def get_order_status(self, token, order_id, conversation_id=""):
        return OrderStatus(
            orderId=self.id,
            code="PED-PRUEBA",
            channel="delivery",
            status=self.status,
            subtotal=self.total - Decimal("8.00"),
            deliveryFee=Decimal("8.00"),
            total=self.total,
            active=self.status not in {"cancelado", "entregado"},
            supportsAdditions=self.supports,
            items=[
                RegisteredOrderItem(
                    productId=line["productId"],
                    name="Lomo saltado" if line["productId"] == "p1" else "Tallarín saltado",
                    qty=line["qty"],
                    unitPrice=Decimal("20.00") if line["productId"] == "p1" else Decimal("14.00"),
                )
                for line in self.lines
            ],
        )

    async def create_delivery_order(self, token, order):
        if order.get("orderId"):
            assert order["orderId"] == self.id and Decimal(order["expectedOrderTotal"]) == self.total
        else:
            self.created += 1
        self.delivery_orders.append(order)
        self.lines += order["items"]
        self.total = Decimal("8.00") + sum(
            (Decimal("20.00") if line["productId"] == "p1" else Decimal("14.00")) * Decimal(str(line["qty"]))
            for line in self.lines
        )
        return OrderResult(id=self.id, code="PED-PRUEBA", status="preparando", total=self.total)


class EagerAgent:
    async def reply(self, session, text, tool):
        raw = text.rsplit("Mensaje del cliente:\n", 1)[-1]
        if "Tallarín" in raw:
            await tool("add_item", {"productId": "p2", "quantity": 1})
            # Reproduce the reported bug: the model tries to prepare before
            # its question has an answer and then invents a textual summary.
            result = await tool("prepare_confirmation", {})
            assert result["code"] == "continue_ordering"
            return "¿Deseas algo más? Tu pedido: 1 tallarín, Total S/ 22.00."
        await tool("prepare_confirmation", {})
        return "Resumen en texto que no debe llegar al navegador."


async def first_order():
    store, backend = MemoryConversationStore(), AppendBackend()
    service = ConciergeService(store, backend, NoModel())
    await service.process_inbound(message("start", "FUDIA:" + "a" * 32))
    for ref, action in [
        ("add", WebAction(name="add_item", args={"productId": "p1", "quantity": 1})),
        ("pickup", WebAction(name="set_fulfillment", args={"type": "pickup"})),
        (
            "details",
            WebAction(name="set_delivery_details", args={"customerName": "Ana Quispe", "address": "Calle 1"}),
        ),
    ]:
        await service.process_inbound(message(ref, "Actualizar pedido", action))
    from tests.test_delivery import PIN

    await service.process_inbound(message("pin", "Mi ubicación").model_copy(update={"location": PIN}))
    await service.process_inbound(message("phone", "987654321"))
    session = await store.get("browser-session", "web-route")
    assert session and session.pending_order_request_id
    await service.process_inbound(
        message(
            "confirm",
            "Sí, confirmo mi pedido",
            WebAction(
                name="confirm_order",
                confirmation_ref=session.pending_order_request_id,
            ),
        )
    )
    assert backend.created == 1 and backend.total == Decimal("28.00")
    return store, backend


@pytest.mark.asyncio
async def test_rescan_waits_then_confirms_into_the_same_order_with_one_delivery_fee():
    store, backend = await first_order()
    service = ConciergeService(store, backend, EagerAgent())
    await service.process_inbound(message("start:again", "FUDIA:" + "a" * 32))
    added = await service.process_inbound(message("add-more", "Agrega Tallarín saltado"))
    assert added.reply == "Actualicé tu pedido. ¿Deseas agregar algo más?"
    session = await store.get("browser-session", "web-route")
    assert session and session.last_order_id == backend.id and not session.awaiting_confirmation
    assert backend.created == 1 and len(backend.delivery_orders) == 1
    reviewed = await service.process_inbound(message("done", "No, eso es todo"))
    assert reviewed.reply == "Tu pedido está listo para confirmar en «Mi pedido»."
    session = await store.get("browser-session", "web-route")
    assert session and session.pending_order_request_id
    summary = order_summary(session, "delivery")
    assert summary.addingToOrder and summary.subtotal == "34.00"
    assert summary.deliveryFee == "8.00" and summary.total == "42.00"
    assert registered_items(session)[0]["name"] == "Lomo saltado"
    confirm = message(
        "confirm-addition",
        "Sí, confirmo mi pedido",
        WebAction(
            name="confirm_order",
            confirmation_ref=session.pending_order_request_id,
        ),
    )
    done = await service.process_inbound(confirm)
    await service.process_inbound(confirm)
    assert "Agregué los productos a tu pedido" in done.reply
    assert backend.created == 1 and len(backend.delivery_orders) == 2 and backend.total == Decimal("42.00")
    assert backend.delivery_orders[-1]["orderId"] == backend.id
    assert Decimal(backend.delivery_orders[-1]["expectedOrderTotal"]) == Decimal("28.00")


@pytest.mark.asyncio
@pytest.mark.parametrize("unsupported", [False, True])
async def test_unappendable_or_old_backend_never_silently_starts_another_order(unsupported):
    store, backend = await first_order()
    backend.supports = not unsupported
    backend.status = "en_camino" if not unsupported else "preparando"
    service = ConciergeService(store, backend, NoModel())
    await service.process_inbound(
        message(
            "add-more",
            "Agregar",
            WebAction(
                name="add_item",
                args={"productId": "p2", "quantity": 1},
            ),
        )
    )
    reviewed = await service.process_inbound(
        message("review", "Mi pedido", WebAction(name="prepare_confirmation"))
    )
    assert "personal" in reviewed.reply
    session = await store.get("browser-session", "web-route")
    assert session and not session.awaiting_confirmation and session.cart
    assert backend.created == 1 and len(backend.delivery_orders) == 1
