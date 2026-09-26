from __future__ import annotations

from typing import Any, Protocol

from src.domain.intents import ConciergeRoute
from src.domain.models import ConversationSession


class ConciergeRouter(Protocol):
    async def route(
        self,
        session: ConversationSession,
        user_message: str,
    ) -> ConciergeRoute: ...


class ConversationAgent(Protocol):
    async def reply(
        self,
        session: ConversationSession,
        user_message: str,
        tool_handler: Any,
    ) -> str: ...
