"""Deterministic button actions use the same tools, journal and order rules."""

from __future__ import annotations

from typing import Any


def cart_line(cart: dict[str, Any], currency: str) -> str:
    """What the order holds after a cart button, in one line."""
    count = sum(float(item["quantity"]) for item in cart.get("items", []))
    if not count:
        return "Tu pedido quedó vacío."
    noun = "producto" if count == 1 else "productos"
    return f"Llevas {count:g} {noun} · {currency} {cart.get('total', '0.00')}."


def action_reply(
    name: str, result: dict[str, Any], currency: str, *, dish: str = "", quantity: float = 1
) -> str:
    """`dish` is what a cart button acted on: its tap is not echoed as a
    bubble, so the reply names it."""
    if not result.get("ok"):
        messages = {
            "empty_cart": "Tu pedido está vacío. Agrega un plato de la carta.",
            "customer_name_required": "¿A nombre de quién registramos el pedido? Escríbeme tu nombre.",
            "customer_phone_required": (
                "Escríbeme tu número de contacto con código de país, por ejemplo +51 seguido de tu número."
            ),
            "delivery_details_required": (
                "Para continuar, dime si recogerás en el local o deseas delivery "
                "y a nombre de quién va el pedido."
            ),
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
        return f"Tu pedido\n{summary}\n\nTotal: {currency} {total}" + (
            "\nRevisa el resumen y pulsa «Confirmar pedido» para enviarlo."
            if name == "prepare_confirmation"
            else ""
        )
    if dish and name.startswith("add_"):
        return f"Agregué {quantity:g} × {dish}.\n{cart_line(cart, currency)}"
    if dish and name == "remove_item":
        return f"Quité {dish} de tu pedido.\n{cart_line(cart, currency)}"
    return "Actualicé tu pedido. Puedes seguir eligiendo o revisar el resumen."
