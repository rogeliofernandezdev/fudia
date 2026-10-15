"""Model tiering: a lighter model handles easy work and a stronger one takes
over when the light attempt fails.

Escalation is only safe when the failed attempt changed nothing: re-running a
turn that already added items, requested the bill or called staff would
repeat that action. The wrapper therefore records which tools ran and only
escalates after read-only tool calls.
"""

from __future__ import annotations

import asyncio
import logging
from collections.abc import Awaitable, Callable
from typing import Any

from src.ai.contracts import ConciergeRouter, ConversationAgent
from src.domain.intents import AgentIntent, ConciergeRoute
from src.domain.models import ConversationSession
from src.metrics import MODEL_ESCALATIONS
from src.observability import fingerprint, log_event

logger = logging.getLogger(__name__)

ToolHandler = Callable[[str, dict[str, Any]], Awaitable[dict[str, Any]]]

# Tools without side effects on the cart, the order or the table service.
READ_ONLY_TOOLS = frozenset(
    {
        "search_menu",
        "get_modifier_options",
        "get_combo_options",
        "view_cart",
    }
)

TOOL_LIMIT_REPLY = "No pude completar la operación. Intenta nuevamente."


class AgentIncompleteError(RuntimeError):
    """The agent stopped without a usable answer (tool limit or empty text).

    ``reply`` is the safe text to send when no stronger model can retry.
    """

    def __init__(self, reply: str = TOOL_LIMIT_REPLY) -> None:
        super().__init__(reply)
        self.reply = reply


class TieredAgent:
    def __init__(
        self,
        primary: ConversationAgent,
        escalation: ConversationAgent | None,
        agent_name: AgentIntent,
    ) -> None:
        self.primary = primary
        self.escalation = escalation
        self.agent_name = agent_name

    async def reply(
        self,
        session: ConversationSession,
        user_message: str,
        tool_handler: ToolHandler,
    ) -> str:
        executed: list[str] = []

        async def tracking_handler(
            name: str,
            args: dict[str, Any],
        ) -> dict[str, Any]:
            executed.append(name)
            return await tool_handler(name, args)

        try:
            return await self.primary.reply(
                session,
                user_message,
                tracking_handler,
            )
        except asyncio.CancelledError:
            raise
        except Exception as exc:
            mutated = [
                name for name in executed if name not in READ_ONLY_TOOLS
            ]
            can_escalate = self.escalation is not None and not mutated
            MODEL_ESCALATIONS.labels(
                agent=self.agent_name,
                result="escalated" if can_escalate else "not_escalated",
            ).inc()
            log_event(
                logger,
                "concierge.agent.primary_failed",
                conversation=session.conversation_id,
                phone=fingerprint(session.phone),
                agent=self.agent_name,
                errorType=type(exc).__name__,
                escalated=can_escalate,
                mutatingTools=mutated,
            )
            if self.escalation is None or mutated:
                if isinstance(exc, AgentIncompleteError):
                    return exc.reply
                raise
        assert self.escalation is not None
        try:
            return await self.escalation.reply(
                session,
                user_message,
                tool_handler,
            )
        except AgentIncompleteError as exc:
            return exc.reply


class TieredRouter:
    """Routes with the light model and retries once with the stronger one
    when the light call errors. Classification has no side effects."""

    def __init__(
        self,
        primary: ConciergeRouter,
        fallback: ConciergeRouter | None,
    ) -> None:
        self.primary = primary
        self.fallback = fallback

    async def route(
        self,
        session: ConversationSession,
        user_message: str,
    ) -> ConciergeRoute:
        try:
            return await self.primary.route(session, user_message)
        except asyncio.CancelledError:
            raise
        except Exception as exc:
            if self.fallback is None:
                raise
            MODEL_ESCALATIONS.labels(
                agent="router",
                result="escalated",
            ).inc()
            log_event(
                logger,
                "concierge.router.primary_failed",
                conversation=session.conversation_id,
                phone=fingerprint(session.phone),
                errorType=type(exc).__name__,
            )
            return await self.fallback.route(session, user_message)
