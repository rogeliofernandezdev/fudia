import re
import unicodedata


def _normalized(text: str) -> str:
    value = unicodedata.normalize("NFKD", text.lower())
    value = "".join(ch for ch in value if not unicodedata.combining(ch))
    value = re.sub(r"[^a-z0-9 ]+", " ", value)
    return " ".join(value.split())


def explicit_confirmation(text: str) -> bool:
    value = _normalized(text)
    if not value or value.startswith("no"):
        return False
    exact = {
        "si",
        "confirmo",
        "confirmar",
        "de acuerdo",
        "ok",
        "dale",
        "haz el pedido",
        "hacer el pedido",
        "envia el pedido",
        "enviar el pedido",
    }
    return value in exact or value.startswith("si confirm")
