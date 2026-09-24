from __future__ import annotations

import logging
from typing import Any

from fastapi import APIRouter, HTTPException, Request
from fastapi.responses import PlainTextResponse

router = APIRouter(prefix="/webhook", tags=["whatsapp"])
logger = logging.getLogger(__name__)


@router.get("")
async def verify_webhook(request: Request):
    settings = request.app.state.settings
    mode = request.query_params.get("hub.mode")
    token = request.query_params.get("hub.verify_token")
    challenge = request.query_params.get("hub.challenge", "")

    if mode == "subscribe" and settings.whatsapp_verify_token:
        if token == settings.whatsapp_verify_token:
            return PlainTextResponse(challenge)
        raise HTTPException(status_code=403, detail="Verification failed")
    return PlainTextResponse("Fudia Concierge webhook")


def _incoming_text_messages(data: dict[str, Any]) -> list[tuple[str, str, str]]:
    result: list[tuple[str, str, str]] = []
    for entry in data.get("entry", []):
        for change in entry.get("changes", []):
            value = change.get("value", {})
            for message in value.get("messages", []):
                if message.get("type") != "text":
                    continue
                result.append(
                    (
                        str(message.get("id", "")),
                        str(message.get("from", "")),
                        str(message.get("text", {}).get("body", "")),
                    )
                )
    return result


@router.post("")
async def receive_webhook(request: Request) -> dict[str, int | str]:
    data = await request.json()
    accepted = 0
    service = request.app.state.concierge_service
    whatsapp = request.app.state.whatsapp
    dedup = request.app.state.deduplicator

    for message_id, phone, text in _incoming_text_messages(data):
        if not phone or not text:
            continue
        if not await dedup.claim(message_id):
            continue
        accepted += 1
        try:
            reply = await service.handle_message(phone, text)
            await whatsapp.send_text(phone, reply)
        except Exception:
            logger.exception("No se pudo procesar el mensaje %s", message_id)

    return {"status": "processed", "accepted": accepted}
