from __future__ import annotations

import hashlib
import hmac
import json
import logging
from typing import Any

from fastapi import APIRouter, BackgroundTasks, HTTPException, Request
from fastapi.responses import PlainTextResponse

router = APIRouter(prefix="/webhook", tags=["whatsapp"])
logger = logging.getLogger(__name__)


def _valid_meta_signature(payload: bytes, signature: str, secret: str) -> bool:
    if not secret or not signature.startswith("sha256="):
        return False
    expected = "sha256=" + hmac.new(
        secret.encode("utf-8"), payload, hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, signature)


async def _process_message(
    service: Any,
    whatsapp: Any,
    phone: str,
    text: str,
    message_id: str,
) -> None:
    try:
        reply = await service.handle_message(phone, text)
        await whatsapp.send_text(phone, reply)
    except Exception:
        logger.exception("No se pudo procesar el mensaje %s", message_id)


@router.get("")
async def verify_webhook(request: Request) -> PlainTextResponse:
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
async def receive_webhook(
    request: Request,
    background_tasks: BackgroundTasks,
) -> dict[str, int | str]:
    payload = await request.body()
    signature = request.headers.get("X-Hub-Signature-256", "")
    secret = request.app.state.settings.whatsapp_app_secret
    if not secret:
        raise HTTPException(status_code=503, detail="WhatsApp app secret is not configured")
    if not _valid_meta_signature(payload, signature, secret):
        raise HTTPException(status_code=403, detail="Invalid webhook signature")

    try:
        data = json.loads(payload)
    except json.JSONDecodeError as exc:
        raise HTTPException(status_code=400, detail="Invalid JSON") from exc

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
        background_tasks.add_task(
            _process_message,
            service,
            whatsapp,
            phone,
            text,
            message_id,
        )

    return {"status": "accepted", "accepted": accepted}

