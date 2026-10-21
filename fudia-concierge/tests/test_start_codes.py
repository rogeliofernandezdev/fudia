import re

import pytest

from src.services.ports import LocalTableStartCodes
from src.services.start_codes import (
    START_CODE_ALPHABET,
    looks_like_start_message,
    new_start_code,
    start_code_candidates,
    start_message,
    table_label,
)


def test_new_codes_are_short_unambiguous_and_mix_letters_and_digits() -> None:
    for _ in range(500):
        code = new_start_code()
        assert len(code) == 4
        assert set(code) <= set(START_CODE_ALPHABET)
        assert re.search(r"[A-Z]", code) and re.search(r"\d", code)
        assert not set(code) & set("01OIL")


def test_candidates_ignore_words_and_quantities() -> None:
    assert start_code_candidates("¡Hola! 👋 Quiero ver la carta.\nMesa 7 · K7P2") == ["K7P2"]
    assert start_code_candidates("k7p2") == ["K7P2"]
    assert start_code_candidates("Hola, quiero 2 lomos y una 7up") == []
    assert start_code_candidates("CASA 2024 H0LA") == []


def test_start_message_names_the_table_once() -> None:
    assert table_label("7") == "Mesa 7"
    assert table_label("Mesa 7") == "Mesa 7"
    assert table_label("MESA  12") == "MESA 12"
    assert start_message("7", "K7P2") == "¡Hola! 👋 Quiero ver la carta.\nMesa 7 · K7P2"


def test_start_message_is_recognized_even_when_edited() -> None:
    assert looks_like_start_message("Hola quiero VER LA CARTA")
    assert looks_like_start_message("mesa 7 k7p2")
    assert not looks_like_start_message("k7p2")


@pytest.mark.asyncio
async def test_local_codes_expire_and_resolve_case_insensitively() -> None:
    now = [0.0]
    codes = LocalTableStartCodes(ttl_seconds=600, clock=lambda: now[0])
    code = await codes.issue("a" * 32)
    assert await codes.resolve(code.lower()) == "a" * 32
    now[0] = 600
    assert await codes.resolve(code) is None


class FakeRedis:
    def __init__(self, taken: int = 0) -> None:
        self.values: dict[str, tuple[str, int]] = {}
        self.taken = taken

    async def set(self, key: str, value: str, nx: bool, ex: int) -> bool:
        assert nx and ex == 600
        if self.taken:
            self.taken -= 1
            return False
        self.values[key] = (value, ex)
        return True

    async def get(self, key: str) -> str | None:
        entry = self.values.get(key)
        return entry[0] if entry else None


@pytest.mark.asyncio
async def test_redis_codes_never_reassign_a_live_code_and_resolve_by_key() -> None:
    from src.infrastructure.start_codes import RedisTableStartCodes

    client = FakeRedis(taken=2)
    codes = RedisTableStartCodes("redis://unused", 600, client=client)  # type: ignore[arg-type]

    code = await codes.issue("b" * 32)

    assert list(client.values) == [f"fudia:concierge:startcode:{code}"]
    assert await codes.resolve(code.lower()) == "b" * 32
    assert await codes.resolve("ZZ99") is None
