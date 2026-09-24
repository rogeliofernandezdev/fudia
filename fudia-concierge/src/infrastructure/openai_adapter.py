from __future__ import annotations

import json
from collections.abc import Awaitable, Callable
from pathlib import Path
from typing import Any, Protocol

from openai import AsyncOpenAI

from src.domain.models import ConversationSession

ToolHandler = Callable[[str, dict[str, Any]], Awaitable[dict[str, Any]]]


class AssistantEngine(Protocol):
    async def reply(
        self,
        session: ConversationSession,
        user_message: str,
        tool_handler: ToolHandler,
    ) -> str: ...


TOOLS: list[dict[str, Any]] = [
    {
        "type": "function",
        "name": "search_menu",
        "description": "Busca productos reales de la carta de esta mesa.",
        "parameters": {
            "type": "object",
            "properties": {"query": {"type": "string"}},
            "required": ["query"],
            "additionalProperties": False,
        },
        "strict": True,
    },
    {
        "type": "function",
        "name": "add_item",
        "description": "Agrega al carrito un producto devuelto por search_menu.",
        "parameters": {
            "type": "object",
            "properties": {
                "productId": {"type": "string"},
                "quantity": {"type": "number", "minimum": 0.01},
                "note": {"type": "string"},
            },
            "required": ["productId", "quantity", "note"],
            "additionalProperties": False,
        },
        "strict": True,
    },
    {
        "type": "function",
        "name": "get_combo_options",
        "description": (
            "Obtiene los grupos, reglas y alternativas reales de un combo "
            "devuelto por search_menu."
        ),
        "parameters": {
            "type": "object",
            "properties": {"productId": {"type": "string"}},
            "required": ["productId"],
            "additionalProperties": False,
        },
        "strict": True,
    },
    {
        "type": "function",
        "name": "add_combo_item",
        "description": (
            "Agrega un combo con las selecciones elegidas entre las opciones "
            "devueltas por get_combo_options."
        ),
        "parameters": {
            "type": "object",
            "properties": {
                "productId": {"type": "string"},
                "quantity": {"type": "number", "minimum": 0.01},
                "note": {"type": "string"},
                "selections": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {
                            "groupId": {"type": "string"},
                            "productId": {"type": "string"},
                        },
                        "required": ["groupId", "productId"],
                        "additionalProperties": False,
                    },
                },
            },
            "required": ["productId", "quantity", "note", "selections"],
            "additionalProperties": False,
        },
        "strict": True,
    },
    {
        "type": "function",
        "name": "remove_item",
        "description": "Retira un producto del carrito.",
        "parameters": {
            "type": "object",
            "properties": {"productId": {"type": "string"}},
            "required": ["productId"],
            "additionalProperties": False,
        },
        "strict": True,
    },
    {
        "type": "function",
        "name": "view_cart",
        "description": "Consulta el carrito actual.",
        "parameters": {"type": "object", "properties": {}, "additionalProperties": False},
        "strict": True,
    },
    {
        "type": "function",
        "name": "prepare_confirmation",
        "description": "Prepara el resumen que debe confirmar el cliente.",
        "parameters": {"type": "object", "properties": {}, "additionalProperties": False},
        "strict": True,
    },
    {
        "type": "function",
        "name": "confirm_order",
        "description": "Crea el pedido solo después de una confirmación explícita.",
        "parameters": {"type": "object", "properties": {}, "additionalProperties": False},
        "strict": True,
    },
]


class OpenAIConciergeEngine:
    def __init__(
        self,
        api_key: str,
        model: str,
        prompt_path: Path | None = None,
        client: AsyncOpenAI | None = None,
    ) -> None:
        self.model = model
        self.client = client or AsyncOpenAI(api_key=api_key)
        path = prompt_path or Path(__file__).resolve().parents[2] / "prompts" / "system.md"
        self.instructions = path.read_text(encoding="utf-8")

    async def reply(
        self,
        session: ConversationSession,
        user_message: str,
        tool_handler: ToolHandler,
    ) -> str:
        history = [
            {"role": item.role, "content": item.content}
            for item in session.messages[-12:]
        ]
        history.append({"role": "user", "content": user_message})

        response = await self.client.responses.create(
            model=self.model,
            instructions=self.instructions,
            input=history,  # type: ignore[arg-type]
            tools=TOOLS,  # type: ignore[arg-type]
        )

        for _ in range(8):
            calls: list[Any] = [
                item
                for item in response.output
                if getattr(item, "type", "") == "function_call"
            ]
            if not calls:
                text = (response.output_text or "").strip()
                return text or "¿Qué deseas pedir?"

            outputs: list[dict[str, str]] = []
            for call in calls:
                try:
                    arguments = json.loads(call.arguments or "{}")
                except json.JSONDecodeError:
                    arguments = {}
                result = await tool_handler(call.name, arguments)
                outputs.append(
                    {
                        "type": "function_call_output",
                        "call_id": call.call_id,
                        "output": json.dumps(result, ensure_ascii=False, default=str),
                    }
                )

            response = await self.client.responses.create(
                model=self.model,
                previous_response_id=response.id,
                input=outputs,  # type: ignore[arg-type]
                tools=TOOLS,  # type: ignore[arg-type]
            )

        return "No pude completar la operación. Intenta nuevamente."
