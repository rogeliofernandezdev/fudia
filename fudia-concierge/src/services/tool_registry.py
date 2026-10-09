from __future__ import annotations

from typing import Any

from src.domain.intents import AgentIntent
from src.domain.models import ConversationSession
from src.infrastructure.fudia_client import FudiaClient, FudiaError
from src.metrics import BUSINESS_ACTIONS, TOOL_CALLS
from src.services.menu_tools import MenuTools
from src.services.order_tools import OrderTools
from src.services.service_tools import ServiceTools
from src.services.tooling import ToolContext

MENU_TOOLS = frozenset(
    {
        "search_menu",
        "get_modifier_options",
        "get_combo_options",
    }
)
ORDER_TOOLS = frozenset(
    {
        *MENU_TOOLS,
        "add_item",
        "add_modified_item",
        "add_combo_item",
        "remove_item",
        "view_cart",
        "prepare_confirmation",
        "confirm_order",
    }
)
SERVICE_TOOLS = frozenset(
    {
        "request_bill",
        "request_human",
    }
)
TOOLS_BY_INTENT: dict[AgentIntent, frozenset[str]] = {
    "menu": MENU_TOOLS,
    "order": ORDER_TOOLS,
    "service": SERVICE_TOOLS,
}


class ToolRegistry:
    def __init__(
        self,
        fudia: FudiaClient,
        session: ConversationSession,
        user_message: str,
    ) -> None:
        context = ToolContext(fudia, session, user_message)
        self.menu = MenuTools(context)
        self.order = OrderTools(context)
        self.service = ServiceTools(context)

    async def execute_for(
        self,
        intent: AgentIntent,
        name: str,
        args: dict[str, Any],
    ) -> dict[str, Any]:
        if name not in TOOLS_BY_INTENT[intent]:
            TOOL_CALLS.labels(
                agent=intent,
                tool=name,
                result="denied",
            ).inc()
            return {
                "ok": False,
                "code": "tool_not_allowed_for_agent",
            }
        try:
            if intent == "menu":
                result = await self.menu.execute(
                    name,
                    args,
                )
            elif intent == "order":
                result = await self.order.execute(
                    name,
                    args,
                )
            else:
                result = await self.service.execute(
                    name,
                    args,
                )
            outcome = (
                "ok"
                if result.get("ok") is True
                else "error"
            )
            TOOL_CALLS.labels(
                agent=intent,
                tool=name,
                result=outcome,
            ).inc()
            if name in {
                "confirm_order",
                "request_bill",
                "request_human",
            }:
                BUSINESS_ACTIONS.labels(
                    action=name,
                    result=outcome,
                ).inc()
            return result
        except FudiaError as exc:
            TOOL_CALLS.labels(
                agent=intent,
                tool=name,
                result="backend_error",
            ).inc()
            return {
                "ok": False,
                "code": exc.code,
                "message": exc.message,
                "status": exc.status_code,
            }

    async def execute_any(
        self,
        name: str,
        args: dict[str, Any],
    ) -> dict[str, Any]:
        try:
            if name in ORDER_TOOLS:
                result = await self.order.execute(
                    name,
                    args,
                )
            elif name in SERVICE_TOOLS:
                result = await self.service.execute(
                    name,
                    args,
                )
            else:
                return {
                    "ok": False,
                    "code": "unknown_tool",
                }
            return result
        except FudiaError as exc:
            return {
                "ok": False,
                "code": exc.code,
                "message": exc.message,
                "status": exc.status_code,
            }
