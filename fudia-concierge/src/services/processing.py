"""Recovery points of one inbound message.

Once a message has persisted any effect (a cart change, a prepared
confirmation or a backend operation), the model never runs again for it:
a retry either replays the pending backend call with the same idempotency key
or answers from what was persisted. Before that point nothing was written, so
running the turn again is safe.
"""

from __future__ import annotations

from decimal import Decimal, InvalidOperation
from typing import Any

from src.domain.models import (
    ConversationSession,
    EffectKind,
    EffectRecord,
    ProcessingRecord,
)
from src.services.ports import ConversationLease, ConversationStore


class TurnJournal:
    """`EffectJournal` that checkpoints session and record together."""

    def __init__(
        self,
        store: ConversationStore,
        lease: ConversationLease,
        session: ConversationSession,
        record: ProcessingRecord,
    ) -> None:
        self.store = store
        self.lease = lease
        self.session = session
        self.record = record

    def ensure_active(self) -> None:
        self.lease.ensure()

    async def checkpoint(self, *, with_session: bool = True) -> None:
        await self.store.commit(
            self.lease,
            self.session if with_session else None,
            self.record,
        )

    async def local_effect(
        self,
        kind: EffectKind,
        result: dict[str, Any],
    ) -> None:
        self.record.effects.append(
            EffectRecord(kind=kind, result=result)
        )
        self.record.stage = "effects_applied"
        await self.checkpoint()

    async def before_effect(
        self,
        kind: EffectKind,
        request_id: str,
        payload: dict[str, Any],
    ) -> None:
        # The ownership check happens again inside the fenced commit; the
        # backend call only starts once its replay data is durable.
        await self.lease.verify()
        self.record.effects.append(
            EffectRecord(
                kind=kind,
                status="pending",
                request_id=request_id,
                payload=payload,
            )
        )
        self.record.stage = "effect_pending"
        await self.checkpoint()

    async def after_effect(self, result: dict[str, Any]) -> None:
        effect = self.record.pending_effect()
        if effect is None:
            return
        effect.status = "done"
        effect.result = result
        self.record.stage = "effects_applied"
        await self.checkpoint()

    async def effect_failed(self, code: str, definitive: bool) -> None:
        effect = self.record.pending_effect()
        if effect is None:
            return
        # A definitive rejection did not happen; an unreachable backend may
        # have applied it, so the idempotency key stays in the session.
        effect.status = "failed" if definitive else "unknown"
        effect.error_code = code
        self.record.stage = "effects_applied"
        await self.checkpoint()


def _money(symbol: str, value: Any) -> str:
    try:
        amount = Decimal(str(value)).quantize(Decimal("0.01"))
    except (InvalidOperation, ValueError):
        amount = Decimal("0.00")
    return f"{symbol} {amount}".strip()


def _cart_lines(cart: dict[str, Any]) -> str:
    items = cart.get("items", [])
    if not isinstance(items, list) or not items:
        return "Tu pedido está vacío."
    rows = []
    for item in items:
        if not isinstance(item, dict):
            continue
        quantity = item.get("quantity", "")
        if isinstance(quantity, float) and quantity.is_integer():
            quantity = int(quantity)
        rows.append(f"- {quantity} × {item.get('name', '')}")
    rows.append(f"Total: {_money('', cart.get('total', '0'))}")
    return "\n".join(rows)


def recovery_reply(
    record: ProcessingRecord,
    session: ConversationSession,
) -> str:
    """Deterministic answer built from the persisted effects of a message
    whose turn was interrupted after it changed something."""
    effect = next(
        (
            item
            for item in reversed(record.effects)
            if item.status != "unknown"
        ),
        None,
    )
    if effect is None:
        return (
            "No pudimos confirmar el resultado de tu última solicitud. "
            "Revisa tu pedido o pide ayuda al personal de tu mesa."
        )
    if effect.status == "failed":
        return (
            "No pudimos completar tu solicitud. Revisa tu pedido e "
            "inténtalo nuevamente, o pide ayuda al personal de tu mesa."
        )
    if effect.kind == "create_order":
        order = effect.result.get("order", {})
        fulfillment = effect.payload.get("order", {}).get("fulfillment")
        if effect.payload.get("mode") == "delivery":
            how = (
                "Te lo enviaremos con nuestro delivery y pagas al recibirlo."
                if fulfillment == "delivery"
                else "Puedes recogerlo en el local y pagas al recogerlo."
            )
            return (
                f"Tu pedido {order.get('code', '')} fue registrado. "
                f"Total: {_money('', order.get('total', '0'))}. {how}"
            )
        return (
            f"Tu pedido {order.get('code', '')} fue registrado. "
            f"Total de esta ronda: {_money('', order.get('total', '0'))}."
        )
    if effect.kind == "request_handoff":
        return (
            "El personal del local ya fue avisado. "
            "Espera un momento mientras se acercan a tu mesa."
        )
    if effect.kind == "request_bill":
        symbol = str(effect.result.get("currencySymbol", ""))
        return (
            f"🧾 Tu cuenta {effect.result.get('code', '')}:\n"
            f"- Total: {_money(symbol, effect.result.get('total'))}\n"
            f"- Pagado: {_money(symbol, effect.result.get('paidAmount'))}\n"
            "- Saldo pendiente: "
            f"{_money(symbol, effect.result.get('remainingAmount'))}\n\n"
            "El pago se realiza con el personal del local."
        )
    if effect.kind == "delivery":
        return (
            "Actualicé los datos de entrega de tu pedido. "
            "¿Deseas algo más o confirmamos?"
        )
    cart = effect.result.get("cart", {})
    if effect.kind == "confirmation" and session.awaiting_confirmation:
        return (
            f"🧾 Tu pedido:\n{_cart_lines(cart)}\n\n"
            "¿Confirmas este pedido? Responde «sí» para enviarlo."
        )
    return f"🧾 Actualicé tu pedido:\n{_cart_lines(cart)}\n\n¿Deseas algo más?"
