"""Inspect and recover dead-lettered Concierge messages.

Never prints phones, message text, QR tokens or replies: only fingerprints,
attempts, error types and the recovery stage of each message.

    python scripts/dlq.py list [--count 50]
    python scripts/dlq.py show ENTRY_ID
    python scripts/dlq.py requeue ENTRY_ID --yes
    python scripts/dlq.py discard ENTRY_ID --yes

`requeue` moves the entry back to the stream with its original message id.
The processing record decides what happens next: an already prepared reply is
only sent, a pending backend operation is replayed with its original
idempotency key, and only a message that never changed anything runs the
model again.
"""

from __future__ import annotations

import argparse
import asyncio
import sys
from datetime import UTC, datetime
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from config.settings import settings  # noqa: E402
from src.infrastructure.queue import DeadLetter, RedisInboundQueue  # noqa: E402
from src.infrastructure.state_store import RedisConversationStore  # noqa: E402
from src.observability import fingerprint  # noqa: E402

NEXT_STEP = {
    None: "sin registro: se procesará de nuevo con el modelo",
    "processing": "sin efectos persistidos: se procesará de nuevo con el modelo",
    "effects_applied": "respuesta determinista desde los efectos guardados",
    "effect_pending": "se reintentará la operación con su misma clave idempotente",
    "reply_ready": "solo se enviará la respuesta ya preparada",
    "sending": "se reconciliará el envío incierto",
    "uncertain": "se reconciliará el envío incierto",
    "sent": "ya enviado: se confirmará sin reenviar",
    "failed": "envío marcado como fallido: no se reenviará",
    "superseded": "contexto reemplazado por otro QR: no se aplicará",
}


async def describe(
    letter: DeadLetter,
    store: RedisConversationStore,
    detailed: bool = False,
) -> str:
    record = await store.get_record(letter.message.message_id)
    stage = record.stage if record else None
    failed_at = (
        datetime.fromtimestamp(letter.failed_at_ms / 1000, UTC).isoformat()
        if letter.failed_at_ms
        else "-"
    )
    lines = [
        f"{letter.entry_id}  message={fingerprint(letter.message.message_id)}"
        f"  attempts={letter.attempt}  seq={letter.sequence}"
        f"  failedAt={failed_at}  error={letter.error or '-'}"
        f"  stage={stage or '-'}",
        f"    siguiente paso: {NEXT_STEP.get(stage, stage or '-')}",
    ]
    if detailed and record is not None:
        lines.append(
            f"    attempts={record.attempts}  sendAttempts={record.send_attempts}"
            f"  resends={record.resends}  lastError={record.last_error or '-'}"
        )
        for effect in record.effects:
            lines.append(
                f"    effect {effect.kind}: {effect.status}"
                f"  requestId={fingerprint(effect.request_id) or '-'}"
                f"  error={effect.error_code or '-'}"
            )
    return "\n".join(lines)


async def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    commands = parser.add_subparsers(dest="command", required=True)
    listing = commands.add_parser("list")
    listing.add_argument("--count", type=int, default=50)
    for name in ("show", "requeue", "discard"):
        command = commands.add_parser(name)
        command.add_argument("entry_id")
        if name != "show":
            command.add_argument("--yes", action="store_true")
    args = parser.parse_args(argv)

    queue = RedisInboundQueue(
        settings.redis_url,
        stream=settings.queue_stream,
        group=settings.queue_group,
        dedup_ttl_seconds=86_400,
    )
    store = RedisConversationStore(settings.redis_url, settings.session_ttl_seconds)
    try:
        letters = await queue.dead_letters(
            args.count if args.command == "list" else 10_000
        )
        if args.command == "list":
            for letter in letters:
                print(await describe(letter, store))
            print(f"{len(letters)} mensaje(s) en dead-letter")
            return 0

        target = next(
            (item for item in letters if item.entry_id == args.entry_id), None
        )
        if target is None:
            print("No existe esa entrada en dead-letter.", file=sys.stderr)
            return 1
        print(await describe(target, store, detailed=True))
        if args.command == "show":
            return 0
        if not args.yes:
            print("Revisa el siguiente paso y repite con --yes para confirmar.")
            return 2
        if args.command == "requeue":
            moved = await queue.requeue_dead(target.entry_id)
            print("Reencolado." if moved else "La entrada ya no existe.")
            return 0 if moved else 1
        removed = await queue.discard_dead(target.entry_id)
        print("Descartado." if removed else "La entrada ya no existe.")
        return 0 if removed else 1
    finally:
        await queue.close()
        await store.close()


if __name__ == "__main__":
    raise SystemExit(asyncio.run(main(sys.argv[1:])))
