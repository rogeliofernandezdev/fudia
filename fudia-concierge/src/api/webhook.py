from __future__ import annotations

import hashlib
import hmac
import json
import re
from typing import Any

from fastapi import APIRouter, HTTPException, Request
from fastapi.responses import PlainTextResponse

from src.domain.models import InboundMessage
from src.metrics import INBOUND_MESSAGES

# Our own `message_ref` (SHA-256 hex) echoed back by Meta status webhooks.
CALLBACK_REF_PATTERN = re.compile(r"^[0-9a-f]{64}$")
OUTBOUND_STATUSES = frozenset({"sent", "delivered", "read", "failed"})

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
    """Only user-authored text, quick replies and list choices enter the queue.

    Delivery receipts, template statuses and unsupported media never become
    chat turns. Malformed webhook shapes are ignored safely.
    """
    result: list[tuple[str, str, str, str]] = []
    entries = data.get("entry", [])
    if not isinstance(entries, list):
        return result
    for entry in entries:
        if not isinstance(entry, dict):
            continue
        changes = entry.get("changes", [])
        if not isinstance(changes, list):
            continue
        for change in changes:
            if not isinstance(change, dict):
                continue
            value = change.get("value", {})
            if not isinstance(value, dict):
                continue
            metadata = value.get("metadata", {})
            if not isinstance(metadata, dict):
                metadata = {}
            sender_phone_id = str(metadata.get("phone_number_id", ""))
            messages = value.get("messages", [])
            if not isinstance(messages, list):
                continue
            for message in messages:
                if not isinstance(message, dict):
                    continue
                kind = message.get("type")
                content: Any = {}
                if kind == "text":
                    content = message.get("text", {})
                    field = "body"
                elif kind == "button":
                    content = message.get("button", {})
                    field = "text"
                elif kind == "interactive":
                    interactive = message.get("interactive", {})
                    if not isinstance(interactive, dict):
                        continue
                    subtype = interactive.get("type")
                    if subtype not in {"button_reply", "list_reply"}:
                        continue
                    content = interactive.get(subtype, {})
                    field = "title"
                else:
                    continue
                if not isinstance(content, dict):
                    continue
                body = content.get(field, "")
                if not isinstance(body, str) or not body.strip():
                    continue
                result.append((
                    str(message.get("id", "")),
                    str(message.get("from", "")),
                    body.strip(),
                    sender_phone_id,
                ))
    return result

def _outbound_statuses(
    data: dict[str, Any],
    configured_phone_id: str,
) -> list[tuple[str, str]]:
    """Status receipts of replies sent with `biz_opaque_callback_data`.

    They only reconcile sends whose outcome was uncertain; they never become
    chat turns. Unknown references and other phone ids are ignored.
    """
    result: list[tuple[str, str]] = []
    entries = data.get("entry", [])
    if not isinstance(entries, list):
        return result
    for entry in entries:
        changes = entry.get("changes", []) if isinstance(entry, dict) else []
        for change in changes if isinstance(changes, list) else []:
            value = change.get("value", {}) if isinstance(change, dict) else {}
            if not isinstance(value, dict):
                continue
            metadata = value.get("metadata", {})
            phone_id = (
                str(metadata.get("phone_number_id", ""))
                if isinstance(metadata, dict)
                else ""
            )
            if not _matches_configured_phone_id(phone_id, configured_phone_id):
                continue
            statuses = value.get("statuses", [])
            for status in statuses if isinstance(statuses, list) else []:
                if not isinstance(status, dict):
                    continue
                ref = str(status.get("biz_opaque_callback_data", ""))
                state = str(status.get("status", ""))
                if (
                    CALLBACK_REF_PATTERN.fullmatch(ref)
                    and state in OUTBOUND_STATUSES
                ):
                    result.append((ref, state))
    return result


@router.post("")
async def receive_webhook(
    request: Request,
) -> dict[str, int | str]:
    maximum = request.app.state.settings.webhook_max_payload_bytes
    declared = request.headers.get("content-length", "")
    if declared.isdecimal() and int(declared) > maximum:
        raise HTTPException(status_code=413, detail="Payload too large")
    payload = await request.body()
    if len(payload) > maximum:
        raise HTTPException(status_code=413, detail="Payload too large")
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

    store = request.app.state.conversation_store
    for ref, state in _outbound_statuses(
        data,
        request.app.state.settings.whatsapp_phone_id.strip(),
    ):
        await store.record_outbound_status(ref, state)
        INBOUND_MESSAGES.labels(result=f"status_{state}").inc()

    return {
        "status": "accepted",
        "accepted": accepted,
    }
