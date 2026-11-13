"""Private browser BFF API. Never accept organization, location or prices.

The BFF supplies its authenticated visitor cookie, never a browser phone claim.
The QR is resolved by Go; an opaque Redis capability binds every later request.
"""

from __future__ import annotations

import hashlib
import re
import time
from collections.abc import Callable, Coroutine
from typing import Any, Literal

import httpx
import redis.exceptions
from fastapi import APIRouter, HTTPException, Request, Response
from fastapi.routing import APIRoute
from pydantic import BaseModel, Field, ValidationError

from src.api.whatsapp_providers import manager, private_body
from src.domain.models import TERMINAL_STAGES, ConversationSession, GeoPoint, InboundMessage, TableContext
from src.domain.web_chat import WebAction, WebOrderSummary
from src.services.delivery_tools import delivery_summary
from src.services.ports import FudiaError
from src.services.tooling import cart_payload


class WebChatRoute(APIRoute):
    def get_route_handler(self) -> Callable[[Request], Coroutine[Any, Any, Response]]:
        original = super().get_route_handler()

        async def handle(request: Request) -> Response:
            try:
                return await original(request)
            except FudiaError as exc:
                raise HTTPException(exc.status_code, exc.message) from exc
            except (httpx.HTTPError, redis.exceptions.RedisError, RuntimeError) as exc:
                raise HTTPException(503, "El chat no está disponible. Inténtalo de nuevo.") from exc

        return handle


router = APIRouter(prefix="/internal/web-chat", tags=["internal-web-chat"], route_class=WebChatRoute)


class WebRequest(BaseModel):
    qr: str = Field(pattern=r"^[a-f0-9]{32}$")
    visitor: str = Field(pattern=r"^[A-Za-z0-9_-]{43}$")
    capability: str = Field(default="", max_length=100)
    requestId: str = Field(default="", max_length=80)
    text: str = Field(default="", max_length=2000)
    action: WebAction | None = None
    location: GeoPoint | None = None
    productId: str = Field(default="", max_length=80)
    # Reference of the QR scan that opened the page; a reload repeats it.
    scan: str = Field(default="", pattern=r"^[A-Za-z0-9_-]{0,40}$")


def request_document() -> dict[str, Any]:
    schema = WebRequest.model_json_schema()
    definitions = schema.pop("$defs", {})

    def inline(value: Any) -> Any:
        if isinstance(value, dict):
            if "$ref" in value:
                return inline(definitions[value["$ref"].split("/")[-1]])
            return {key: inline(item) for key, item in value.items()}
        if isinstance(value, list):
            return [inline(item) for item in value]
        return value

    return {
        "requestBody": {"required": True, "content": {"application/json": {"schema": inline(schema)}}},
        "parameters": [
            {"name": "X-Fudia-Concierge-Key", "in": "header", "required": True, "schema": {"type": "string"}}
        ],
    }


WEB_REQUEST_DOC = request_document()


class WebChatState(BaseModel):
    context: TableContext
    messages: list[dict[str, Any]]
    cart: dict[str, Any]
    orderSummary: WebOrderSummary
    confirmationRef: str | None
    mode: Literal["table", "counter", "delivery"]
    customerName: str
    pending: bool
    failed: bool
    active: bool


def order_summary(session: ConversationSession | None, service_type: str) -> WebOrderSummary:
    subtotal = cart_payload(session)["total"] if session else "0.00"
    mode = session.mode if session else service_type
    symbol = session.currency_symbol if session else ""
    if mode != "delivery":
        return WebOrderSummary(
            subtotal=subtotal, deliveryFee="0.00", total=subtotal, currencySymbol=symbol,
            fulfillment="counter" if mode == "counter" else "table",
        )
    delivery = session.delivery if session else None
    fulfillment = delivery.fulfillment if delivery else None
    # An unquoted/out-of-range delivery has no known price, never a free one.
    quoted = bool(delivery and delivery.location and delivery.covered and delivery.fee is not None)
    known = fulfillment == "pickup" or (fulfillment == "delivery" and quoted)
    summary = delivery_summary(session) if session else {}
    return WebOrderSummary(
        subtotal=subtotal,
        deliveryFee=str(summary["deliveryFee"]) if known else None,
        total=str(summary["total"]) if known else None,
        currencySymbol=symbol or (delivery.currency_symbol if delivery else ""),
        fulfillment=fulfillment,
    )


async def body(request: Request) -> WebRequest:
    data = await private_body(request)
    try:
        return WebRequest.model_validate(data)
    except ValidationError as exc:
        raise HTTPException(400, "La solicitud de chat no es válida.") from exc


# Every open chat polls /state. What cannot change between two polls is kept
# in the process: the table of the QR for a few seconds and the stage of
# messages that already finished. Writes always read foods-backend again.
POLLED_TABLE_SECONDS = 30.0
POLLED_TABLES_LIMIT = 2_048
SETTLED_STAGES_LIMIT = 20_000


async def polled_table(request: Request, qr: str) -> TableContext:
    cache: dict[str, tuple[float, TableContext]] | None
    cache = getattr(request.app.state, "web_polled_tables", None)
    if cache is None:
        cache = request.app.state.web_polled_tables = {}
    now = time.monotonic()
    known = cache.get(qr)
    if known is not None and known[0] > now:
        return known[1]
    table: TableContext = await request.app.state.fudia.resolve_table(qr)
    if len(cache) >= POLLED_TABLES_LIMIT:
        cache.clear()
    cache[qr] = (now + POLLED_TABLE_SECONDS, table)
    return table


def polled_stages(request: Request) -> dict[str, str]:
    cache: dict[str, str] | None = getattr(request.app.state, "web_settled_stages", None)
    if cache is None:
        cache = request.app.state.web_settled_stages = {}
    return cache


async def context(request: Request, data: WebRequest, *, writing: bool = False) -> dict[str, Any]:
    providers = manager(request)
    bound = await providers.web.resolve(data.capability)
    if not bound or bound["qr"] != data.qr or bound["visitor"] != data.visitor:
        raise HTTPException(410, "Tu sesión venció. Vuelve a abrir el chat desde el QR.")
    if writing:
        routes = await providers.routes(fresh=True)
        if not any(
            r.routingKey == bound["channel"] and r.provider == "FUDIA_WEB" and r.active for r in routes
        ):
            raise HTTPException(409, "El canal de este local cambió. Vuelve a escanear el QR.")
        table = await request.app.state.fudia.resolve_table(data.qr)
        if not table.conciergeEnabled:
            raise HTTPException(404, "El chat del local no está habilitado.")
    return bound


def inbound(
    data: WebRequest, bound: dict[str, Any], request_id: str, text: str, action: WebAction | None = None
) -> InboundMessage:
    return InboundMessage(
        provider="FUDIA_WEB",
        country_code=bound["country"],
        subject=data.visitor,
        phone=data.capability,
        sender_phone_id=bound["channel"],
        bound_qr=data.qr,
        message_id=f"web:{data.capability}:{request_id}",
        text=text,
        sent_at=time.time(),
        web_action=action,
        location=data.location,
    )


@router.post("/session", openapi_extra=WEB_REQUEST_DOC)
async def open_chat(request: Request) -> dict[str, Any]:
    data = await body(request)
    providers = manager(request)
    providers.web_origin()
    table = await request.app.state.fudia.resolve_table(data.qr)
    if not table.conciergeEnabled:
        raise HTTPException(404, "El chat del local no está habilitado.")
    routes = [
        r for r in await providers.routes(fresh=True) if r.active and r.countryCode == table.countryCode
    ]
    if len(routes) != 1 or routes[0].provider != "FUDIA_WEB":
        raise HTTPException(409, "El país no tiene el chat web activo. Vuelve a escanear el QR.")
    # A reload repeats the reference of its scan and the queue deduplicates it;
    # a new scan is a new start message. The worker opens a new context only
    # when the attention of this QR is over, so a live cart is never reset.
    start = f"start:{data.scan}" if data.scan else "start"
    if data.capability:
        bound = await context(request, data, writing=True)
    else:
        limiter = getattr(request.app.state, "web_session_limiter", None)
        if limiter is not None and not await limiter.allow(f"web-session:{data.qr}"):
            raise HTTPException(
                429, "Hay muchas personas abriendo este chat. Inténtalo de nuevo en unos minutos."
            )
        data.capability = await providers.web.open(
            data.qr, data.visitor, routes[0].routingKey, table.countryCode, start
        )
        opened = await providers.web.resolve(data.capability)
        assert opened is not None
        bound = opened
    # No synthetic start text is displayed.
    await request.app.state.inbound_queue.enqueue(inbound(data, bound, start, f"FUDIA:{data.qr}"))
    return {"capability": data.capability, "expiresIn": providers.web.ttl}


@router.post("/state", response_model=WebChatState, openapi_extra=WEB_REQUEST_DOC)
async def chat_state(request: Request) -> dict[str, Any]:
    data = await body(request)
    bound = await context(request, data)
    table = await polled_table(request, data.qr)
    if not table.conciergeEnabled:
        raise HTTPException(404, "El chat del local no está habilitado.")
    store = request.app.state.conversation_store
    session = await store.get(data.capability, bound["channel"])
    messages = await manager(request).web.messages(data.capability)
    failures = {
        entry["id"].removeprefix("failure:") for entry in messages if entry["id"].startswith("failure:")
    }
    pending = False
    failed = False
    settled = polled_stages(request)
    # Read the durable processing status, not a timeout that could resubmit an order.
    for entry in messages[-20:]:
        if entry["role"] == "user":
            if entry["id"] in failures:
                entry["status"] = "failed"
                failed = True
                continue
            stage = settled.get(entry["id"])
            if stage is None:
                record = await store.get_record(entry["id"])
                stage = record.stage if record else "queued"
                if stage in TERMINAL_STAGES:
                    # A finished message never changes again: later polls skip Redis.
                    if len(settled) >= SETTLED_STAGES_LIMIT:
                        settled.clear()
                    settled[entry["id"]] = stage
            entry["status"] = stage
            pending = pending or stage not in TERMINAL_STAGES
            failed = failed or stage == "failed"
    if session is None:
        start = f"web:{data.capability}:{bound.get('start') or 'start'}"
        record = await store.get_record(start)
        start_failed = start in failures
        pending = not start_failed and (record is None or not record.terminal)
        failed = start_failed or bool(record and record.stage == "failed")
    routes = await manager(request).routes()
    active = any(r.active and r.routingKey == bound["channel"] for r in routes)
    # The echo of a cart button is tracked above but not displayed: the reply
    # already names the dish, so each tap leaves one bubble instead of two.
    messages = [entry for entry in messages if not entry.get("hidden")]
    # Internal ids contain the private recipient capability. The browser only
    # needs stable display keys, never the capability hidden by its HttpOnly cookie.
    for entry in messages:
        entry["id"] = hashlib.sha256(entry["id"].encode()).hexdigest()
    return {
        "context": table.model_dump(mode="json"),
        "messages": messages,
        "cart": cart_payload(session) if session else {"items": [], "total": "0.00"},
        "orderSummary": order_summary(session, table.serviceType).model_dump(mode="json"),
        "confirmationRef": session.pending_order_request_id
        if session and session.awaiting_confirmation
        else None,
        "mode": session.mode if session else table.serviceType,
        "customerName": session.delivery.customer_name if session else "",
        "pending": pending,
        "failed": failed,
        "active": active,
    }


@router.post("/menu", openapi_extra=WEB_REQUEST_DOC)
async def chat_menu(request: Request) -> dict[str, Any]:
    data = await body(request)
    await context(request, data, writing=True)
    return dict((await request.app.state.fudia.search_menu(data.qr)).model_dump(mode="json"))


@router.post("/options", openapi_extra=WEB_REQUEST_DOC)
async def chat_options(request: Request) -> dict[str, Any]:
    data = await body(request)
    await context(request, data, writing=True)
    item = await request.app.state.fudia.get_product(data.qr, data.productId)
    if item is None:
        raise HTTPException(404, "El plato ya no está disponible.")
    detail = (
        await request.app.state.fudia.get_combo(data.qr, data.productId)
        if item.isCombo
        else await request.app.state.fudia.get_modifiers(data.qr, data.productId)
    )
    return {"kind": "combo" if item.isCombo else "modifiers", "detail": detail.model_dump(mode="json")}


@router.post("/messages", openapi_extra=WEB_REQUEST_DOC)
async def send_message(request: Request) -> dict[str, Any]:
    data = await body(request)
    bound = await context(request, data, writing=True)
    if not re.fullmatch(r"[A-Za-z0-9_-]{16,80}", data.requestId):
        raise HTTPException(400, "La referencia del mensaje no es válida.")
    if data.action and data.location:
        raise HTTPException(400, "Envía una acción o una ubicación por mensaje.")
    if not data.action and not data.text.strip() and data.location is None:
        raise HTTPException(400, "Escribe un mensaje.")
    if data.action:
        args = data.action.args
        # Bound numeric input before tools convert to floats/Decimals.
        if data.action.name.startswith("add_"):
            quantity = args.get("quantity", 1)
            if not isinstance(quantity, (int, float)) or not 0 < quantity <= 50:
                raise HTTPException(400, "La cantidad no es válida.")
        labels = {
            "confirm_order": "Sí, confirmo mi pedido",
            "prepare_confirmation": "Revisar mi pedido",
            "view_cart": "Ver mi pedido",
            "request_bill": "Pedir la cuenta",
            "request_human": "Solicitar atención",
        }
        text = labels.get(data.action.name, "Actualizar mi pedido")
        if data.action.name.startswith("add_"):
            item = await request.app.state.fudia.get_product(data.qr, str(args.get("productId", "")))
            if item is None:
                raise HTTPException(404, "El plato ya no está disponible.")
            text = f"Agregar {args.get('quantity', 1):g} × {item.name}"

    else:
        text = data.text.strip() or "Compartí mi ubicación para la entrega."
    message = inbound(data, bound, data.requestId, text, data.action)
    hidden = data.action is not None and (
        data.action.name.startswith("add_") or data.action.name in {"remove_item", "prepare_confirmation"}
    )
    # Store transcript before enqueue: retry repairs a transient queue failure.
    # Both append and enqueue deduplicate by the SAME stable message id.
    await manager(request).web.append(data.capability, message.message_id, "user", text, hidden=hidden)
    await request.app.state.inbound_queue.enqueue(message)
    return {"accepted": True}
