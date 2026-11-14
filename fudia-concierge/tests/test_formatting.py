from src.services.formatting import whatsapp_text


def test_markdown_carta_becomes_plain_headings_and_dashes() -> None:
    reply = (
        "¡Hola! 👋 Aquí tienes la carta disponible en el local:\n\n"
        "*Entradas*\n"
        "* Causa Rellena: S/ 20.00\n\n"
        "**Segundos**\n"
        "• Ají de gallina: S/ 15.00\n"
        "  - Lomo Saltado: S/ 15.00\n\n\n"
        "¿Qué deseas pedir? 😊"
    )

    assert whatsapp_text(reply) == (
        "¡Hola! Aquí tienes la carta disponible en el local:\n\n"
        "Entradas:\n"
        "- Causa Rellena: S/ 20.00\n\n"
        "Segundos:\n"
        "- Ají de gallina: S/ 15.00\n"
        "- Lomo Saltado: S/ 15.00\n\n"
        "¿Qué deseas pedir?"
    )


def test_sober_icons_and_plain_sentences_are_kept() -> None:
    reply = "✅ Tu pedido PED-001 fue enviado a cocina.\n\n🧾 Total: S/ 30.00"

    assert whatsapp_text(reply) == reply


def test_markdown_headings_and_playful_icons_are_removed() -> None:
    reply = "## Postres 🎉\n- Suspiro limeño ⭐: S/ 12.00\n\n¡Gracias! ❤️🔥"

    assert whatsapp_text(reply) == (
        "Postres:\n- Suspiro limeño: S/ 12.00\n\n¡Gracias!"
    )


def test_reference_quotes_are_removed_and_web_keeps_action_emphasis() -> None:
    reply = "Revisa tu pedido y toca **Confirmar pedido** para enviarlo."
    assert whatsapp_text(reply, allow_bold=True) == reply
    assert whatsapp_text(reply) == "Revisa tu pedido y toca Confirmar pedido para enviarlo."
    assert whatsapp_text("Abre «Mi pedido». Responde «sí».", allow_bold=True) == (
        "Abre Mi pedido. Responde sí."
    )
    assert whatsapp_text("## Opciones\n* Agua\nToca **Carta**. *incompleto", allow_bold=True) == (
        "Opciones:\n- Agua\nToca **Carta**. incompleto"
    )
