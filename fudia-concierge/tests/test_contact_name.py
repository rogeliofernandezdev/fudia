"""Full names are collected before checkout, alongside the contact phone."""

from __future__ import annotations

import pytest

from src.domain.web_chat import WebAction
from src.services.contact_name import CUSTOMER_NAME_PROMPT, customer_name_reply, full_customer_name
from tests.test_contact_phone import checkout
from tests.test_web_chat import message


@pytest.mark.parametrize("name", [
    "Ana Quispe", "María del Pilar Huamán Quispe", "Jean-Luc O’Neill", "A. Pérez", "李 明",
])
def test_full_names_accept_compound_names_and_surnames(name: str) -> None:
    assert full_customer_name(name)
    assert customer_name_reply(name) == name


@pytest.mark.parametrize("name", ["", "Ana", "   ", "Ana 123", "Ana🙂 Quispe", "Ana " + "Q" * 160])
def test_first_name_or_invalid_text_is_not_a_complete_name(name: str) -> None:
    assert not full_customer_name(name)


@pytest.mark.parametrize("text", ["ver carta", "quiero otro lomo", "no", "sí", "la cuenta"])
def test_name_question_does_not_consume_order_instructions(text: str) -> None:
    assert customer_name_reply(text) is None


@pytest.mark.asyncio
@pytest.mark.parametrize("delivery", [False, True])
async def test_name_then_phone_preserves_the_order_until_confirmation(delivery: bool) -> None:
    store, backend, service = await checkout(delivery=delivery, customer_name="")
    requested = await service.process_inbound(message(
        "review", "Mi pedido", WebAction(name="prepare_confirmation"),
    ))
    assert requested.reply == CUSTOMER_NAME_PROMPT
    incomplete = await service.process_inbound(message("first-name", "Ana"))
    assert incomplete.reply == CUSTOMER_NAME_PROMPT
    session = await store.get("browser-session", "web-route")
    assert session and session.awaiting_customer_name and not session.awaiting_confirmation
    assert not session.delivery.customer_name and not backend.delivery_orders

    name_message = message("full-name", "  Ana María   Quispe Huamán  ")
    saved = await service.process_inbound(name_message)
    repeated = await service.process_inbound(name_message)
    assert saved.reply == repeated.reply
    assert "Guardé tus nombres y apellidos" in saved.reply and "9 dígitos" in saved.reply
    assert "¿Qué deseas pedir?" not in saved.reply and not backend.delivery_orders
    session = await store.get("browser-session", "web-route")
    assert session and session.delivery.customer_name == "Ana María Quispe Huamán"
    assert not session.awaiting_customer_name and session.awaiting_contact_phone
    assert len(session.cart) == 1

    summary = await service.process_inbound(message("phone", "987654321"))
    assert "**Confirmar pedido**" in summary.reply and "Total:" not in summary.reply
    assert CUSTOMER_NAME_PROMPT not in summary.reply
    session = await store.get("browser-session", "web-route")
    assert session and session.awaiting_confirmation and session.pending_order_request_id
    if delivery:
        assert session.delivery.address == "Av. de prueba 100"
        assert session.delivery.reference == "Puerta azul" and session.delivery.location
    confirm = message(
        "confirm", "Sí, confirmo mi pedido",
        WebAction(name="confirm_order", confirmation_ref=session.pending_order_request_id),
    )
    await service.process_inbound(confirm)
    await service.process_inbound(confirm)
    assert len(backend.delivery_orders) == 1
    assert backend.delivery_orders[0]["customerName"] == "Ana María Quispe Huamán"
    assert backend.delivery_orders[0]["customerPhone"] == "51987654321"


@pytest.mark.asyncio
async def test_phone_then_name_resumes_checkout_without_repeating_phone_question() -> None:
    store, backend, service = await checkout(customer_name="")
    phone = await service.process_inbound(message("phone", "987654321"))
    assert CUSTOMER_NAME_PROMPT in phone.reply
    named = await service.process_inbound(message("name", "Me llamo José Pérez"))
    assert "**Confirmar pedido**" in named.reply and "Total:" not in named.reply
    assert "9 dígitos" not in named.reply and not backend.delivery_orders
    session = await store.get("browser-session", "web-route")
    assert session and session.delivery.customer_name == "José Pérez"
    assert session.contact_phone == "51987654321" and not session.profile_phone


@pytest.mark.asyncio
async def test_old_single_name_summary_cannot_be_confirmed_without_surnames() -> None:
    store, backend, service = await checkout()
    session = await store.get("browser-session", "web-route")
    assert session
    session.delivery.customer_name = "Ana"
    session.customer_contact_phone = "51987654321"
    session.awaiting_confirmation = True
    session.pending_order_request_id = "old-summary"
    await store.save(session)
    rejected = await service.process_inbound(message(
        "old-confirm", "Sí, confirmo mi pedido",
        WebAction(name="confirm_order", confirmation_ref="old-summary"),
    ))
    assert rejected.reply == CUSTOMER_NAME_PROMPT and not backend.delivery_orders
    fixed = await service.process_inbound(message("full-name", "Ana Quispe"))
    assert "**Confirmar pedido**" in fixed.reply and "Total:" not in fixed.reply
    assert "9 dígitos" not in fixed.reply
