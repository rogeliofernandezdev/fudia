"""Private provider management and normalized whatsmeow ingress."""

from __future__ import annotations

import hmac
import json
import re
import time
from typing import Any, Literal
from uuid import UUID

from fastapi import APIRouter, HTTPException, Request
from pydantic import BaseModel, Field, ValidationError

from src.domain.models import GeoPoint, InboundMessage
from src.infrastructure.whatsapp_providers import WhatsAppProviders
from src.services.delivery_partners import parse_payload

router = APIRouter(prefix="/internal/whatsapp", tags=["internal-whatsapp"])


async def private_body(request: Request, *, gateway: bool = False) -> dict[str, Any]:
    cfg = request.app.state.settings
    expected = cfg.whatsapp_gateway_ingress_key.get_secret_value() if gateway else cfg.fudia_concierge_api_key
    header = "X-Fudia-Gateway-Key" if gateway else "X-Fudia-Concierge-Key"
    provided = request.headers.get(header, "")
    if not expected:
        raise HTTPException(503, "El servicio de WhatsApp no está configurado.")
    if not hmac.compare_digest(provided.encode(), expected.encode()):
        raise HTTPException(401, "Credencial interna inválida.")
    raw = bytearray()
    async for chunk in request.stream():
        raw.extend(chunk)
        if len(raw) > cfg.webhook_max_payload_bytes:
            raise HTTPException(413, "El mensaje supera el límite permitido.")
    try:
        body = json.loads(raw)
        if not isinstance(body, dict):
            raise ValueError
    except (ValueError, json.JSONDecodeError) as exc:
        raise HTTPException(400, "Solicitud interna inválida.") from exc
    return body


def manager(request: Request) -> WhatsAppProviders:
    provider = request.app.state.whatsapp
    if not isinstance(provider, WhatsAppProviders):
        raise HTTPException(503, "El transporte multiproveedor no está configurado.")
    return provider


@router.post("/verify")
async def verify_provider(request: Request) -> dict[str, Any]:
    body = await private_body(request)
    try:
        country = str(body.get("countryCode", ""))
        if not re.fullmatch(r"[A-Z]{2}", country):
            raise ValueError
        return await manager(request).verify(
            country,
            str(body.get("provider", "META_CLOUD_API")),
            str(body.get("phoneNumber", "")),
            str(body.get("sessionRef", "")),
        )
    except Exception as exc:
        raise HTTPException(503, "No pudimos verificar el proveedor y el número seleccionado.") from exc


@router.post("/link")
async def link_provider(request: Request) -> dict[str, Any]:
    body = await private_body(request)
    try:
        result = await manager(request).link(
            str(body.get("countryCode", "")), str(body.get("sessionRef", ""))
        )
    except Exception as exc:
        raise HTTPException(503, "No pudimos consultar la vinculación de WhatsApp.") from exc
    return result


class GatewayEvent(BaseModel):
    type: Literal["message", "status"]
    provider: Literal["WHATSMEOW"]
    sessionRef: UUID
    countryCode: str = Field(pattern=r"^[A-Z]{2}$")
    externalId: str = Field(default="", max_length=200)
    sender: str = Field(default="", max_length=160)
    verifiedPhone: str = Field(default="", max_length=15)
    text: str = Field(default="", max_length=8000)
    location: GeoPoint | None = None
    buttonPayload: str = Field(default="", max_length=300)
    timestamp: int = Field(ge=0)
    callbackRef: str = ""
    state: str = ""


@router.post("/gateway/events")
async def gateway_event(request: Request) -> dict[str, Any]:
    body = await private_body(request, gateway=True)
    try:
        event = GatewayEvent.model_validate(body)
        if event.timestamp > time.time() + 300:
            raise ValueError
    except (ValidationError, ValueError) as exc:
        raise HTTPException(400, "Evento de WhatsApp inválido.") from exc
    providers = manager(request)
    await providers.routes(fresh=True)
    action = parse_payload(event.buttonPayload) if event.buttonPayload else None
    route = await providers.current_gateway(
        str(event.sessionRef), historical=event.type == "status" or action is not None
    )
    if route is None or route.countryCode != event.countryCode:
        return {"status": "ignored"}
    if event.type == "status":
        if not re.fullmatch(r"[0-9a-f]{64}", event.callbackRef) or event.state not in {
            "sent",
            "delivered",
            "read",
            "failed",
        }:
            raise HTTPException(400, "Acuse inválido.")
        await request.app.state.conversation_store.record_outbound_status(event.callbackRef, event.state)
        return {"status": "accepted"}
    if (
        not event.externalId
        or not re.fullmatch(r"[0-9]+@(s\.whatsapp\.net|lid)", event.sender)
        or (event.verifiedPhone and not re.fullmatch(r"[1-9][0-9]{7,14}", event.verifiedPhone))
    ):
        raise HTTPException(400, "Identidad de WhatsApp inválida.")
    if event.sender.endswith("@s.whatsapp.net") and event.verifiedPhone != event.sender.split("@", 1)[0]:
        raise HTTPException(400, "La identidad no coincide con el teléfono verificado.")
    text = event.text or ("📍 Compartí mi ubicación." if event.location else "")
    if not text:
        return {"status": "ignored"}
    message = InboundMessage(
        message_id=f"WHATSMEOW:{event.sessionRef}:{event.externalId}",
        phone=event.sender,
        text=text,
        sender_phone_id=route.routingKey,
        provider="WHATSMEOW",
        country_code=route.countryCode,
        subject=event.sender,
        verified_phone=event.verifiedPhone,
        sent_at=event.timestamp,
        location=event.location,
        partner_action=action,
    )
    queued = await request.app.state.inbound_queue.enqueue(message)
    return {"status": "accepted", "accepted": int(queued)}
