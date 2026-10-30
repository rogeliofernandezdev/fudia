"""Outbound phase: send a reply that the processing phase already persisted.

Sending never runs the model, the cart or a backend operation again. The
record moves through `reply_ready -> sending -> sent`, or to `uncertain` when
Meta may have accepted the message without answering. Delivery is at least
once: an uncertain send is reconciled with Meta status webhooks
(`biz_opaque_callback_data`) and, if none arrives, resent at most
`max_resends` times before it is marked `failed`.
"""

from __future__ import annotations

import logging
import time
from collections.abc import Callable
from enum import StrEnum
from typing import Protocol

from src.domain.models import InboundMessage, ProcessingRecord, ProcessingStage
from src.metrics import MESSAGES_SUPERSEDED, OUTBOUND_REPLIES
from src.observability import fingerprint, log_event
from src.services.ports import (
    ProcessingRecords,
    WhatsAppDeliveryUncertain,
    message_ref,
)

logger = logging.getLogger(__name__)

CONFIRMED_STATUSES = frozenset({"sent", "delivered", "read"})


class WhatsAppSender(Protocol):
    async def send_image(self, phone: str, url: str, caption: str = "", callback_ref: str = "") -> None: ...

    async def send_text(
        self,
        phone: str,
        message: str,
        callback_ref: str = "",
    ) -> None: ...


class DeliveryOutcome(StrEnum):
    SENT = "sent"
    SUPERSEDED = "superseded"
    FAILED = "failed"
    # Not finished yet: leave the queue entry pending and look again later.
    DEFERRED = "deferred"


class ReplyDispatcher:
    def __init__(
        self,
        records: ProcessingRecords,
        whatsapp: WhatsAppSender,
        reconcile_seconds: float = 30.0,
        max_resends: int = 1,
        clock: Callable[[], float] = time.time,
    ) -> None:
        self.records = records
        self.whatsapp = whatsapp
        self.reconcile_seconds = reconcile_seconds
        self.max_resends = max_resends
        self.clock = clock

    async def deliver(
        self,
        message: InboundMessage,
        record: ProcessingRecord,
    ) -> DeliveryOutcome:
        if record.stage == "sent":
            return DeliveryOutcome.SENT
        if record.stage == "superseded":
            return DeliveryOutcome.SUPERSEDED
        if record.stage == "failed":
            return DeliveryOutcome.FAILED

        ref = message_ref(
            record.message_id if record.part_index == 0 else f"{record.message_id}:image:{record.part_index}"
        )
        if await self._context_replaced(message, record):
            return await self._transition(record, "superseded", DeliveryOutcome.SUPERSEDED)

        if record.stage in ("sending", "uncertain"):
            status = await self.records.outbound_status(ref)
            if status in CONFIRMED_STATUSES:
                return await self._accepted(message, record)
            if status == "failed":
                record.last_error = "meta_status_failed"
                return await self._transition(record, "failed", DeliveryOutcome.FAILED)
            if self.clock() - record.last_send_at < self.reconcile_seconds:
                return DeliveryOutcome.DEFERRED
            if record.resends >= self.max_resends:
                record.last_error = "delivery_unconfirmed"
                return await self._transition(record, "failed", DeliveryOutcome.FAILED)
            # No confirmation within the window: Meta most likely never
            # accepted it. Resending may duplicate a text, never an action.
            record.resends += 1

        record.stage = "sending"
        record.send_attempts += 1
        record.last_send_at = self.clock()
        await self.records.put_record(record)
        OUTBOUND_REPLIES.labels(state="sending").inc()
        try:
            if record.part_index == 0:
                await self.whatsapp.send_text(message.phone, record.reply, callback_ref=ref)
            else:
                image = record.media[record.part_index - 1]
                await self.whatsapp.send_image(message.phone, image.url, image.caption, callback_ref=ref)
        except WhatsAppDeliveryUncertain:
            record.last_error = "delivery_uncertain"
            await self._transition(record, "uncertain", DeliveryOutcome.DEFERRED)
            return DeliveryOutcome.DEFERRED
        except Exception as exc:
            # Meta answered with an error or was unreachable: the message
            # was not accepted, so the same reply can be sent again.
            record.stage = "reply_ready"
            record.last_error = type(exc).__name__
            await self.records.put_record(record)
            OUTBOUND_REPLIES.labels(state="retry").inc()
            raise
        return await self._accepted(message, record)

    async def _accepted(self, message: InboundMessage, record: ProcessingRecord) -> DeliveryOutcome:
        if record.part_index < len(record.media):
            record.part_index += 1
            record.stage = "reply_ready"
            record.send_attempts = 0
            record.resends = 0
            record.last_send_at = 0
            await self.records.put_record(record)
            return await self.deliver(message, record)
        return await self._transition(record, "sent", DeliveryOutcome.SENT)

    async def _context_replaced(
        self,
        message: InboundMessage,
        record: ProcessingRecord,
    ) -> bool:
        if not record.conversation_id:
            return False
        session = await self.records.get(
            message.phone,
            message.sender_phone_id.strip() or "default",
        )
        return session is not None and session.conversation_id != record.conversation_id

    async def _transition(
        self,
        record: ProcessingRecord,
        stage: ProcessingStage,
        outcome: DeliveryOutcome,
    ) -> DeliveryOutcome:
        record.stage = stage
        await self.records.put_record(record)
        OUTBOUND_REPLIES.labels(state=stage).inc()
        if stage == "superseded":
            MESSAGES_SUPERSEDED.labels(phase="delivery").inc()
        if stage in ("uncertain", "failed", "superseded"):
            log_event(
                logger,
                f"concierge.outbound.{stage}",
                message=fingerprint(record.message_id),
                sendAttempts=record.send_attempts,
                resends=record.resends,
                reason=record.last_error,
            )
        return outcome
