from typing import Any

from src.domain.models import ConversationSession
from src.infrastructure.fudia_client import FudiaClient
from src.services.confirmation import explicit_confirmation
from src.services.tool_registry import ToolRegistry

__all__ = ["ConciergeTools", "explicit_confirmation"]


class ConciergeTools:
    """Compatibility facade over domain-specific tool services."""

    def __init__(
        self,
        fudia: FudiaClient,
        session: ConversationSession,
        user_message: str,
    ) -> None:
        self.registry = ToolRegistry(
            fudia,
            session,
            user_message,
        )

    async def execute(
        self,
        name: str,
        args: dict[str, Any],
    ) -> dict[str, Any]:
        return await self.registry.execute_any(
            name,
            args,
        )
