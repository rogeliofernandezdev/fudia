"""Deterministic button actions use the same tools, journal and order rules."""

from __future__ import annotations

from typing import Any

from src.services.contact_name import CUSTOMER_NAME_PROMPT

WEB_CONFIRMATION_REPLY = "Revisa tu pedido y toca **Confirmar pedido** para enviarlo."

def cart_line(cart: dict[str, Any], currency: str) -> str:
    """What the order holds after a cart button, in one line."""
    count = sum(float(item["quantity"]) for item in cart.get("items", []))
    if not count:
        return "Tu pedido quedó vacío."
    noun = "producto" if count == 1 else "productos"
    return f"Llevas {count:g} {noun} · {currency} {cart.get('total', '0.00')}."


def action_reply(
    name: str, result: dict[str, Any], currency: str, *, dish: str = "", quantity: float = 1,
    web: bool = True,
) -> str:
    """`dish` is what a cart button acted on: its tap is not echoed as a
    bubble, so the reply names it."""
    if not result.get("ok"):
        messages = {
            "empty_cart": "Tu pedido está vacío. Agrega un plato de la carta.",
            "customer_name_required": CUSTOMER_NAME_PROMPT,
            "customer_phone_required": (
                result.get("message") or "¿Cuál es tu número de contacto?"
            ),
            "delivery_details_required": missing_delivery_reply(result.get("missing", [])),
            "insufficient_stock": "La cantidad disponible cambió. Revisa la carta antes de agregarlo.",
            "product_unavailable": "Ese plato ya no está disponible. Puedes elegir otro de la carta.",
            "stale_confirmation": "El pedido cambió. Revisa nuevamente el resumen antes de confirmar.",
        }
        return str(
            messages.get(
                str(result.get("code")),
                result.get("message")
                or "No pudimos completar esa acción. Revisa tu pedido e inténtalo otra vez.",
            )
        )
    if name == "confirm_order":
        if result.get("order", {}).get("addedToOrder"):
            return (
                "Agregué los productos a tu pedido. "
                "El personal revisará esta adición para enviarla a preparación."
            )
        return "Tu pedido fue registrado. El personal del local lo revisará para enviarlo a preparación."
    if name == "request_human":
        return "El personal del local recibió tu solicitud de atención."
    if name == "request_bill":
        bill = result["bill"]
        symbol = bill["currencySymbol"]
        return (
            f"Tu cuenta\nTotal: {symbol} {bill['total']}\n"
            f"Pagado: {symbol} {bill['paidAmount']}\n"
            f"Saldo pendiente: {symbol} {bill['remainingAmount']}\n\n"
            "El personal recibió tu solicitud. El pago se realiza en el local."
        )
    if name == "set_delivery_details":
        return "Guardé tus datos para este pedido. Ya puedes revisar el resumen."
    if name == "set_fulfillment":
        return "Actualicé la forma de entrega. Revisa el pedido antes de confirmarlo."
    cart = result.get("cart", {})
    if name in {"prepare_confirmation", "view_cart"}:
        if web:
            return WEB_CONFIRMATION_REPLY
        items = cart.get("items", [])
        lines = []
        for item in items:
            lines.append(f"{item['quantity']:g} × {item['name']} · {currency} {item['lineTotal']}")
            choices = [choice["name"] for choice in item.get("modifiers", []) + item.get("selections", [])]
            if choices:
                lines.append("  " + ", ".join(choices))
            if item.get("note"):
                lines.append("  Nota: " + item["note"])
        delivery = result.get("delivery", {})
        customer = delivery.get("customerName") or result.get("counter", {}).get("customerName")
        if customer:
            lines.append("A nombre de: " + customer)
        if delivery.get("fulfillment") == "delivery":
            lines.extend(
                ["Entrega en: " + delivery["address"], f"Envío: {currency} {delivery['deliveryFee']}"]
            )
        elif delivery.get("fulfillment") == "pickup":
            lines.append("Recojo en el local.")
        summary = "\n".join(lines)
        total = result.get("delivery", {}).get("total", cart.get("total", "0.00"))
        confirmation = (
            "\nRevisa el resumen y toca **Confirmar pedido** para enviarlo."
            if web else "\n¿Confirmas este pedido? Responde sí para enviarlo."
        )
        return f"Tu pedido\n{summary}\n\nTotal: {currency} {total}" + (
            confirmation if name == "prepare_confirmation" else ""
        )
    if dish and name.startswith("add_"):
        return f"Agregué {quantity:g} × {dish}.\n{cart_line(cart, currency)}"
    if dish and name == "remove_item":
        return f"Quité {dish} de tu pedido.\n{cart_line(cart, currency)}"
    return "Actualicé tu pedido. Puedes seguir eligiendo o revisar el resumen."


def missing_delivery_reply(missing: list[str]) -> str:
    prompts = {
        "fulfillment": "¿Recogerás el pedido en el local o deseas delivery?",
        "customerName": CUSTOMER_NAME_PROMPT,
        "location": "Comparte tu ubicación para calcular el costo de envío.",
        "locationInRange": "Comparte una ubicación dentro de la cobertura o elige recojo en el local.",
        "address": "Escríbeme la dirección exacta de entrega.",
    }
    return " ".join(prompts[key] for key in missing if key in prompts) or (
        "Revisa los datos de entrega para continuar."
    )
