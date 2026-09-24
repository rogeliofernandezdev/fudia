from typing import TypedDict

from src.domain.intents import ConciergeRoute
from src.domain.models import ConversationSession


class ConciergeGraphState(TypedDict):
    session: ConversationSession
    user_message: str
    intent: ConciergeRoute
    reply: str
