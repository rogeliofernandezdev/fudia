from __future__ import annotations

import re

# Playful emoji the replies never use: faces, hands, hearts, stars, sparkles,
# fire and celebrations. Sober icons (🍽️ 🧾 ✅ 📍 🛵 ⏳) are kept.
PLAYFUL_EMOJI = re.compile(
    "["
    "\U0001f600-\U0001f64f"  # faces and gesturing people
    "\U0001f910-\U0001f92f\U0001f970-\U0001f97a\U0001f9d0"  # more faces
    "\U0001f440-\U0001f450\U0001f590-\U0001f596\U0001f918-\U0001f91f"  # hands
    "\U0001f493-\U0001f49f❤♥"  # hearts
    "⭐\U0001f31f✨\U0001f525\U0001f389\U0001f38a\U0001f973"
    "]️?"
)
BULLET = re.compile(r"^\s*(?:[*•·+▪◦–]|-)\s+")
HEADING = re.compile(r"^\s*#{1,6}\s+")


def whatsapp_text(text: str) -> str:
    """Model replies as plain WhatsApp text: no markdown marks, «- » bullets,
    list headings ending in «:» and no playful emoji."""
    lines: list[str] = []
    for raw in text.replace("\r\n", "\n").split("\n"):
        bullet = bool(BULLET.match(raw))
        line = BULLET.sub("", raw) if bullet else HEADING.sub("", raw)
        line = line.replace("*", "").replace("__", "")
        cleaned = PLAYFUL_EMOJI.sub("", line)
        if cleaned != line:
            # A removed icon must not leave a gap before punctuation.
            cleaned = re.sub(r"[ \t]+([:,.!?])", r"\1", cleaned)
        line = re.sub(r"[ \t]{2,}", " ", cleaned).strip()
        if bullet and line:
            line = f"- {line}"
        lines.append(line)

    for index, line in enumerate(lines):
        following = next((item for item in lines[index + 1 :] if item), "")
        if (
            line
            and not line.startswith("- ")
            and following.startswith("- ")
            and len(line) <= 60
            and line[-1] not in ":.?!"
        ):
            lines[index] = f"{line}:"

    joined = "\n".join(lines)
    return re.sub(r"\n{3,}", "\n\n", joined).strip()
