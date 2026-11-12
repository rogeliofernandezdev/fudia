from __future__ import annotations

import hashlib
import hmac
import json
import re
import time
from typing import Any

from fastapi import APIRouter, HTTPException, Request
from fastapi.responses import PlainTextResponse
from pydantic import ValidationError

from src.domain.models import GeoPoint, InboundMessage, PartnerAction
from src.metrics import INBOUND_MESSAGES
from src.services.delivery_partners import parse_payload

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
                # Los botones de una central de Delivery Fudia no son turnos de chat.
                if _button_partner_action(kind, content) is not None:
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


LOCATION_TEXT = "📍 Compartí mi ubicación."


def _button_partner_action(kind: Any, content: dict[str, Any]) -> PartnerAction | None:
    """Botón de plantilla (payload) o interactivo (id) con el formato de Delivery Fudia."""
    raw = content.get("payload") if kind == "button" else content.get("id")
    return parse_payload(raw) if isinstance(raw, str) else None


def _incoming_partner_actions(
    data: dict[str, Any],
) -> list[tuple[str, str, PartnerAction, str, str]]:
    """Aceptar, Rechazar o Recoger de una central: (id, from, acción, título, phone_id)."""
    result: list[tuple[str, str, PartnerAction, str, str]] = []
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
            sender_phone_id = (
                str(metadata.get("phone_number_id", ""))
                if isinstance(metadata, dict)
                else ""
            )
            messages = value.get("messages", [])
            for message in messages if isinstance(messages, list) else []:
                if not isinstance(message, dict):
                    continue
                kind = message.get("type")
                content: Any = {}
                if kind == "button":
                    content = message.get("button", {})
                    title_field = "text"
                elif kind == "interactive":
                    interactive = message.get("interactive", {})
                    if not isinstance(interactive, dict) or interactive.get("type") != "button_reply":
                        continue
                    content = interactive.get("button_reply", {})
                    title_field = "title"
                else:
                    continue
                if not isinstance(content, dict):
                    continue
                action = _button_partner_action(kind, content)
                if action is None:
                    continue
                title = content.get(title_field, "")
                result.append((
                    str(message.get("id", "")),
                    str(message.get("from", "")),
                    action,
                    title.strip() if isinstance(title, str) and title.strip() else action.action,
                    sender_phone_id,
                ))
    return result


def _incoming_locations(
    data: dict[str, Any],
) -> list[tuple[str, str, GeoPoint, str]]:
    """WhatsApp location pins: the delivery point used to quote the fee."""
    result: list[tuple[str, str, GeoPoint, str]] = []
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
            sender_phone_id = (
                str(metadata.get("phone_number_id", ""))
                if isinstance(metadata, dict)
                else ""
            )
            messages = value.get("messages", [])
            for message in messages if isinstance(messages, list) else []:
                if not isinstance(message, dict) or message.get("type") != "location":
                    continue
                content = message.get("location", {})
                if not isinstance(content, dict):
                    continue
                try:
                    point = GeoPoint(
                        latitude=float(content.get("latitude", "")),
                        longitude=float(content.get("longitude", "")),
                        name=str(content.get("name", ""))[:160],
                        address=str(content.get("address", ""))[:240],
                    )
                except (TypeError, ValueError, ValidationError):
                    continue
                result.append((
                    str(message.get("id", "")),
                    str(message.get("from", "")),
                    point,
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

    if not isinstance(data, dict):
        raise HTTPException(400, "Invalid webhook payload")
    providers = request.app.state.whatsapp
    meta_resolver = getattr(providers, "current_meta", None)
    timestamps: dict[str, float] = {}
    if meta_resolver is not None:
        await providers.routes(fresh=True)
        entries = data.get("entry", []) if isinstance(data, dict) else []
        for entry in entries if isinstance(entries, list) else []:
            changes = entry.get("changes", []) if isinstance(entry, dict) else []
            for change in changes if isinstance(changes, list) else []:
                value = change.get("value", {}) if isinstance(change, dict) else {}
                messages = value.get("messages", []) if isinstance(value, dict) else []
                for item in messages if isinstance(messages, list) else []:
                    if not isinstance(item, dict):
                        continue
                    try:
                        timestamps[str(item.get("id", ""))] = float(item.get("timestamp", 0))
                    except (ValueError, TypeError):
                        continue
    accepted = 0
    queue = request.app.state.inbound_queue
    incoming: list[tuple[str, str, str, str, GeoPoint | None, PartnerAction | None]] = [
        (message_id, phone, text, sender_phone_id, None, None)
        for message_id, phone, text, sender_phone_id
        in _incoming_text_messages(data)
    ]
    incoming.extend(
        (message_id, phone, LOCATION_TEXT, sender_phone_id, point, None)
        for message_id, phone, point, sender_phone_id
        in _incoming_locations(data)
    )
    incoming.extend(
        (message_id, phone, title, sender_phone_id, None, action)
        for message_id, phone, action, title, sender_phone_id
        in _incoming_partner_actions(data)
    )
    for (
        message_id,
        phone,
        text,
        sender_phone_id,
        location,
        partner_action,
    ) in incoming:
        if not message_id or not phone or not text:
            continue
        extra: dict[str, Any] = {}
        channel_key = sender_phone_id
        if meta_resolver is not None:
            route = await meta_resolver(sender_phone_id)
            historical = getattr(providers, "historical_meta", None)
            if route is None and partner_action is not None and historical is not None:
                route = await historical(sender_phone_id)
            sent_at = timestamps.get(message_id, 0)
            if (route is None or not re.fullmatch(r"[1-9][0-9]{7,14}", phone)
                    or sent_at <= 0 or sent_at > time.time() + 300):
                INBOUND_MESSAGES.labels(result="wrong_phone_id").inc()
                continue
            extra = {"legacy_message_id": message_id, "provider": "META_CLOUD_API",
                     "country_code": route.countryCode, "subject": phone,
                     "verified_phone": phone, "sent_at": sent_at}
            message_id = f"META_CLOUD_API:{sender_phone_id}:{message_id}"
            channel_key = route.routingKey
        elif not _matches_configured_phone_id(
            sender_phone_id, request.app.state.settings.whatsapp_phone_id.strip(),
        ):
            INBOUND_MESSAGES.labels(result="wrong_phone_id").inc()
            continue
        queued = await queue.enqueue(
            InboundMessage(
                message_id=message_id, phone=phone, text=text, sender_phone_id=channel_key,
                location=location, partner_action=partner_action, **extra,
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
    status_accounts = [request.app.state.settings.whatsapp_phone_id.strip()]
    if meta_resolver is not None:
        status_accounts = list(providers.accounts)
    for account_id in status_accounts:
        for ref, state in _outbound_statuses(data, account_id):
            await store.record_outbound_status(ref, state)
            INBOUND_MESSAGES.labels(result=f"status_{state}").inc()

    return {
        "status": "accepted",
        "accepted": accepted,
    }
