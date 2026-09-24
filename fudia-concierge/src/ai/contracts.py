from __future__ import annotations

from typing import Any, Protocol

from src.domain.intents import AgentIntent
from src.domain.models import ConversationSession


class ScopeClassifier(Protocol):
    async def is_in_scope(
        self,
        session: ConversationSession,
        user_message: str,
    ) -> bool: ...


class IntentRouter(Protocol):
    async def route(
        self,
        session: ConversationSession,
        user_message: str,
    ) -> AgentIntent: ...


class ConversationAgent(Protocol):
    async def reply(
        self,
        session: ConversationSession,
        user_message: str,
        tool_handler: Any,
    ) -> str: ...
