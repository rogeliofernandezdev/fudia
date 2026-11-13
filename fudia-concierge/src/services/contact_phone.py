"""Contact data for an order; a typed number never proves customer identity."""

from __future__ import annotations

import re


def phone_text(text: str) -> bool:
    return bool(re.fullmatch(r"\+?[0-9][0-9 ()-]*", text.strip()))


def normalize_contact_phone(text: str, country: str) -> str:
    """Accept Peru's national mobile format or an explicit international number.

    Without known national rules, require an explicit international prefix
    rather than assigning the restaurant's country to an ambiguous number.
    """
    value = text.strip()
    if not phone_text(value):
        return ""
    digits = re.sub(r"[^0-9]", "", value)
    if value.startswith("+"):
        return digits if re.fullmatch(r"[1-9][0-9]{7,14}", digits) else ""
    if country == "PE":
        if re.fullmatch(r"9[0-9]{8}", digits):
            return "51" + digits
        if re.fullmatch(r"519[0-9]{8}", digits):
            return digits
    return ""


def contact_phone_prompt(country: str, *, invalid: bool = False) -> str:
    if country == "PE":
        return (
            "Escribe un número de celular válido de 9 dígitos. "
            "Si es de otro país, incluye + y su código de país."
            if invalid
            else "¿Cuál es tu número de celular de contacto? Puedes escribir los 9 dígitos."
        )
    return "Escríbeme tu número de contacto con + y su código de país."
