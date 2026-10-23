from __future__ import annotations

import json
import logging
import time
from collections.abc import Awaitable, Callable
from pathlib import Path
from typing import Any, Literal, cast

from openai import AsyncOpenAI

from src.ai.tiering import TOOL_LIMIT_REPLY, AgentIncompleteError
from src.domain.intents import ROUTES, AgentIntent, ConciergeRoute
from src.domain.models import ConversationSession
from src.metrics import (
    AGENT_DURATION,
    AGENT_REQUESTS,
    INTENT_ROUTES,
    MODEL_REQUESTS,
    MODEL_TOKENS,
    SCOPE_DECISIONS,
)
from src.observability import fingerprint, log_event
from src.services.usage_meter import record_tokens

logger = logging.getLogger(__name__)

ToolHandler = Callable[
    [str, dict[str, Any]],
    Awaitable[dict[str, Any]],
]
ModelTier = Literal["lite", "flash"]
ROUTE_PUNCTUATION = "\"'`.,;:!¡¿?"


def normalize_route(text: str) -> str:
    """Accept a bare label with incidental quotes or punctuation."""
    return text.strip().strip(ROUTE_PUNCTUATION).strip().lower()


def _record_usage(response: Any, role: str, tier: ModelTier) -> tuple[int, int]:
    usage = getattr(response, "usage", None)
    if usage is None:
        return 0, 0
    used_in = int(getattr(usage, "input_tokens", 0) or 0)
    used_out = int(getattr(usage, "output_tokens", 0) or 0)
    MODEL_TOKENS.labels(role=role, tier=tier, direction="input").inc(used_in)
    MODEL_TOKENS.labels(role=role, tier=tier, direction="output").inc(used_out)
    record_tokens(tier, used_in, used_out)
    return used_in, used_out


def _conversation_context(
    session: ConversationSession,
    limit: int,
) -> list[dict[str, str]]:
    return [
        {
            "role": item.role,
            "content": item.content,
        }
        for item in session.messages[-limit:]
    ]


class OpenAIConciergeRouter:
    def __init__(
        self,
        client: AsyncOpenAI,
        model: str,
        prompt_path: Path,
        tier: ModelTier = "flash",
        max_output_tokens: int | None = None,
    ) -> None:
        self.client = client
        self.model = model
        self.tier = tier
        self.max_output_tokens = max_output_tokens
        self.instructions = prompt_path.read_text(
            encoding="utf-8"
        )

    async def route(
        self,
        session: ConversationSession,
        user_message: str,
    ) -> ConciergeRoute:
        context = "\n".join(
            f"{message.role}: {message.content}"
            for message in session.messages[-8:]
        )
        router_input = (
            "Contexto reciente del pedido:\n"
            f"{context or '(sin contexto)'}\n\n"
            "Mensaje actual del cliente:\n"
            f"{user_message}\n\n"
            f"awaiting_confirmation={session.awaiting_confirmation}\n"
            f"modo={session.mode}"
        )
        options: dict[str, Any] = {}
        if self.max_output_tokens:
            options["max_output_tokens"] = self.max_output_tokens
        try:
            response = await self.client.responses.create(
                model=self.model,
                instructions=self.instructions,
                input=router_input,
                **options,
            )
        except Exception:
            MODEL_REQUESTS.labels(role="router", tier=self.tier, result="error").inc()
            raise
        MODEL_REQUESTS.labels(role="router", tier=self.tier, result="completed").inc()
        _record_usage(response, "router", self.tier)
        value = normalize_route(response.output_text or "")

        if value not in ROUTES:
            SCOPE_DECISIONS.labels(
                decision="invalid_route"
            ).inc()
            INTENT_ROUTES.labels(
                intent="out_of_scope"
            ).inc()
            return "out_of_scope"

        route = cast(ConciergeRoute, value)
        SCOPE_DECISIONS.labels(
            decision=(
                "out_of_scope"
                if route == "out_of_scope"
                else "in_scope"
            )
        ).inc()
        INTENT_ROUTES.labels(
            intent=route
        ).inc()
        return route


class OpenAIToolAgent:
    def __init__(
        self,
        client: AsyncOpenAI,
        model: str,
        system_prompt_path: Path,
        role_prompt_path: Path,
        tool_schemas: list[dict[str, Any]],
        agent_name: AgentIntent,
        tier: ModelTier = "flash",
        *,
        max_rounds: int = 8,
        max_output_tokens: int | None = None,
    ) -> None:
        self.client = client
        self.model = model
        self.agent_name = agent_name
        self.tier = tier
        self.max_rounds = max_rounds
        self.max_output_tokens = max_output_tokens
        base = system_prompt_path.read_text(encoding="utf-8")
        role = role_prompt_path.read_text(encoding="utf-8")
        self.instructions = f"{base}\n\n{role}"
        self.tool_schemas = tool_schemas

    async def reply(
        self,
        session: ConversationSession,
        user_message: str,
        tool_handler: ToolHandler,
    ) -> str:
        started = time.perf_counter()
        tool_calls = 0
        input_tokens = 0
        output_tokens = 0

        def capture_usage(response: Any) -> None:
            nonlocal input_tokens, output_tokens
            used_in, used_out = _record_usage(
                response, self.agent_name, self.tier
            )
            input_tokens += used_in
            output_tokens += used_out

        options: dict[str, Any] = {}
        if self.max_output_tokens:
            options["max_output_tokens"] = self.max_output_tokens

        async def create(**kwargs: Any) -> Any:
            try:
                created = await self.client.responses.create(
                    model=self.model,
                    # Responses API does not carry instructions across
                    # previous_response_id, so every call re-sends them.
                    instructions=self.instructions,
                    tools=self.tool_schemas,  # type: ignore[arg-type]
                    **options,
                    **kwargs,
                )
            except Exception:
                MODEL_REQUESTS.labels(
                    role=self.agent_name, tier=self.tier, result="error"
                ).inc()
                raise
            MODEL_REQUESTS.labels(
                role=self.agent_name, tier=self.tier, result="completed"
            ).inc()
            return created

        history = _conversation_context(session, 12)
        history.append(
            {
                "role": "user",
                "content": user_message,
            }
        )
        response = await create(input=history)
        capture_usage(response)

        for _ in range(self.max_rounds):
            calls: list[Any] = [
                item
                for item in response.output
                if getattr(item, "type", "")
                == "function_call"
            ]
            if not calls:
                text = (response.output_text or "").strip()
                duration = time.perf_counter() - started
                AGENT_REQUESTS.labels(
                    agent=self.agent_name,
                    result="completed" if text else "empty",
                ).inc()
                AGENT_DURATION.labels(
                    agent=self.agent_name
                ).observe(duration)
                log_event(
                    logger,
                    "concierge.agent.completed",
                    conversation=session.conversation_id,
                    phone=fingerprint(session.phone),
                    agent=self.agent_name,
                    model=self.model,
                    tier=self.tier,
                    durationMs=round(
                        duration * 1000,
                        1,
                    ),
                    toolCalls=tool_calls,
                    inputTokens=input_tokens,
                    outputTokens=output_tokens,
                )
                if not text:
                    raise AgentIncompleteError("¿Qué deseas pedir?")
                return text

            outputs: list[dict[str, str]] = []
            tool_calls += len(calls)
            for call in calls:
                try:
                    arguments = json.loads(
                        call.arguments or "{}"
                    )
                except json.JSONDecodeError:
                    arguments = {}
                result = await tool_handler(
                    call.name,
                    arguments,
                )
                outputs.append(
                    {
                        "type": "function_call_output",
                        "call_id": call.call_id,
                        "output": json.dumps(
                            result,
                            ensure_ascii=False,
                            default=str,
                        ),
                    }
                )

            response = await create(
                previous_response_id=response.id,
                input=outputs,
            )
            capture_usage(response)

        AGENT_REQUESTS.labels(
            agent=self.agent_name,
            result="tool_limit",
        ).inc()
        AGENT_DURATION.labels(
            agent=self.agent_name
        ).observe(
            time.perf_counter() - started
        )
        log_event(
            logger,
            "concierge.agent.tool_limit",
            conversation=session.conversation_id,
            phone=fingerprint(session.phone),
            agent=self.agent_name,
            model=self.model,
            tier=self.tier,
            toolCalls=tool_calls,
        )
        raise AgentIncompleteError(TOOL_LIMIT_REPLY)
