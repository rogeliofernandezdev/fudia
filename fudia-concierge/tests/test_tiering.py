from pathlib import Path
from typing import Any

import pytest

from config.settings import Settings
from src.ai.tiering import (
    TOOL_LIMIT_REPLY,
    AgentIncompleteError,
    TieredAgent,
    TieredRouter,
)
from src.domain.intents import ConciergeRoute
from src.domain.models import ConversationSession
from src.infrastructure import ai_factory
from src.infrastructure.ai_factory import AGENT_TIERS, ROUTER_TIER, build_ai


def session() -> ConversationSession:
    return ConversationSession(phone="51999999999", qr_token="a" * 32)


class ScriptedAgent:
    """Runs the given tools, then fails or answers."""

    def __init__(
        self,
        tools: list[str],
        *,
        error: Exception | None = None,
        reply: str = "listo",
    ) -> None:
        self.tools = tools
        self.error = error
        self.reply_text = reply
        self.calls = 0

    async def reply(
        self,
        session: ConversationSession,
        user_message: str,
        tool_handler: Any,
    ) -> str:
        self.calls += 1
        for name in self.tools:
            await tool_handler(name, {})
        if self.error is not None:
            raise self.error
        return self.reply_text


async def handler(name: str, args: dict[str, Any]) -> dict[str, Any]:
    return {"ok": True}


@pytest.mark.asyncio
async def test_light_failure_after_reads_escalates_to_flash() -> None:
    lite = ScriptedAgent(["search_menu"], error=TimeoutError("lite"))
    flash = ScriptedAgent([], reply="respuesta flash")

    reply = await TieredAgent(lite, flash, "menu").reply(session(), "carta", handler)

    assert reply == "respuesta flash"
    assert (lite.calls, flash.calls) == (1, 1)


@pytest.mark.asyncio
async def test_light_failure_after_a_mutation_never_repeats_it() -> None:
    lite = ScriptedAgent(["request_bill"], error=TimeoutError("lite"))
    flash = ScriptedAgent([], reply="no debe correr")

    with pytest.raises(TimeoutError):
        await TieredAgent(lite, flash, "service").reply(session(), "la cuenta", handler)
    assert flash.calls == 0


@pytest.mark.asyncio
async def test_incomplete_answer_without_escalation_returns_safe_reply() -> None:
    lite = ScriptedAgent(["add_item"], error=AgentIncompleteError(TOOL_LIMIT_REPLY))
    flash = ScriptedAgent([])

    reply = await TieredAgent(lite, flash, "order").reply(session(), "dos", handler)

    assert reply == TOOL_LIMIT_REPLY
    assert flash.calls == 0


@pytest.mark.asyncio
async def test_escalated_agent_incomplete_returns_its_safe_reply() -> None:
    lite = ScriptedAgent([], error=AgentIncompleteError("¿Qué deseas pedir?"))
    flash = ScriptedAgent([], error=AgentIncompleteError("¿Qué deseas pedir?"))

    reply = await TieredAgent(lite, flash, "menu").reply(session(), "hola", handler)

    assert reply == "¿Qué deseas pedir?"
    assert flash.calls == 1


class ScriptedRouter:
    def __init__(self, route: ConciergeRoute | None = None) -> None:
        self.value = route
        self.calls = 0

    async def route(self, session: ConversationSession, user_message: str) -> ConciergeRoute:
        self.calls += 1
        if self.value is None:
            raise ConnectionError("lite down")
        return self.value


@pytest.mark.asyncio
async def test_router_falls_back_to_flash_on_error() -> None:
    lite = ScriptedRouter(None)
    flash = ScriptedRouter("order")

    assert await TieredRouter(lite, flash).route(session(), "quiero") == "order"
    assert (lite.calls, flash.calls) == (1, 1)


def test_tier_policy_matches_task_difficulty() -> None:
    assert ROUTER_TIER == "lite"
    assert AGENT_TIERS == {"menu": "lite", "order": "flash", "service": "lite"}


def test_vertex_stack_uses_lite_and_flash_models(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    created: dict[str, Any] = {}

    class FakeAio:
        async def aclose(self) -> None:
            created["closed"] = True

    class FakeClient:
        aio = FakeAio()

    def fake_client(**kwargs: Any) -> FakeClient:
        created.update(kwargs)
        return FakeClient()

    monkeypatch.setattr(
        "src.infrastructure.vertex_adapter.build_vertex_client", fake_client
    )
    cfg = Settings(
        _env_file=None,
        llm_provider="vertex",
        google_cloud_project="demo-project",
        gemini_model_lite="lite-model",
        gemini_model_flash="flash-model",
    )
    prompts = Path(ai_factory.__file__).resolve().parents[2] / "prompts"

    stack = build_ai(cfg, prompts)

    assert created["project"] == "demo-project"
    assert created["location"] == "global"
    assert stack.provider == "vertex"
    assert stack.models == {"lite": "lite-model", "flash": "flash-model"}
    router: Any = stack.router
    assert (router.primary.model, router.fallback.model) == ("lite-model", "flash-model")
    menu: Any = stack.agents["menu"]
    order: Any = stack.agents["order"]
    service: Any = stack.agents["service"]
    assert (menu.primary.model, menu.escalation.model) == ("lite-model", "flash-model")
    assert (service.primary.model, service.escalation.model) == ("lite-model", "flash-model")
    assert (order.primary.model, order.escalation) == ("flash-model", None)


def test_production_requires_vertex_project_but_not_openai_key() -> None:
    base: dict[str, Any] = {
        "_env_file": None,
        "env": "production",
        "fudia_concierge_api_key": "k",
        "whatsapp_token": "t",
        "whatsapp_phone_id": "p",
        "whatsapp_verify_token": "v",
        "whatsapp_app_secret": "s",
        "whatsapp_graph_version": "v1",
    }
    with pytest.raises(ValueError, match="GOOGLE_CLOUD_PROJECT"):
        Settings(**base, llm_provider="vertex", google_cloud_project="")
    Settings(**base, llm_provider="vertex", google_cloud_project="demo")
    with pytest.raises(ValueError, match="OPENAI_API_KEY"):
        Settings(**base, llm_provider="openai", openai_api_key="")
