from typing import TypedDict

from src.domain.intents import AgentIntent
from src.domain.models import ConversationSession


class ConciergeGraphState(TypedDict):
    session: ConversationSession
    user_message: str
    intent: AgentIntent
    reply: str
