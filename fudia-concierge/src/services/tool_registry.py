from __future__ import annotations

from typing import Any

from src.domain.intents import AgentIntent
from src.domain.models import ConversationSession
from src.infrastructure.fudia_client import FudiaClient, FudiaError
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
            return {
                "ok": False,
                "code": "tool_not_allowed_for_agent",
            }
        try:
            if intent == "menu":
                return await self.menu.execute(name, args)
            if intent == "order":
                return await self.order.execute(name, args)
            return await self.service.execute(name, args)
        except FudiaError as exc:
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
                return await self.order.execute(name, args)
            if name in SERVICE_TOOLS:
                return await self.service.execute(name, args)
            return {"ok": False, "code": "unknown_tool"}
        except FudiaError as exc:
            return {
                "ok": False,
                "code": exc.code,
                "message": exc.message,
                "status": exc.status_code,
            }
