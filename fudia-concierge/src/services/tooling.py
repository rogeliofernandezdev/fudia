from __future__ import annotations

from decimal import Decimal
from typing import Any, Protocol

from src.domain.models import ConversationSession, EffectKind
from src.services.ports import FudiaBackend


class EffectJournal(Protocol):
    """Recovery points around every state change made by a tool.

    `local_effect` persists a cart/confirmation change together with the
    session. `before_effect` persists the idempotency key and replay payload
    of a backend operation before calling it; `after_effect` / `effect_failed`
    close it. All of them refuse to write once the conversation lease is lost.
    """

    def ensure_active(self) -> None: ...
    async def local_effect(
        self, kind: EffectKind, result: dict[str, Any]
    ) -> None: ...
    async def before_effect(
        self, kind: EffectKind, request_id: str, payload: dict[str, Any]
    ) -> None: ...
    async def after_effect(self, result: dict[str, Any]) -> None: ...
    async def effect_failed(self, code: str, definitive: bool) -> None: ...


class NullJournal:
    """No persistence: direct tool use outside a queued message."""

    def ensure_active(self) -> None:
        return None

    async def local_effect(
        self, kind: EffectKind, result: dict[str, Any]
    ) -> None:
        return None

    async def before_effect(
        self, kind: EffectKind, request_id: str, payload: dict[str, Any]
    ) -> None:
        return None

    async def after_effect(self, result: dict[str, Any]) -> None:
        return None

    async def effect_failed(self, code: str, definitive: bool) -> None:
        return None


class ToolContext:
    def __init__(
        self,
        fudia: FudiaBackend,
        session: ConversationSession,
        user_message: str,
        journal: EffectJournal | None = None,
    ) -> None:
        self.fudia = fudia
        self.session = session
        self.user_message = user_message
        self.journal: EffectJournal = journal or NullJournal()

    def token(self) -> str:
        if not self.session.qr_token:
            raise RuntimeError("La sesión no tiene QR.")
        return self.session.qr_token

    def invalidate_confirmation(self) -> None:
        self.session.awaiting_confirmation = False
        self.session.pending_order_request_id = None

    def cart_payload(self) -> dict[str, Any]:
        return cart_payload(self.session)


def cart_payload(session: ConversationSession) -> dict[str, Any]:
    total = Decimal("0")
    items: list[dict[str, Any]] = []
    for line in session.cart:
        line_total = line.unit_price * Decimal(str(line.quantity))
        total += line_total
        items.append(
            {
                "productId": line.product_id,
                "name": line.name,
                "quantity": line.quantity,
                "unitPrice": str(line.unit_price),
                "lineTotal": str(line_total.quantize(Decimal("0.01"))),
                "note": line.note,
                "itemType": line.item_type,
                "selections": [
                    {
                        "groupId": selection.group_id,
                        "groupName": selection.group_name,
                        "productId": selection.product_id,
                        "name": selection.name,
                        "surcharge": str(selection.surcharge),
                    }
                    for selection in line.selections
                ],
                "modifiers": [
                    {
                        "groupId": modifier.group_id,
                        "groupName": modifier.group_name,
                        "optionId": modifier.option_id,
                        "name": modifier.name,
                        "surcharge": str(modifier.surcharge),
                    }
                    for modifier in line.modifiers
                ],
            }
        )
    return {
        "items": items,
        "total": str(total.quantize(Decimal("0.01"))),
    }
