"""Checkout contact replies stay in the order flow, without a model or profile lookup."""

from __future__ import annotations

import pytest

from src.domain.models import ConversationSession, EffectRecord, ProcessingRecord
from src.domain.web_chat import WebAction
from src.infrastructure.state_store import MemoryConversationStore
from src.services.concierge_service import OFF_SCOPE_REPLY, ConciergeService
from src.services.contact_phone import normalize_contact_phone
from src.services.processing import recovery_reply
from tests.test_delivery import PIN, DeliveryFudia
from tests.test_web_chat import NoModel, message


class ContactFudia(DeliveryFudia):
    async def resolve_table(self, token: str):
        return (await super().resolve_table(token)).model_copy(
            update={"countryCode": "PE", "serviceType": "delivery"}
        )

    async def get_customer(self, token: str, phone: str):
        raise AssertionError("A typed contact phone cannot read a customer profile")


async def checkout(
    *, delivery: bool = False, store: MemoryConversationStore | None = None,
    customer_name: str = "Ana Quispe",
):
    store, backend = store or MemoryConversationStore(), ContactFudia()
    service = ConciergeService(store, backend, NoModel())
    await service.process_inbound(message("start", "FUDIA:" + "a" * 32))
    await service.process_inbound(message(
        "add", "Agregar un plato",
        WebAction(name="add_item", args={"productId": "p1", "quantity": 1}),
    ))
    await service.process_inbound(message(
        "fulfillment", "Para recoger", WebAction(name="set_fulfillment", args={"type": "pickup"}),
    ))
    if customer_name:
        await service.process_inbound(message(
            "details", customer_name,
            WebAction(name="set_delivery_details", args={"customerName": customer_name}),
        ))
    if delivery:
        await service.process_inbound(message(
            "address", "Dirección de entrega",
            WebAction(name="set_delivery_details", args={
                "address": "Av. de prueba 100", "reference": "Puerta azul",
            }),
        ))
        await service.process_inbound(message("pin", "Mi ubicación").model_copy(update={"location": PIN}))
    return store, backend, service


@pytest.mark.parametrize(("text", "country", "expected"), [
    ("987654321", "PE", "51987654321"),
    ("987 654 321", "PE", "51987654321"),
    ("51987654321", "PE", "51987654321"),
    ("+51 987-654-321", "PE", "51987654321"),
    ("+1 (202) 555-0123", "PE", "12025550123"),
    ("+34 612 345 678", "ES", "34612345678"),
    ("987654321", "", ""),
    ("987654321", "ES", ""),
    ("123456789", "PE", ""),
    ("98765432", "PE", ""),
    ("+000000000", "PE", ""),
    ("+1234567890123456", "PE", ""),
    ("2+2", "PE", ""),
    ("mi número es 987654321", "PE", ""),
])
def test_contact_normalization(text: str, country: str, expected: str) -> None:
    assert normalize_contact_phone(text, country) == expected


@pytest.mark.asyncio
@pytest.mark.parametrize("phone", ["987654321", "+51987654321", "51987654321"])
@pytest.mark.parametrize("delivery", [False, True])
async def test_contact_completes_checkout_without_losing_order(phone: str, delivery: bool) -> None:
    store, backend, service = await checkout(delivery=delivery)
    review = await service.process_inbound(message(
        "review", "Revisar mi pedido", WebAction(name="prepare_confirmation"),
    ))
    assert "9 dígitos" in review.reply and "código de país" not in review.reply
    session = await store.get("browser-session", "web-route")
    assert session and session.awaiting_contact_phone and not session.awaiting_confirmation
    assert not session.pending_order_request_id and not session.order_submission_started
    assert not backend.delivery_orders

    contact_message = message("phone", phone)
    accepted = await service.process_inbound(contact_message)
    repeated = await service.process_inbound(contact_message)
    assert repeated.reply == accepted.reply
    assert "Guardé tu número" in accepted.reply and "Confirmar pedido" in accepted.reply
    assert "Lomo saltado" in accepted.reply and "Ana Quispe" in accepted.reply
    assert OFF_SCOPE_REPLY not in accepted.reply and "¿Qué deseas pedir?" not in accepted.reply
    assert not backend.delivery_orders
    session = await store.get("browser-session", "web-route")
    assert session and session.contact_phone == "51987654321"
    assert not session.profile_phone and not session.awaiting_contact_phone
    assert session.awaiting_confirmation and session.pending_order_request_id
    assert session.delivery.customer_name == "Ana Quispe" and len(session.cart) == 1
    if delivery:
        assert "Av. de prueba 100" in accepted.reply and "8.00" in accepted.reply
        assert session.delivery.reference == "Puerta azul" and session.delivery.location
    confirm = message(
        "confirm", "Sí, confirmo mi pedido",
        WebAction(name="confirm_order", confirmation_ref=session.pending_order_request_id),
    )
    done = await service.process_inbound(confirm)
    await service.process_inbound(confirm)
    assert "registrado" in done.reply and len(backend.delivery_orders) == 1
    order = backend.delivery_orders[0]
    assert order["customerPhone"] == "51987654321" and order["customerName"] == "Ana Quispe"
    assert len(order["items"]) == 1 and order["fulfillment"] == ("delivery" if delivery else "pickup")
    session = await store.get("browser-session", "web-route")
    assert session and not session.cart


@pytest.mark.asyncio
async def test_old_prepared_order_requests_contact_without_starting_submission() -> None:
    store, backend, service = await checkout()
    # Session prepared before this fix: Confirm used to discover the missing phone too late.
    session = await store.get("browser-session", "web-route")
    assert session
    session.awaiting_confirmation = True
    session.pending_order_request_id = "old-summary"
    await store.save(session)
    reply = await service.process_inbound(message(
        "old-confirm", "Sí, confirmo mi pedido",
        WebAction(name="confirm_order", confirmation_ref="old-summary"),
    ))
    assert "9 dígitos" in reply.reply and not backend.delivery_orders
    session = await store.get("browser-session", "web-route")
    assert session and session.awaiting_contact_phone and not session.order_submission_started
    resumed = await service.process_inbound(message("phone", "+51987654321"))
    assert "Confirmar pedido" in resumed.reply and "Lomo saltado" in resumed.reply
    assert not backend.delivery_orders
    stale = await service.process_inbound(message(
        "stale-confirm", "Sí, confirmo mi pedido",
        WebAction(name="confirm_order", confirmation_ref="old-summary"),
    ))
    assert "cambió" in stale.reply and not backend.delivery_orders


@pytest.mark.asyncio
async def test_pre_fix_phone_refusal_recovers_existing_cart_without_pending_flag() -> None:
    store, backend, service = await checkout()
    resumed = await service.process_inbound(message("phone", "+51987654321"))
    assert "Confirmar pedido" in resumed.reply and not backend.delivery_orders
    session = await store.get("browser-session", "web-route")
    assert session and session.awaiting_confirmation and session.contact_phone == "51987654321"


@pytest.mark.asyncio
async def test_invalid_phone_keeps_checkout_pending_and_arithmetic_stays_out_of_scope() -> None:
    store, backend, service = await checkout()
    await service.process_inbound(message("review", "Mi pedido", WebAction(name="prepare_confirmation")))
    invalid = await service.process_inbound(message("invalid-phone", "98765"))
    assert "celular válido de 9 dígitos" in invalid.reply
    rejected = await service.process_inbound(message("math", "2+2"))
    assert rejected.reply == OFF_SCOPE_REPLY
    session = await store.get("browser-session", "web-route")
    assert session and not session.contact_phone and session.awaiting_contact_phone
    assert len(session.cart) == 1 and not backend.delivery_orders


@pytest.mark.asyncio
async def test_whatsmeow_contact_does_not_grant_verified_identity() -> None:
    store, backend, service = await checkout()
    inbound = message("phone", "987654321").model_copy(update={"provider": "WHATSMEOW"})
    accepted = await service.process_inbound(inbound)
    assert "Responde «sí»" in accepted.reply and "pulsa" not in accepted.reply
    session = await store.get("browser-session", "web-route")
    assert session and not session.profile_phone and session.contact_phone == "51987654321"
    assert not backend.delivery_orders


def test_contact_recovery_preserves_checkout_reply() -> None:
    record = ProcessingRecord(message_id="phone", effects=[EffectRecord(
        kind="contact", result={"reply": "Guardé tu número. Revisa tu pedido y confirma."},
    )])
    assert recovery_reply(record, ConversationSession(phone="browser-session")) == (
        record.effects[0].result["reply"]
    )


@pytest.mark.asyncio
async def test_interrupted_contact_turn_recovers_the_summary_without_restarting_checkout() -> None:
    class InterruptedStore(MemoryConversationStore):
        interrupted = False

        async def commit(self, lease, session=None, record=None):
            await super().commit(lease, session, record)
            if (
                not self.interrupted and record and record.stage == "effects_applied"
                and record.effects[-1].kind == "contact"
                and str(record.effects[-1].result.get("reply", "")).startswith("Guardé")
            ):
                self.interrupted = True
                raise RuntimeError("Worker stopped after saving contact")

    store, backend, service = await checkout(store=InterruptedStore())
    await service.process_inbound(message("review", "Mi pedido", WebAction(name="prepare_confirmation")))
    contact_message = message("phone", "987654321")
    with pytest.raises(RuntimeError, match="Worker stopped"):
        await service.process_inbound(contact_message)
    resumed = await service.process_inbound(contact_message)
    assert "Lomo saltado" in resumed.reply and "Confirmar pedido" in resumed.reply
    session = await store.get("browser-session", "web-route")
    assert session and session.awaiting_confirmation and session.contact_phone == "51987654321"
    assert not backend.delivery_orders
    confirm = message(
        "confirm", "Sí, confirmo mi pedido",
        WebAction(name="confirm_order", confirmation_ref=session.pending_order_request_id or ""),
    )
    await service.process_inbound(confirm)
    await service.process_inbound(confirm)
    assert len(backend.delivery_orders) == 1
