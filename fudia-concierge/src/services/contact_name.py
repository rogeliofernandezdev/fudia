"""Collect the diner's full name without guessing how to split surnames."""

from __future__ import annotations

from typing import Any

from src.services.tooling import ToolContext

CUSTOMER_NAME_PROMPT = "Escríbeme tus nombres y apellidos para registrar el pedido."


def full_customer_name(value: str) -> bool:
    parts = value.split()
    return 2 <= len(parts) and len(value) <= 160 and all(
        any(char.isalpha() for char in part) and all(char.isalpha() or char in "-'’." for char in part)
        for part in parts
    )


def customer_name_reply(text: str) -> str | None:
    """Only consume a standalone answer to a pending name question.

    Order instructions continue through the regular router; the tool validates
    the extracted name separately, so these words never ban a real surname.
    """
    value = " ".join(text.split())
    for prefix in ("me llamo ", "mi nombre es ", "soy "):
        if value.lower().startswith(prefix):
            value = value[len(prefix):]
            break
    commands = {
        "carta", "pedido", "delivery", "recojo", "confirmo", "cancelar", "agrega", "agregar",
        "quita", "quitar", "quiero", "ver", "cuenta", "ayuda", "gracias", "hola", "no", "sí", "si",
    }
    if commands.intersection(value.lower().split()) or any(
        not (char.isalpha() or char in " -'’.") for char in value
    ):
        return None
    return value if full_customer_name(value) else ""


async def request_customer_name(context: ToolContext) -> dict[str, Any]:
    context.session.awaiting_customer_name = True
    context.invalidate_confirmation()
    context.session.order_submission_started = False
    await context.journal.local_effect("contact", {"reply": CUSTOMER_NAME_PROMPT})
    return {
        "ok": False, "code": "customer_name_required", "missing": ["customerName"],
        "message": CUSTOMER_NAME_PROMPT,
    }
