"""Per-turn memo of catalog reads (`TurnScopedFudia`)."""

from __future__ import annotations

import asyncio
from typing import Any, cast

import pytest

from src.services.ports import FudiaBackend, FudiaError
from src.services.turn_cache import TurnScopedFudia


class CountingBackend:
    def __init__(self) -> None:
        self.calls: list[tuple[Any, ...]] = []
        self.fail_next = False

    async def search_menu(self, token: str, query: str = "", product_id: str = "") -> Any:
        self.calls.append(("search_menu", query, product_id))
        await asyncio.sleep(0)
        if self.fail_next:
            self.fail_next = False
            raise FudiaError(503, "unavailable", "down")
        return {"query": query, "product_id": product_id, "n": len(self.calls)}

    async def get_combo(self, token: str, product_id: str) -> Any:
        self.calls.append(("get_combo", product_id))
        await asyncio.sleep(0)
        return {"combo": product_id}

    async def get_order_status(self, token: str, order_id: str, conversation_id: str) -> Any:
        self.calls.append(("get_order_status", order_id))
        return {"status": "listo"}

    async def create_order(
        self,
        token: str,
        phone: str,
        lines: list[dict[str, Any]],
        conversation_id: str,
        request_id: str,
        customer_name: str = "",
    ) -> Any:
        self.calls.append(("create_order", request_id))
        return {"id": "order-1"}


def scoped() -> tuple[TurnScopedFudia, CountingBackend]:
    backend = CountingBackend()
    return TurnScopedFudia(cast(FudiaBackend, backend)), backend


async def test_identical_reads_reach_the_backend_once() -> None:
    fudia, backend = scoped()
    first = await fudia.search_menu("t", "pollo")
    second = await fudia.search_menu("t", "pollo")
    assert first is second
    await fudia.search_menu("t", "lomo")
    await fudia.get_combo("t", "c1")
    await fudia.get_combo("t", "c1")
    assert backend.calls == [("search_menu", "pollo", ""), ("search_menu", "lomo", ""), ("get_combo", "c1")]
    assert fudia.backend_reads == 3


async def test_concurrent_identical_reads_share_one_request() -> None:
    fudia, backend = scoped()
    results = await asyncio.gather(*(fudia.get_combo("t", "c1") for _ in range(4)))
    assert all(result is results[0] for result in results)
    assert backend.calls == [("get_combo", "c1")]


async def test_failed_read_is_not_reused() -> None:
    fudia, backend = scoped()
    backend.fail_next = True
    with pytest.raises(FudiaError):
        await fudia.search_menu("t", "pollo")
    await fudia.search_menu("t", "pollo")
    assert backend.calls == [("search_menu", "pollo", ""), ("search_menu", "pollo", "")]


async def test_writes_discard_memoized_reads() -> None:
    fudia, backend = scoped()
    await fudia.search_menu("t", "pollo")
    await fudia.create_order("t", "51999", [], "conv", "req-1")
    await fudia.search_menu("t", "pollo")
    assert backend.calls == [
        ("search_menu", "pollo", ""),
        ("create_order", "req-1"),
        ("search_menu", "pollo", ""),
    ]


async def test_order_status_is_never_memoized() -> None:
    fudia, backend = scoped()
    await fudia.get_order_status("t", "o1", "conv")
    await fudia.get_order_status("t", "o1", "conv")
    assert backend.calls == [("get_order_status", "o1"), ("get_order_status", "o1")]
