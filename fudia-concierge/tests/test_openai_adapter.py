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


class SequenceResponses:
    def __init__(self, responses: list[Any]) -> None:
        self.responses = list(responses)
        self.calls: list[dict[str, Any]] = []

    async def create(self, **kwargs: Any) -> Any:
        self.calls.append(kwargs)
        return self.responses.pop(0)


class FunctionCallItem:
    type = "function_call"
    name = "search_menu"
    arguments = '{"query": "lomo"}'
    call_id = "call-1"


class ToolCallResponse:
    id = "resp-1"
    output_text = ""
    output = [FunctionCallItem()]
    usage = None


class FinalResponse:
    id = "resp-2"
    output_text = "Tenemos lomo saltado."
    output: list[Any] = []
    usage = None


@pytest.mark.asyncio
async def test_agent_resends_instructions_after_tool_calls(
    tmp_path: Path,
) -> None:
    from src.infrastructure.openai_adapter import OpenAIToolAgent
    from src.infrastructure.tool_schemas import MENU_TOOL_SCHEMAS

    responses = SequenceResponses([ToolCallResponse(), FinalResponse()])
    client = type("Client", (), {"responses": responses})()
    agent = OpenAIToolAgent(
        client,  # type: ignore[arg-type]
        "test-model",
        prompt(tmp_path, "system.md", "SYSTEM"),
        prompt(tmp_path, "menu.md", "ROLE"),
        MENU_TOOL_SCHEMAS,
        "menu",
    )

    async def handler(name: str, args: dict[str, Any]) -> dict[str, Any]:
        return {"items": []}

    assert await agent.reply(session(), "¿hay lomo?", handler) == "Tenemos lomo saltado."
    assert len(responses.calls) == 2
    assert all(call["instructions"] == "SYSTEM\n\nROLE" for call in responses.calls)
    assert responses.calls[1]["previous_response_id"] == "resp-1"


def test_route_normalization_tolerates_punctuation() -> None:
    from src.infrastructure.openai_adapter import normalize_route

    assert normalize_route(' "Menu." ') == "menu"
    assert normalize_route("order!") == "order"
