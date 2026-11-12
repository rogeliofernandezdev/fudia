"""Per-turn memo of catalog reads made by the tools of one agent turn.

foods-backend stays the source of truth: the memo lives only while one agent
turn runs (a few seconds), keys on the exact method and arguments, never stores
failures, and is discarded on any backend write. Orders and deliveries are
revalidated by the backend when they are created, so a catalog read reused
within the same turn cannot sell stock that no longer exists.
"""

from __future__ import annotations

import asyncio
from collections.abc import Awaitable, Callable
from typing import Any, TypeVar, cast

from src.domain.models import (
    BillSummary,
    ComboDetail,
    ConsentDecision,
    CustomerProfile,
    DeliveryActionResult,
    DeliveryOptions,
    DeliveryQuote,
    MenuItem,
    MenuResponse,
    ModifierConfig,
    OrderResult,
    OrderStatus,
    TableContext,
)
from src.services.ports import FudiaBackend

T = TypeVar("T")


class TurnScopedFudia:
    """`FudiaBackend` that reuses identical catalog reads within one turn.

    Memoized: `search_menu`, `get_product`, `get_combo`, `get_modifiers` and
    `get_delivery_options`. Concurrent identical calls (e.g. `build_carta`)
    share one in-flight request. Order status, handoff, bill and customer
    reads always reach the backend.
    """

    def __init__(self, backend: FudiaBackend) -> None:
        self._backend = backend
        self._reads: dict[tuple[Any, ...], asyncio.Future[Any]] = {}
        self.backend_reads = 0

    async def _memo(self, key: tuple[Any, ...], load: Callable[[], Awaitable[T]]) -> T:
        shared = self._reads.get(key)
        if shared is not None:
            # A finished read returns without yielding; an in-flight one is shared,
            # and cancelling this waiter does not cancel the owner's request.
            return cast(T, await asyncio.shield(shared))
        # The first caller awaits the backend itself (no extra task or loop hop),
        # so tool timing and cancellation behave exactly as without the memo.
        shared = asyncio.get_running_loop().create_future()
        self._reads[key] = shared
        self.backend_reads += 1
        try:
            value = await load()
        except BaseException as error:
            # A failed read is never reused: the next call retries the backend.
            if self._reads.get(key) is shared:
                del self._reads[key]
            if isinstance(error, Exception):
                shared.set_exception(error)
                shared.exception()  # mark retrieved when nobody else was waiting
            else:
                shared.cancel()
            raise
        shared.set_result(value)
        return value

    def _invalidate(self) -> None:
        self._reads.clear()

    async def _write(self, call: Awaitable[T]) -> T:
        self._invalidate()
        try:
            return await call
        finally:
            self._invalidate()

    # Memoized catalog reads.
    async def search_menu(
        self, token: str, query: str = "", product_id: str = ""
    ) -> MenuResponse:
        return await self._memo(
            ("search_menu", token, query, product_id),
            lambda: self._backend.search_menu(token, query, product_id),
        )

    async def get_product(self, token: str, product_id: str) -> MenuItem | None:
        return await self._memo(
            ("get_product", token, product_id),
            lambda: self._backend.get_product(token, product_id),
        )

    async def get_combo(self, token: str, product_id: str) -> ComboDetail:
        return await self._memo(
            ("get_combo", token, product_id),
            lambda: self._backend.get_combo(token, product_id),
        )

    async def get_modifiers(self, token: str, product_id: str) -> ModifierConfig:
        return await self._memo(
            ("get_modifiers", token, product_id),
            lambda: self._backend.get_modifiers(token, product_id),
        )

    async def get_delivery_options(self, token: str) -> DeliveryOptions:
        return await self._memo(
            ("get_delivery_options", token),
            lambda: self._backend.get_delivery_options(token),
        )

    # Live reads: state that changes outside the conversation.
    async def resolve_table(self, token: str) -> TableContext:
        return await self._backend.resolve_table(token)

    async def get_handoff_status(
        self, token: str, conversation_id: str
    ) -> dict[str, Any]:
        return await self._backend.get_handoff_status(token, conversation_id)

    async def get_order_status(
        self, token: str, order_id: str, conversation_id: str
    ) -> OrderStatus:
        return await self._backend.get_order_status(token, order_id, conversation_id)

    async def get_customer(self, token: str, phone: str) -> CustomerProfile:
        return await self._backend.get_customer(token, phone)

    # Writes discard every memoized read, before and after the call.
    async def request_handoff(
        self, token: str, phone: str, conversation_id: str, reason: str
    ) -> dict[str, Any]:
        return await self._write(
            self._backend.request_handoff(token, phone, conversation_id, reason)
        )

    async def create_order(
        self,
        token: str,
        phone: str,
        lines: list[dict[str, Any]],
        conversation_id: str,
        request_id: str,
        customer_name: str = "",
    ) -> OrderResult:
        return await self._write(
            self._backend.create_order(
                token, phone, lines, conversation_id, request_id, customer_name
            )
        )

    async def request_bill(
        self, token: str, phone: str, conversation_id: str, request_id: str = ""
    ) -> BillSummary:
        return await self._write(
            self._backend.request_bill(token, phone, conversation_id, request_id)
        )

    async def quote_delivery(
        self, token: str, latitude: float, longitude: float
    ) -> DeliveryQuote:
        return await self._write(
            self._backend.quote_delivery(token, latitude, longitude)
        )

    async def create_delivery_order(
        self, token: str, order: dict[str, Any]
    ) -> OrderResult:
        return await self._write(self._backend.create_delivery_order(token, order))

    async def record_marketing_consent(
        self,
        token: str,
        phone: str,
        conversation_id: str,
        request_id: str,
        decision: ConsentDecision,
        evidence: str,
    ) -> CustomerProfile:
        return await self._write(
            self._backend.record_marketing_consent(
                token, phone, conversation_id, request_id, decision, evidence
            )
        )

    async def open_conversation(
        self, token: str, conversation_id: str, kind: str, origin: dict[str, Any] | None = None,
    ) -> None:
        if origin is None:
            await self._write(self._backend.open_conversation(token, conversation_id, kind))
        else:
            await self._write(self._backend.open_conversation(token, conversation_id, kind, origin=origin))

    async def report_usage(
        self,
        token: str,
        conversation_id: str,
        request_id: str,
        tiers: list[dict[str, Any]],
    ) -> None:
        await self._backend.report_usage(token, conversation_id, request_id, tiers)

    async def apply_delivery_action(self, action: str, token: str, phone: str) -> DeliveryActionResult:
        return await self._backend.apply_delivery_action(action, token, phone)
