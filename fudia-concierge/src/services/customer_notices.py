"""Avisos de foods-backend para el cliente («pedido listo»).

foods-backend deja el aviso en su bandeja de salida cuando Cocina/Barra termina
un pedido de mostrador o recojo creado por Concierge. `NoticeRelay` lo reclama y
lo encola en la conversación del cliente como un mensaje más (`InboundMessage`
con `notice`): hereda el orden, el lock, los reintentos y la dead-letter de la
cola. `ConciergeService` lo redacta y el `ReplyDispatcher` lo envía.
"""

from __future__ import annotations

import logging
import re
from typing import Protocol

from src.domain.models import CustomerNotice, InboundMessage
from src.metrics import CUSTOMER_NOTICES
from src.observability import fingerprint, log_event

logger = logging.getLogger(__name__)

# El id del aviso es la clave de deduplicación en la cola: reclamarlo de nuevo
# (reclamo vencido o confirmación perdida) no lo encola dos veces.
NOTICE_MESSAGE_PREFIX = "notice:"

# WhatsApp solo admite texto libre dentro de las 24 h posteriores al último
# mensaje del cliente; se deja un margen para el envío y los reintentos.
CUSTOMER_WINDOW_SECONDS = 23 * 3600


class NoticeSource(Protocol):
    async def pending_customer_notices(self, limit: int = 20) -> list[CustomerNotice]: ...

    async def mark_customer_notice_dispatched(self, notice_id: str) -> None: ...


class NoticeQueue(Protocol):
    async def enqueue(self, message: InboundMessage) -> bool: ...


def order_ready_text(notice: CustomerNotice) -> str:
    """Mensaje sobrio y en texto plano, con el código del pedido y el local."""
    order = f"Tu pedido {notice.orderCode}" if notice.orderCode.strip() else "Tu pedido"
    venue = " · ".join(
        part.strip() for part in (notice.organizationName, notice.locationName) if part.strip()
    )
    if notice.fulfillment == "pickup":
        first = f"✅ {order} está listo para recoger."
        second = f"Te esperamos en {venue}." if venue else "Te esperamos en el local."
    else:
        first = f"✅ {order} está listo."
        second = f"Recógelo en el mostrador de {venue}." if venue else "Recógelo en el mostrador."
    return f"{first}\n{second}"


class NoticeRelay:
    """Lleva los avisos de la bandeja de foods-backend a la cola de mensajes."""

    def __init__(
        self,
        source: NoticeSource,
        queue: NoticeQueue,
        channel_id: str = "",
        batch_size: int = 20,
    ) -> None:
        self.source = source
        self.queue = queue
        self.channel_id = channel_id
        self.batch_size = batch_size

    async def poll_once(self) -> int:
        """Encola los avisos pendientes y confirma cada uno ya encolado.

        Un aviso que no se pudo encolar no se confirma: su reclamo vence en
        foods-backend y vuelve en una consulta posterior.
        """
        relayed = 0
        for notice in await self.source.pending_customer_notices(self.batch_size):
            phone = re.sub(r"\D", "", notice.customerPhone)
            try:
                if phone:
                    await self.queue.enqueue(
                        InboundMessage(
                            message_id=NOTICE_MESSAGE_PREFIX + notice.id,
                            phone=phone,
                            text="",
                            sender_phone_id=self.channel_id,
                            notice=notice,
                        )
                    )
                # Sin número no hay a quién avisar: se confirma para no reintentar.
                await self.source.mark_customer_notice_dispatched(notice.id)
            except Exception as exc:
                CUSTOMER_NOTICES.labels(result="relay_failed").inc()
                log_event(
                    logger,
                    "concierge.customer_notice.relay_failed",
                    notice=fingerprint(notice.id),
                    errorType=type(exc).__name__,
                )
                continue
            CUSTOMER_NOTICES.labels(result="relayed" if phone else "no_phone").inc()
            relayed += 1
        return relayed
