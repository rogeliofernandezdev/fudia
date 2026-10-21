"""Short table codes for the QR → WhatsApp start message.

WhatsApp cannot carry hidden data and the scan does not know the diner's
number, so the first message must name its table. Instead of the long QR token
it carries a short code issued at scan time ("Mesa 7 · K7P2") that maps to the
QR token for a few minutes. The code never decides a table on its own: it only
resolves to the QR token, which foods-backend validates as before.
"""

from __future__ import annotations

import re
import secrets
import unicodedata

# No 0/O, 1/I/L: the code is read and sometimes retyped by people.
START_CODE_ALPHABET = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
START_CODE_LENGTH = 4
_LETTERS = "".join(c for c in START_CODE_ALPHABET if c.isalpha())
_DIGITS = "".join(c for c in START_CODE_ALPHABET if c.isdigit())
_CANDIDATE = re.compile(r"(?<![A-Za-z0-9])([A-Za-z0-9]{4})(?![A-Za-z0-9])")
# At most this many code-shaped words are looked up per message.
MAX_CODE_CANDIDATES = 3


def new_start_code() -> str:
    """Random code with at least one letter and one digit, so ordinary
    words ("hola") or quantities ("2") are never mistaken for a code."""
    while True:
        code = "".join(
            secrets.choice(START_CODE_ALPHABET)
            for _ in range(START_CODE_LENGTH)
        )
        if any(c in _LETTERS for c in code) and any(c in _DIGITS for c in code):
            return code


def start_code_candidates(text: str) -> list[str]:
    """Code-shaped words in a message, uppercased, in order of appearance."""
    found: list[str] = []
    for match in _CANDIDATE.finditer(text):
        word = match.group(1).upper()
        if (
            all(c in START_CODE_ALPHABET for c in word)
            and any(c in _LETTERS for c in word)
            and any(c in _DIGITS for c in word)
            and word not in found
        ):
            found.append(word)
        if len(found) == MAX_CODE_CANDIDATES:
            break
    return found


def table_label(name: str) -> str:
    value = " ".join(name.split())
    return value if value.lower().startswith("mesa") else f"Mesa {value}"


def start_message(table_name: str, code: str) -> str:
    return f"¡Hola! 👋 Quiero ver la carta.\n{table_label(table_name)} · {code}"


def looks_like_start_message(text: str) -> bool:
    """The prefilled start message, even if partly edited: used to tell an
    expired code apart from ordinary text that happens to look like one."""
    plain = unicodedata.normalize("NFKD", text).encode("ascii", "ignore").decode()
    lowered = plain.lower()
    return "ver la carta" in lowered or bool(re.search(r"\bmesa\b", lowered))
