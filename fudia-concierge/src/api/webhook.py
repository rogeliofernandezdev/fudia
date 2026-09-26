from __future__ import annotations

import hashlib
import hmac
import json
from typing import Any

from fastapi import APIRouter, HTTPException, Request
from fastapi.responses import PlainTextResponse

from src.domain.models import InboundMessage
from src.metrics import INBOUND_MESSAGES

router = APIRouter(
    prefix="/webhook",
    tags=["whatsapp"],
)


def _valid_meta_signature(
    payload: bytes,
    signature: str,
    secret: str,
) -> bool:
    if (
        not secret
        or not signature.startswith("sha256=")
    ):
        return False
    expected = "sha256=" + hmac.new(
        secret.encode("utf-8"),
        payload,
        hashlib.sha256,
    ).hexdigest()
    return hmac.compare_digest(
        expected,
        signature,
    )


@router.get("")
async def verify_webhook(
    request: Request,
) -> PlainTextResponse:
    settings = request.app.state.settings
    mode = request.query_params.get("hub.mode")
    token = request.query_params.get(
        "hub.verify_token"
    )
    challenge = request.query_params.get(
        "hub.challenge",
        "",
    )

    if (
        mode == "subscribe"
        and settings.whatsapp_verify_token
    ):
        if token == settings.whatsapp_verify_token:
            return PlainTextResponse(challenge)
        raise HTTPException(
            status_code=403,
            detail="Verification failed",
        )
    return PlainTextResponse(
        "Fudia Concierge webhook"
    )


def _matches_configured_phone_id(
    incoming_phone_id: str,
    configured_phone_id: str,
) -> bool:
    configured = configured_phone_id.strip()
    return (
        not configured
        or incoming_phone_id == configured
    )


def _incoming_text_messages(
    data: dict[str, Any],
) -> list[tuple[str, str, str, str]]:
    result: list[
        tuple[str, str, str, str]
    ] = []
    for entry in data.get("entry", []):
        for change in entry.get("changes", []):
            value = change.get("value", {})
            metadata = value.get("metadata", {})
            sender_phone_id = str(
                metadata.get(
                    "phone_number_id",
                    "",
                )
            )
            for message in value.get(
                "messages",
                [],
            ):
                if message.get("type") != "text":
                    continue
                result.append(
                    (
                        str(message.get("id", "")),
                        str(message.get("from", "")),
                        str(
                            message.get(
                                "text",
                                {},
                            ).get("body", "")
                        ),
                        sender_phone_id,
                    )
                )
    return result


@router.post("")
async def receive_webhook(
    request: Request,
) -> dict[str, int | str]:
    payload = await request.body()
    signature = request.headers.get(
        "X-Hub-Signature-256",
        "",
    )
    secret = (
        request.app.state.settings
        .whatsapp_app_secret
    )
    if not secret:
        raise HTTPException(
            status_code=503,
            detail=(
                "WhatsApp app secret "
                "is not configured"
            ),
        )
    if not _valid_meta_signature(
        payload,
        signature,
        secret,
    ):
        raise HTTPException(
            status_code=403,
            detail="Invalid webhook signature",
        )

    try:
        data = json.loads(payload)
    except json.JSONDecodeError as exc:
        raise HTTPException(
            status_code=400,
            detail="Invalid JSON",
        ) from exc

    accepted = 0
    queue = request.app.state.inbound_queue
    for (
        message_id,
        phone,
        text,
        sender_phone_id,
    ) in _incoming_text_messages(data):
        if not message_id or not phone or not text:
            continue
        configured_phone_id = (
            request.app.state.settings
            .whatsapp_phone_id.strip()
        )
        if not _matches_configured_phone_id(
            sender_phone_id,
            configured_phone_id,
        ):
            INBOUND_MESSAGES.labels(
                result="wrong_phone_id"
            ).inc()
            continue
        queued = await queue.enqueue(
            InboundMessage(
                message_id=message_id,
                phone=phone,
                text=text,
                sender_phone_id=sender_phone_id,
            )
        )
        if queued:
            accepted += 1
            INBOUND_MESSAGES.labels(
                result="queued"
            ).inc()
        else:
            INBOUND_MESSAGES.labels(
                result="duplicate"
            ).inc()

    return {
        "status": "accepted",
        "accepted": accepted,
    }
