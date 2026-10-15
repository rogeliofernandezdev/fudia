from pathlib import Path
from typing import Any

import pytest
from google.genai import types

from src.ai.tiering import AgentIncompleteError
from src.domain.models import ChatMessage, ConversationSession
from src.infrastructure.tool_schemas import MENU_TOOL_SCHEMAS, ORDER_TOOL_SCHEMAS
from src.infrastructure.vertex_adapter import (
    GeminiConciergeRouter,
    GeminiToolAgent,
    function_declarations,
)


def text_response(text: str) -> types.GenerateContentResponse:
    return types.GenerateContentResponse(
        candidates=[
            types.Candidate(
                content=types.Content(
                    role="model",
                    parts=[types.Part.from_text(text=text)],
                )
            )
        ],
        usage_metadata=types.GenerateContentResponseUsageMetadata(
            prompt_token_count=10,
            candidates_token_count=4,
        ),
    )


def call_response(
    name: str,
    args: dict[str, Any],
    call_id: str = "call-1",
) -> types.GenerateContentResponse:
    return types.GenerateContentResponse(
        candidates=[
            types.Candidate(
                content=types.Content(
                    role="model",
                    parts=[
                        types.Part(
                            function_call=types.FunctionCall(
                                id=call_id,
                                name=name,
                                args=args,
                            ),
                            thought_signature=b"signature",
                        )
                    ],
                )
            )
        ]
    )


class FakeModels:
    def __init__(self, responses: list[types.GenerateContentResponse]) -> None:
        self.responses = list(responses)
        self.calls: list[dict[str, Any]] = []

    async def generate_content(self, **kwargs: Any) -> types.GenerateContentResponse:
        # Snapshot contents: the agent keeps appending to the same list.
        kwargs["contents"] = (
            list(kwargs["contents"])
            if isinstance(kwargs["contents"], list)
            else kwargs["contents"]
        )
        self.calls.append(kwargs)
        return self.responses.pop(0)


class FakeAio:
    def __init__(self, models: FakeModels) -> None:
        self.models = models


class FakeClient:
    def __init__(self, responses: list[types.GenerateContentResponse]) -> None:
        self.models = FakeModels(responses)
        self.aio = FakeAio(self.models)


def prompt(tmp_path: Path, name: str, content: str) -> Path:
    path = tmp_path / name
    path.write_text(content, encoding="utf-8")
    return path


def session() -> ConversationSession:
    return ConversationSession(
        phone="51999999999",
        qr_token="a" * 32,
        messages=[
            ChatMessage(role="user", content="hola"),
            ChatMessage(role="assistant", content="¿Qué deseas pedir?"),
        ],
    )


def agent(
    tmp_path: Path,
    client: FakeClient,
    *,
    max_rounds: int = 8,
) -> GeminiToolAgent:
    return GeminiToolAgent(
        client,  # type: ignore[arg-type]
        "gemini-test-flash",
        prompt(tmp_path, "system.md", "SYSTEM"),
        prompt(tmp_path, "order.md", "ROLE"),
        ORDER_TOOL_SCHEMAS,
        "order",
        "flash",
        max_rounds=max_rounds,
    )


@pytest.mark.asyncio
async def test_router_constrains_output_to_known_routes(tmp_path: Path) -> None:
    client = FakeClient([text_response('"Order"')])
    router = GeminiConciergeRouter(
        client,  # type: ignore[arg-type]
        "gemini-test-lite",
        prompt(tmp_path, "router.md", "route exactly once"),
        "lite",
    )
    current = session()
    current.awaiting_confirmation = True

    assert await router.route(current, "sí") == "order"
    call = client.models.calls[0]
    config: types.GenerateContentConfig = call["config"]
    assert call["model"] == "gemini-test-lite"
    assert config.system_instruction == "route exactly once"
    assert config.response_mime_type == "text/x.enum"
    assert config.response_schema is not None
    assert set(config.response_schema.enum or []) == {
        "out_of_scope",
        "menu",
        "order",
        "service",
    }
    assert "awaiting_confirmation=True" in call["contents"]
    assert "¿Qué deseas pedir?" in call["contents"]


@pytest.mark.asyncio
async def test_router_is_fail_closed_on_unknown_label(tmp_path: Path) -> None:
    client = FakeClient([text_response("menu porque pregunta")])
    router = GeminiConciergeRouter(
        client,  # type: ignore[arg-type]
        "gemini-test-lite",
        prompt(tmp_path, "router.md", "route"),
        "lite",
    )

    assert await router.route(session(), "¿Quién fue Napoleón?") == "out_of_scope"


@pytest.mark.asyncio
async def test_agent_runs_tools_and_keeps_system_instruction_every_round(
    tmp_path: Path,
) -> None:
    client = FakeClient(
        [
            call_response("search_menu", {"query": "lomo"}),
            text_response("Tenemos Lomo saltado a S/ 25."),
        ]
    )
    executed: list[tuple[str, dict[str, Any]]] = []

    async def handler(name: str, args: dict[str, Any]) -> dict[str, Any]:
        executed.append((name, args))
        return {"items": [{"id": "p1", "name": "Lomo saltado"}]}

    reply = await agent(tmp_path, client).reply(session(), "¿hay lomo?", handler)

    assert reply == "Tenemos Lomo saltado a S/ 25."
    assert executed == [("search_menu", {"query": "lomo"})]
    assert len(client.models.calls) == 2
    for call in client.models.calls:
        config: types.GenerateContentConfig = call["config"]
        assert config.system_instruction == "SYSTEM\n\nROLE"
        assert config.automatic_function_calling is not None
        assert config.automatic_function_calling.disable is True

    first, second = client.models.calls
    assert [c.role for c in first["contents"]] == ["user", "model", "user"]
    model_turn = second["contents"][-2]
    tool_turn = second["contents"][-1]
    # The model turn goes back unchanged so Gemini keeps its thought signature.
    assert model_turn.parts[0].thought_signature == b"signature"
    response_part = tool_turn.parts[0].function_response
    assert response_part.id == "call-1"
    assert response_part.name == "search_menu"
    assert response_part.response == {
        "output": {"items": [{"id": "p1", "name": "Lomo saltado"}]}
    }


@pytest.mark.asyncio
async def test_agent_stops_at_tool_limit(tmp_path: Path) -> None:
    client = FakeClient(
        [
            call_response("view_cart", {}, "a"),
            call_response("view_cart", {}, "b"),
        ]
    )

    async def handler(name: str, args: dict[str, Any]) -> dict[str, Any]:
        return {"items": []}

    with pytest.raises(AgentIncompleteError) as error:
        await agent(tmp_path, client, max_rounds=2).reply(
            session(), "carrito", handler
        )
    assert "No pude completar" in error.value.reply


@pytest.mark.asyncio
async def test_agent_reports_empty_answer(tmp_path: Path) -> None:
    client = FakeClient([text_response("   ")])

    async def handler(name: str, args: dict[str, Any]) -> dict[str, Any]:
        return {}

    with pytest.raises(AgentIncompleteError) as error:
        await agent(tmp_path, client).reply(session(), "hola", handler)
    assert error.value.reply == "¿Qué deseas pedir?"


def test_function_declarations_reuse_shared_json_schema() -> None:
    declarations = function_declarations(MENU_TOOL_SCHEMAS)

    assert [d.name for d in declarations] == [
        "search_menu",
        "get_modifier_options",
        "get_combo_options",
    ]
    assert declarations[0].parameters_json_schema == MENU_TOOL_SCHEMAS[0]["parameters"]
