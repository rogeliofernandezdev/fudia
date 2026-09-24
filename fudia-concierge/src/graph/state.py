from typing import TypedDict

from src.domain.models import ConversationSession


class ConciergeGraphState(TypedDict):
    session: ConversationSession
    user_message: str
    reply: str
