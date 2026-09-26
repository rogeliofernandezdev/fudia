from pathlib import Path
from typing import Any

import pytest

from src.domain.models import ChatMessage, ConversationSession
from src.infrastructure.openai_adapter import OpenAIConciergeRouter


class FakeResponse:
    def __init__(self, output_text: str) -> None:
        self.output_text = output_text


class FakeResponses:
    def __init__(self, output_text: str) -> None:
        self.output_text = output_text
        self.calls: list[dict[str, Any]] = []

    async def create(self, **kwargs: Any) -> FakeResponse:
        self.calls.append(kwargs)
        return FakeResponse(self.output_text)


class FakeOpenAI:
    def __init__(self, output_text: str) -> None:
        self.responses = FakeResponses(output_text)


def prompt(
    tmp_path: Path,
    name: str,
    content: str,
) -> Path:
    path = tmp_path / name
    path.write_text(content, encoding="utf-8")
    return path


def session() -> ConversationSession:
    return ConversationSession(
        phone="51999999999",
        qr_token="a" * 32,
        messages=[
            ChatMessage(
                role="assistant",
                content="¿Qué deseas pedir?",
            ),
        ],
    )


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("output", "expected"),
    [
        ("menu", "menu"),
        ("order", "order"),
        ("service", "service"),
        ("out_of_scope", "out_of_scope"),
    ],
)
async def test_unified_router_accepts_only_known_routes(
    tmp_path: Path,
    output: str,
    expected: str,
) -> None:
    client = FakeOpenAI(output)
    router = OpenAIConciergeRouter(
        client,  # type: ignore[arg-type]
        "test-model",
        prompt(
            tmp_path,
            "router.md",
            "route exactly once",
        ),
    )

    assert (
        await router.route(
            session(),
            "mensaje",
        )
        == expected
    )
    assert len(client.responses.calls) == 1
    assert (
        client.responses.calls[0]["instructions"]
        == "route exactly once"
    )


@pytest.mark.asyncio
async def test_unified_router_is_fail_closed_on_invalid_output(
    tmp_path: Path,
) -> None:
    client = FakeOpenAI(
        "menu porque parece una consulta"
    )
    router = OpenAIConciergeRouter(
        client,  # type: ignore[arg-type]
        "test-model",
        prompt(
            tmp_path,
            "router.md",
            "route",
        ),
    )

    assert (
        await router.route(
            session(),
            "¿Quién fue Napoleón?",
        )
        == "out_of_scope"
    )


@pytest.mark.asyncio
async def test_unified_router_sends_context_and_confirmation_state(
    tmp_path: Path,
) -> None:
    client = FakeOpenAI("order")
    router = OpenAIConciergeRouter(
        client,  # type: ignore[arg-type]
        "test-model",
        prompt(
            tmp_path,
            "router.md",
            "route",
        ),
    )
    current = session()
    current.awaiting_confirmation = True

    await router.route(
        current,
        "sí",
    )

    router_input = str(
        client.responses.calls[0]["input"]
    )
    assert "¿Qué deseas pedir?" in router_input
    assert "awaiting_confirmation=True" in router_input
