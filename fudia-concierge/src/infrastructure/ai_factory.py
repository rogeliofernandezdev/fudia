"""Builds the router and specialist agents for the configured model provider.

Tier policy (by task difficulty):

- Flash-Lite: routing, menu questions and service requests (bill, staff).
  These are short, single-purpose turns with at most a couple of reads.
- Flash: building and confirming orders, which chains several tools
  (search, modifiers, combos, cart, confirmation) and must not misread a line.
- When a Flash-Lite turn fails before changing anything, Flash retries it once.
"""

from __future__ import annotations

from collections.abc import Awaitable, Callable
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Literal

from config.settings import Settings
from src.ai.contracts import ConciergeRouter, ConversationAgent
from src.ai.tiering import TieredAgent, TieredRouter
from src.domain.intents import AgentIntent
from src.infrastructure.tool_schemas import (
    MENU_TOOL_SCHEMAS,
    ORDER_TOOL_SCHEMAS,
    SERVICE_TOOL_SCHEMAS,
)

ModelTier = Literal["lite", "flash"]

AGENT_TIERS: dict[AgentIntent, ModelTier] = {
    "menu": "lite",
    "order": "flash",
    "service": "lite",
}
ROUTER_TIER: ModelTier = "lite"

TOOLS_BY_AGENT: dict[AgentIntent, list[dict[str, Any]]] = {
    "menu": MENU_TOOL_SCHEMAS,
    "order": ORDER_TOOL_SCHEMAS,
    "service": SERVICE_TOOL_SCHEMAS,
}
ROLE_PROMPTS: dict[AgentIntent, str] = {
    "menu": "menu_agent.md",
    "order": "order_agent.md",
    "service": "service_agent.md",
}


@dataclass
class AiStack:
    router: ConciergeRouter
    agents: dict[AgentIntent, ConversationAgent]
    close: Callable[[], Awaitable[None]]
    provider: str
    models: dict[str, str]


def build_ai(cfg: Settings, prompts: Path) -> AiStack:
    if cfg.llm_provider == "openai":
        return _build_openai(cfg, prompts)
    return _build_vertex(cfg, prompts)


def _build_vertex(cfg: Settings, prompts: Path) -> AiStack:
    from src.infrastructure.vertex_adapter import (
        GeminiConciergeRouter,
        GeminiToolAgent,
        build_vertex_client,
    )

    client = build_vertex_client(
        project=cfg.google_cloud_project,
        location=cfg.google_cloud_location,
        timeout_seconds=cfg.llm_timeout_seconds,
        max_retries=cfg.llm_max_retries,
    )
    models = {"lite": cfg.gemini_model_lite, "flash": cfg.gemini_model_flash}

    def agent(intent: AgentIntent, tier: ModelTier) -> GeminiToolAgent:
        return GeminiToolAgent(
            client,
            models[tier],
            prompts / "system.md",
            prompts / ROLE_PROMPTS[intent],
            TOOLS_BY_AGENT[intent],
            intent,
            tier,
            max_rounds=cfg.llm_max_tool_rounds,
            max_output_tokens=cfg.llm_max_output_tokens,
        )

    router_path = prompts / "concierge_router.md"
    router = TieredRouter(
        GeminiConciergeRouter(client, models["lite"], router_path, "lite"),
        GeminiConciergeRouter(client, models["flash"], router_path, "flash"),
    )
    agents: dict[AgentIntent, ConversationAgent] = {
        intent: TieredAgent(
            agent(intent, tier),
            agent(intent, "flash") if tier == "lite" else None,
            intent,
        )
        for intent, tier in AGENT_TIERS.items()
    }

    async def close() -> None:
        await client.aio.aclose()

    return AiStack(router, agents, close, "vertex", models)


def _build_openai(cfg: Settings, prompts: Path) -> AiStack:
    from openai import AsyncOpenAI

    from src.infrastructure.openai_adapter import (
        OpenAIConciergeRouter,
        OpenAIToolAgent,
    )

    client = AsyncOpenAI(
        api_key=cfg.openai_api_key,
        timeout=cfg.llm_timeout_seconds,
        max_retries=cfg.llm_max_retries,
    )
    models = {
        "lite": cfg.openai_model_lite or cfg.openai_model,
        "flash": cfg.openai_model,
    }
    tiered = models["lite"] != models["flash"]

    def agent(intent: AgentIntent, tier: ModelTier) -> OpenAIToolAgent:
        return OpenAIToolAgent(
            client,
            models[tier],
            prompts / "system.md",
            prompts / ROLE_PROMPTS[intent],
            TOOLS_BY_AGENT[intent],
            intent,
            tier,
            max_rounds=cfg.llm_max_tool_rounds,
            max_output_tokens=cfg.llm_max_output_tokens,
        )

    router_path = prompts / "concierge_router.md"
    router = TieredRouter(
        OpenAIConciergeRouter(client, models["lite"], router_path, "lite"),
        OpenAIConciergeRouter(client, models["flash"], router_path, "flash")
        if tiered
        else None,
    )
    agents: dict[AgentIntent, ConversationAgent] = {
        intent: TieredAgent(
            agent(intent, tier),
            agent(intent, "flash") if tier == "lite" and tiered else None,
            intent,
        )
        for intent, tier in AGENT_TIERS.items()
    }

    async def close() -> None:
        await client.close()

    return AiStack(router, agents, close, "openai", models)
