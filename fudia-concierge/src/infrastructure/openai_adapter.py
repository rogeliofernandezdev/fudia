from __future__ import annotations

import json
import logging
import time
from collections.abc import Awaitable, Callable
from pathlib import Path
from typing import Any, cast

from openai import AsyncOpenAI

from src.domain.intents import ROUTES, AgentIntent, ConciergeRoute
from src.domain.models import ConversationSession
from src.metrics import (
    AGENT_DURATION,
    AGENT_REQUESTS,
    INTENT_ROUTES,
    SCOPE_DECISIONS,
)
from src.observability import fingerprint, log_event

logger = logging.getLogger(__name__)

ToolHandler = Callable[
    [str, dict[str, Any]],
    Awaitable[dict[str, Any]],
]


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
    ) -> None:
        self.client = client
        self.model = model
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
            f"awaiting_confirmation={session.awaiting_confirmation}"
        )
        response = await self.client.responses.create(
            model=self.model,
            instructions=self.instructions,
            input=router_input,
        )
        value = (
            response.output_text or ""
        ).strip().lower()

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
    ) -> None:
        self.client = client
        self.model = model
        self.agent_name = agent_name
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
            usage = getattr(response, "usage", None)
            if usage is None:
                return
            input_tokens += int(
                getattr(usage, "input_tokens", 0) or 0
            )
            output_tokens += int(
                getattr(usage, "output_tokens", 0) or 0
            )

        history = _conversation_context(session, 12)
        history.append(
            {
                "role": "user",
                "content": user_message,
            }
        )
        response = await self.client.responses.create(
            model=self.model,
            instructions=self.instructions,
            input=history,  # type: ignore[arg-type]
            tools=self.tool_schemas,  # type: ignore[arg-type]
        )
        capture_usage(response)

        for _ in range(8):
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
                    result="completed",
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
                    durationMs=round(
                        duration * 1000,
                        1,
                    ),
                    toolCalls=tool_calls,
                    inputTokens=input_tokens,
                    outputTokens=output_tokens,
                )
                return text or "¿Qué deseas pedir?"

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

            response = await self.client.responses.create(
                model=self.model,
                previous_response_id=response.id,
                input=outputs,  # type: ignore[arg-type]
                tools=self.tool_schemas,  # type: ignore[arg-type]
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
            toolCalls=tool_calls,
        )
        return "No pude completar la operación. Intenta nuevamente."
