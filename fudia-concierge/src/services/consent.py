"""Consentimiento promocional por WhatsApp.

Después del primer pedido confirmado, Concierge pregunta una sola vez por
empresa si el cliente quiere recibir ofertas. La pregunta va al final de la
confirmación del pedido y la respuesta se reconoce sin el modelo en el
siguiente mensaje; cualquier otra cosa sigue la conversación normal, sin
interrumpir la compra. foods-backend guarda la pregunta, la respuesta y el
retiro con su fecha y el texto literal del cliente como evidencia.
"""

from __future__ import annotations

import re
import unicodedata
from typing import Literal

CONSENT_YES = "Sí, quiero recibirlos"
CONSENT_NO = "No, gracias"

Answer = Literal["granted", "declined"]

_YES = {
    "si",
    "si quiero",
    "si quiero recibirlos",
    "si porfavor",
    "si por favor",
    "si claro",
    "claro",
    "claro que si",
    "acepto",
    "quiero recibirlos",
    "ok",
    "okey",
    "dale",
}
_NO = {
    "no",
    "no gracias",
    "no quiero",
    "no deseo",
    "no por ahora",
    "ahora no",
    "no me interesa",
}
_WITHDRAW = re.compile(
    r"\b(no\s+(quiero|deseo)(\s+recibir)?(\s+mas)?|ya\s+no\s+quiero(\s+recibir)?"
    r"|dejar\s+de\s+recibir|darme\s+de\s+baja|dar(me)?\s+de\s+baja"
    r"|no\s+me\s+(envies|envien|mandes|manden)(\s+mas)?)\s+"
    r"(de\s+)?(las\s+|los\s+|mas\s+)?(ofertas|promociones|promos|publicidad|beneficios)\b"
)


def _plain(text: str) -> str:
    value = unicodedata.normalize("NFKD", text).encode("ascii", "ignore").decode()
    value = re.sub(r"[^a-z0-9\s]", " ", value.lower())
    return " ".join(value.split())


def consent_question(organization_name: str) -> str:
    """The question exactly as the restaurant asked for it, in plain text."""
    name = organization_name.strip() or "este restaurante"
    return (
        "¿Te gustaría recibir ofertas, promociones y beneficios exclusivos "
        f"de {name} por WhatsApp?\n\n- {CONSENT_YES}\n- {CONSENT_NO}"
    )


def consent_answer(text: str) -> Answer | None:
    """The diner's reply to the question, or None when the message is about
    something else (then the conversation simply continues)."""
    value = _plain(text)
    if value in _YES:
        return "granted"
    if value in _NO:
        return "declined"
    return None


def asks_to_withdraw(text: str) -> bool:
    """«Ya no quiero promociones», «darme de baja de las ofertas»…"""
    return bool(_WITHDRAW.search(_plain(text)))


def consent_reply(decision: str, organization_name: str) -> str:
    name = organization_name.strip() or "este restaurante"
    if decision == "granted":
        return (
            f"Listo, te enviaremos ofertas y beneficios de {name} por "
            "WhatsApp. Si en algún momento ya no los quieres, escríbeme "
            "«ya no quiero promociones»."
        )
    if decision == "withdrawn":
        return f"Listo, ya no te enviaremos promociones de {name} por WhatsApp."
    return "Entendido, no te enviaremos promociones."
