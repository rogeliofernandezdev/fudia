"""Per-turn token metering for the monthly Concierge usage.

Prometheus keeps low-cardinality totals by role and tier; the per-company
figures are persisted by foods-backend. Model adapters record into the meter of
the turn being processed through a ContextVar, so neither the agents nor the
tools carry it as a parameter, and no company, conversation or phone ever
becomes a metric label.
"""

from __future__ import annotations

from collections.abc import Iterator
from contextlib import contextmanager
from contextvars import ContextVar
from typing import Any


class TokenMeter:
    def __init__(self) -> None:
        self._tiers: dict[str, list[int]] = {}
        self.qr_token = ""
        self.conversation_id = ""

    def add(self, tier: str, input_tokens: int, output_tokens: int) -> None:
        totals = self._tiers.setdefault(tier, [0, 0])
        totals[0] += max(0, int(input_tokens))
        totals[1] += max(0, int(output_tokens))

    def bind(self, qr_token: str | None, conversation_id: str) -> None:
        """The conversation the turn's tokens belong to."""
        self.qr_token = qr_token or ""
        self.conversation_id = conversation_id

    def tiers(self) -> list[dict[str, Any]]:
        return [
            {"tier": tier, "inputTokens": used[0], "outputTokens": used[1]}
            for tier, used in sorted(self._tiers.items())
            if used[0] or used[1]
        ]


_CURRENT: ContextVar[TokenMeter | None] = ContextVar(
    "concierge_token_meter", default=None
)


def record_tokens(tier: str, input_tokens: int, output_tokens: int) -> None:
    meter = _CURRENT.get()
    if meter is not None and (input_tokens or output_tokens):
        meter.add(tier, input_tokens, output_tokens)


def current_meter() -> TokenMeter | None:
    return _CURRENT.get()


@contextmanager
def metering() -> Iterator[TokenMeter]:
    """Meter the model calls made while the block runs, including tasks it
    creates (they copy the context and share this meter)."""
    meter = TokenMeter()
    token = _CURRENT.set(meter)
    try:
        yield meter
    finally:
        _CURRENT.reset(token)
