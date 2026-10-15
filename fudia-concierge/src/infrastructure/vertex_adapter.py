"""Gemini on Vertex AI through the Google Gen AI SDK.

The tool loop is manual (automatic function calling disabled): every tool
call goes through ToolRegistry, which enforces the per-intent allowlist and
the business rules. Each request re-sends the system instruction, and the
model turn is appended unchanged so Gemini keeps its thought signatures
between tool calls.
"""

from __future__ import annotations

import logging
import time
from collections.abc import Awaitable, Callable
from pathlib import Path
from typing import Any, Literal, cast

from google import genai
from google.genai import types

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

logger = logging.getLogger(__name__)

ToolHandler = Callable[[str, dict[str, Any]], Awaitable[dict[str, Any]]]
ModelTier = Literal["lite", "flash"]

EMPTY_REPLY = "¿Qué deseas pedir?"
RETRYABLE_STATUS_CODES = [408, 429, 500, 502, 503, 504]


def build_vertex_client(
    *,
    project: str,
    location: str,
    timeout_seconds: float,
    max_retries: int,
) -> genai.Client:
    """Vertex client with Application Default Credentials and bounded I/O."""
    return genai.Client(
        vertexai=True,
        project=project,
        location=location,
        http_options=types.HttpOptions(
            timeout=int(timeout_seconds * 1000),
            retry_options=types.HttpRetryOptions(
                attempts=max_retries + 1,
                http_status_codes=RETRYABLE_STATUS_CODES,
            ),
        ),
    )


def function_declarations(
    tool_schemas: list[dict[str, Any]],
) -> list[types.FunctionDeclaration]:
    """Translate the shared tool schemas into Gemini declarations. The JSON
    Schema of each tool is passed as-is, so both providers read one source."""
    return [
        types.FunctionDeclaration(
            name=str(schema["name"]),
            description=str(schema.get("description", "")),
            parameters_json_schema=schema["parameters"],
        )
        for schema in tool_schemas
    ]


def conversation_contents(
    session: ConversationSession,
    user_message: str,
    limit: int,
) -> list[types.Content]:
    contents = [
        types.Content(
            role="model" if item.role == "assistant" else "user",
            parts=[types.Part.from_text(text=item.content)],
        )
        for item in session.messages[-limit:]
    ]
    contents.append(
        types.Content(
            role="user",
            parts=[types.Part.from_text(text=user_message)],
        )
    )
    return contents


def _record_usage(
    response: types.GenerateContentResponse,
    role: str,
    tier: ModelTier,
) -> tuple[int, int]:
    usage = response.usage_metadata
    if usage is None:
        return 0, 0
    prompt = int(usage.prompt_token_count or 0)
    output = int(usage.candidates_token_count or 0) + int(
        usage.thoughts_token_count or 0
    )
    MODEL_TOKENS.labels(role=role, tier=tier, direction="input").inc(prompt)
    MODEL_TOKENS.labels(role=role, tier=tier, direction="output").inc(output)
    return prompt, output


class GeminiConciergeRouter:
    """Classifies the message into one of ROUTES with enum-constrained output,
    so a decorated answer can never be misread as a route."""

    def __init__(
        self,
        client: genai.Client,
        model: str,
        prompt_path: Path,
        tier: ModelTier,
    ) -> None:
        self.client = client
        self.model = model
        self.tier = tier
        self.config = types.GenerateContentConfig(
            system_instruction=prompt_path.read_text(encoding="utf-8"),
            response_mime_type="text/x.enum",
            response_schema=types.Schema(
                type=types.Type.STRING,
                enum=list(ROUTES),
            ),
            automatic_function_calling=types.AutomaticFunctionCallingConfig(
                disable=True
            ),
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
        try:
            response = await self.client.aio.models.generate_content(
                model=self.model,
                contents=router_input,
                config=self.config,
            )
        except Exception:
            MODEL_REQUESTS.labels(role="router", tier=self.tier, result="error").inc()
            raise
        MODEL_REQUESTS.labels(role="router", tier=self.tier, result="completed").inc()
        _record_usage(response, "router", self.tier)
        value = (response.text or "").strip().strip('"').strip().lower()

        if value not in ROUTES:
            SCOPE_DECISIONS.labels(decision="invalid_route").inc()
            INTENT_ROUTES.labels(intent="out_of_scope").inc()
            return "out_of_scope"

        route = cast(ConciergeRoute, value)
        SCOPE_DECISIONS.labels(
            decision="out_of_scope" if route == "out_of_scope" else "in_scope"
        ).inc()
        INTENT_ROUTES.labels(intent=route).inc()
        return route


class GeminiToolAgent:
    def __init__(
        self,
        client: genai.Client,
        model: str,
        system_prompt_path: Path,
        role_prompt_path: Path,
        tool_schemas: list[dict[str, Any]],
        agent_name: AgentIntent,
        tier: ModelTier,
        *,
        max_rounds: int = 8,
        max_output_tokens: int = 1_024,
    ) -> None:
        self.client = client
        self.model = model
        self.agent_name = agent_name
        self.tier = tier
        self.max_rounds = max_rounds
        base = system_prompt_path.read_text(encoding="utf-8")
        role = role_prompt_path.read_text(encoding="utf-8")
        self.config = types.GenerateContentConfig(
            system_instruction=f"{base}\n\n{role}",
            tools=[
                types.Tool(
                    function_declarations=function_declarations(tool_schemas)
                )
            ],
            automatic_function_calling=types.AutomaticFunctionCallingConfig(
                disable=True
            ),
            max_output_tokens=max_output_tokens,
        )

    async def _generate(
        self,
        contents: list[types.Content],
    ) -> types.GenerateContentResponse:
        try:
            response = await self.client.aio.models.generate_content(
                model=self.model,
                contents=contents,
                config=self.config,
            )
        except Exception:
            MODEL_REQUESTS.labels(
                role=self.agent_name, tier=self.tier, result="error"
            ).inc()
            raise
        MODEL_REQUESTS.labels(
            role=self.agent_name, tier=self.tier, result="completed"
        ).inc()
        return response

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
        contents = conversation_contents(session, user_message, 12)

        for _ in range(self.max_rounds):
            response = await self._generate(contents)
            used_in, used_out = _record_usage(response, self.agent_name, self.tier)
            input_tokens += used_in
            output_tokens += used_out

            calls = response.function_calls or []
            if not calls:
                text = (response.text or "").strip()
                duration = time.perf_counter() - started
                outcome = "completed" if text else "empty"
                AGENT_REQUESTS.labels(agent=self.agent_name, result=outcome).inc()
                AGENT_DURATION.labels(agent=self.agent_name).observe(duration)
                log_event(
                    logger,
                    "concierge.agent.completed",
                    conversation=session.conversation_id,
                    phone=fingerprint(session.phone),
                    agent=self.agent_name,
                    model=self.model,
                    tier=self.tier,
                    durationMs=round(duration * 1000, 1),
                    toolCalls=tool_calls,
                    inputTokens=input_tokens,
                    outputTokens=output_tokens,
                    empty=not text,
                )
                if not text:
                    raise AgentIncompleteError(EMPTY_REPLY)
                return text

            model_turn = (
                response.candidates[0].content
                if response.candidates and response.candidates[0].content
                else None
            )
            if model_turn is None:
                raise AgentIncompleteError(TOOL_LIMIT_REPLY)
            contents.append(model_turn)

            parts: list[types.Part] = []
            tool_calls += len(calls)
            for call in calls:
                name = call.name or ""
                tool_result = await tool_handler(name, dict(call.args or {}))
                parts.append(
                    types.Part(
                        function_response=types.FunctionResponse(
                            id=call.id,
                            name=name,
                            response={"output": tool_result},
                        )
                    )
                )
            contents.append(types.Content(role="user", parts=parts))

        AGENT_REQUESTS.labels(agent=self.agent_name, result="tool_limit").inc()
        AGENT_DURATION.labels(agent=self.agent_name).observe(
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
