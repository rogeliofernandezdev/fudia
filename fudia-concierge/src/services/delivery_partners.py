"""Delivery Fudia: mensajes a las centrales de reparto y sus botones.

foods-backend decide a qué central se ofrece cada pedido (cascada por
prioridad, 3 minutos por oferta) y deja el mensaje en su bandeja de salida.
`PartnerMessageRelay` lo reclama, lo envía al WhatsApp de la central y
confirma el envío (desde ese momento corren los 3 minutos de la oferta).

Las centrales están fuera de la ventana de 24 h de WhatsApp, así que se usan
plantillas aprobadas por Meta con botones de respuesta rápida. Cada botón lleva
el payload `fdx:<acción>:<token>`; el webhook lo reconoce y la acción se aplica
en foods-backend, que responde qué pasó para redactar la respuesta. Sin
plantillas configuradas (pruebas), se envía un mensaje interactivo con los
mismos botones, que Meta solo entrega dentro de la ventana de 24 h.
"""

from __future__ import annotations

import logging
import re
from dataclasses import dataclass
from decimal import Decimal, InvalidOperation
from typing import Protocol

from src.domain.models import DeliveryActionResult, DeliveryPartnerMessage, PartnerAction
from src.metrics import DELIVERY_PARTNER_MESSAGES
from src.observability import fingerprint, log_event
from src.services.ports import WhatsAppDeliveryUncertain

logger = logging.getLogger(__name__)

PAYLOAD_PREFIX = "fdx"
PAYLOAD_PATTERN = re.compile(
    r"^fdx:(accept|reject|pickup):([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$"
)

# Títulos de los botones (WhatsApp admite hasta 20 caracteres).
ACCEPT_TITLE = "Aceptar"
REJECT_TITLE = "Rechazar"
PICKUP_TITLE = "Recoger"

# Meta rechaza parámetros de plantilla con saltos de línea, tabulaciones o más
# de 4 espacios seguidos; cada parámetro se limita para no exceder el cuerpo.
_PARAM_LIMIT = 300


def button_payload(action: str, token: str) -> str:
    return f"{PAYLOAD_PREFIX}:{action}:{token}"


def parse_payload(payload: str) -> PartnerAction | None:
    """Botón de una central; None si el payload no es de Delivery Fudia."""
    match = PAYLOAD_PATTERN.fullmatch(payload.strip())
    if match is None:
        return None
    action = match.group(1)
    if action == "accept":
        return PartnerAction(action="accept", token=match.group(2).lower())
    if action == "reject":
        return PartnerAction(action="reject", token=match.group(2).lower())
    return PartnerAction(action="pickup", token=match.group(2).lower())


def template_param(value: str) -> str:
    """Una línea, sin espacios repetidos y acotada: Meta rechaza el resto."""
    text = " ".join(value.split())
    if len(text) > _PARAM_LIMIT:
        text = text[: _PARAM_LIMIT - 1].rstrip() + "…"
    return text or "-"


def money(symbol: str, amount: str) -> str:
    try:
        value = Decimal(amount.strip() or "0")
    except InvalidOperation:
        return amount
    return f"{symbol} {value:.2f}".strip()


def _quantity(value: float) -> str:
    return str(int(value)) if float(value).is_integer() else f"{value:g}"


def _items(message: DeliveryPartnerMessage) -> list[str]:
    lines = []
    for item in message.items:
        line = f"{_quantity(item.quantity)} × {item.name.strip()}"
        if item.note.strip():
            line += f" ({item.note.strip()})"
        lines.append(line)
    return lines


def _pickup_place(message: DeliveryPartnerMessage) -> str:
    venue = " · ".join(
        part.strip() for part in (message.organizationName, message.locationName) if part.strip()
    )
    address = message.locationAddress.strip()
    return f"{venue} ({address})" if venue and address else venue or address


def _map_link(message: DeliveryPartnerMessage) -> str:
    if message.latitude is None or message.longitude is None:
        return ""
    return f"https://maps.google.com/?q={message.latitude:.6f},{message.longitude:.6f}"


def _destination(message: DeliveryPartnerMessage) -> str:
    parts = [message.address.strip()]
    if message.reference.strip():
        parts.append(f"Ref.: {message.reference.strip()}")
    if message.distanceKm is not None:
        parts.append(f"{message.distanceKm:.1f} km")
    link = _map_link(message)
    if link:
        parts.append(link)
    return " · ".join(part for part in parts if part)


def _customer(message: DeliveryPartnerMessage) -> str:
    return " · ".join(
        part.strip() for part in (message.customerName, message.customerPhone) if part.strip()
    )


def _collect(message: DeliveryPartnerMessage) -> str:
    """Lo que la central cobra al cliente: el pedido más su delivery."""
    total = money(message.currencySymbol, message.total)
    fee = money(message.currencySymbol, message.deliveryFee)
    return f"{total} (incluye tu delivery {fee})"


def _code(message: DeliveryPartnerMessage) -> str:
    return message.orderCode.strip() or "sin código"


def offer_params(message: DeliveryPartnerMessage) -> list[str]:
    """Parámetros de la plantilla de oferta, en el orden de `docs`:
    {{1}} pedido, {{2}} recoge en, {{3}} entrega en, {{4}} cliente,
    {{5}} productos, {{6}} pagas al local, {{7}} cobras al cliente."""
    return [
        template_param(_code(message)),
        template_param(_pickup_place(message)),
        template_param(_destination(message)),
        template_param(_customer(message)),
        template_param(", ".join(_items(message))),
        template_param(money(message.currencySymbol, message.subtotal)),
        template_param(_collect(message)),
    ]


def ready_params(message: DeliveryPartnerMessage) -> list[str]:
    """{{1}} pedido, {{2}} recoge en, {{3}} pagas al local."""
    return [
        template_param(_code(message)),
        template_param(_pickup_place(message)),
        template_param(money(message.currencySymbol, message.subtotal)),
    ]


def cancelled_params(message: DeliveryPartnerMessage) -> list[str]:
    """{{1}} pedido, {{2}} local."""
    return [template_param(_code(message)), template_param(_pickup_place(message))]


def offer_text(message: DeliveryPartnerMessage) -> str:
    """Texto completo de la oferta para el mensaje interactivo (sin plantilla)."""
    lines = [f"🛵 Nuevo delivery {_code(message)}", f"Recoge en: {_pickup_place(message)}"]
    lines.append(f"Entrega en: {_destination(message)}")
    customer = _customer(message)
    if customer:
        lines.append(f"Cliente: {customer}")
    items = _items(message)
    if items:
        lines.append("Pedido:")
        lines.extend(f"- {item}" for item in items)
    if message.notes.strip():
        lines.append(f"Notas: {' '.join(message.notes.split())}")
    lines.append(f"Pagas al local: {money(message.currencySymbol, message.subtotal)}")
    lines.append(f"Cobras al cliente: {_collect(message)}")
    minutes = max(1, round(message.expiresInSeconds / 60)) if message.expiresInSeconds else 3
    lines.append(f"Responde en {minutes} minutos.")
    return "\n".join(lines)[:1024]


def ready_text(message: DeliveryPartnerMessage) -> str:
    return (
        f"✅ El pedido {_code(message)} está listo para recoger en {_pickup_place(message)}.\n"
        f"Paga al local: {money(message.currencySymbol, message.subtotal)}.\n"
        "Pulsa Recoger cuando lo tengas."
    )


def cancelled_text(message: DeliveryPartnerMessage) -> str:
    return (
        f"El pedido {_code(message)} de {_pickup_place(message)} ya no requiere tu delivery.\n"
        "No lo recojas."
    )


def action_reply(result: DeliveryActionResult) -> str:
    """Respuesta a la central después de pulsar un botón. Un reintento del
    mismo botón produce el mismo texto."""
    code = result.orderCode.strip() or "este pedido"
    venue = " · ".join(
        part.strip() for part in (result.organizationName, result.locationName) if part.strip()
    )
    place = f" en {venue}" if venue else ""
    yours = f"✅ El pedido {code} es tuyo.\nTe avisaremos cuando esté listo para recoger{place}."
    replies = {
        "accepted": yours,
        "already_accepted": yours,
        "rejected": f"Entendido, no tomarás el pedido {code}.",
        "taken": f"El pedido {code} ya no está disponible para ti.",
        "payment_conflict": (
            f"El local ya registró un pago del pedido {code}.\n"
            "No lo recojas ni cobres al cliente; el local coordinará el reparto."
        ),
        "expired": f"Se acabó el tiempo para aceptar el pedido {code}.\nLo ofrecimos a otra central.",
        "cancelled": f"El pedido {code} fue cancelado.\nNo lo recojas.",
        "not_ready": f"⏳ El pedido {code} aún no está listo.\nTe avisaremos cuando puedas recogerlo.",
        "picked_up": f"🛵 Registramos que recogiste el pedido {code}.\nCobra al cliente al entregar.",
    }
    return replies.get(
        result.outcome,
        "No encontramos ese pedido.\nUsa los botones del último mensaje de Fudia.",
    )


@dataclass(frozen=True)
class DeliveryTemplates:
    """Plantillas aprobadas por Meta; vacías envían mensajes interactivos."""

    offer: str = ""
    ready: str = ""
    cancelled: str = ""
    language: str = "es"


class PartnerMessageSource(Protocol):
    async def pending_delivery_messages(self, limit: int = 20) -> list[DeliveryPartnerMessage]: ...

    async def mark_delivery_message_dispatched(self, message_id: str) -> None: ...


class PartnerMessageSender(Protocol):
    async def send_text(self, phone: str, message: str, callback_ref: str = "") -> None: ...

    async def send_template(
        self,
        phone: str,
        name: str,
        language: str,
        body: list[str],
        button_payloads: list[str],
    ) -> None: ...

    async def send_buttons(self, phone: str, body: str, buttons: list[tuple[str, str]]) -> None: ...


class PartnerMessageRelay:
    """Envía a las centrales los mensajes de la bandeja de foods-backend."""

    def __init__(
        self,
        source: PartnerMessageSource,
        sender: PartnerMessageSender,
        templates: DeliveryTemplates,
        batch_size: int = 20,
    ) -> None:
        self.source = source
        self.sender = sender
        self.templates = templates
        self.batch_size = batch_size

    async def poll_once(self) -> int:
        """Envía y confirma cada mensaje. Si el envío falla con certeza, no se
        confirma: el reclamo vence y vuelve. Un envío incierto se confirma para
        no repetir una oferta que la central quizá ya recibió."""
        sent = 0
        for message in await self.source.pending_delivery_messages(self.batch_size):
            phone = re.sub(r"\D", "", message.partnerPhone)
            try:
                if phone:
                    await self._send(phone, message)
            except WhatsAppDeliveryUncertain:
                DELIVERY_PARTNER_MESSAGES.labels(kind=message.kind, result="uncertain").inc()
            except Exception as exc:
                DELIVERY_PARTNER_MESSAGES.labels(kind=message.kind, result="failed").inc()
                log_event(
                    logger,
                    "concierge.delivery_partner.send_failed",
                    message=fingerprint(message.id),
                    kind=message.kind,
                    errorType=type(exc).__name__,
                )
                continue
            try:
                await self.source.mark_delivery_message_dispatched(message.id)
            except Exception as exc:
                log_event(
                    logger,
                    "concierge.delivery_partner.confirm_failed",
                    message=fingerprint(message.id),
                    errorType=type(exc).__name__,
                )
                continue
            DELIVERY_PARTNER_MESSAGES.labels(kind=message.kind, result="sent" if phone else "no_phone").inc()
            sent += 1
        return sent

    async def _send(self, phone: str, message: DeliveryPartnerMessage) -> None:
        sender = self.sender
        resolver = getattr(self.sender, "for_channel", None)
        if resolver is not None:
            if not message.channelKey:
                raise RuntimeError("El mensaje no conserva su canal original.")
            sender = await resolver(message.channelKey)
            scope = getattr(sender, "for_event", None)
            if scope is not None:
                sender = scope(message.id)
        templates = bool(getattr(sender, "supports_templates", True))
        token = message.actionToken
        if message.kind == "offer":
            payloads = [button_payload("accept", token), button_payload("reject", token)]
            if templates and self.templates.offer:
                await sender.send_template(
                    phone, self.templates.offer, self.templates.language, offer_params(message), payloads
                )
                return
            await sender.send_buttons(
                phone, offer_text(message), [(payloads[0], ACCEPT_TITLE), (payloads[1], REJECT_TITLE)]
            )
            return
        if message.kind == "ready":
            payload = button_payload("pickup", token)
            if templates and self.templates.ready:
                await sender.send_template(
                    phone, self.templates.ready, self.templates.language, ready_params(message), [payload]
                )
                return
            await sender.send_buttons(phone, ready_text(message), [(payload, PICKUP_TITLE)])
            return
        if templates and self.templates.cancelled:
            await sender.send_template(
                phone, self.templates.cancelled, self.templates.language, cancelled_params(message), []
            )
            return
        await sender.send_text(phone, cancelled_text(message))
