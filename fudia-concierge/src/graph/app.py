from __future__ import annotations

from collections.abc import Awaitable, Callable

from langgraph.graph import END, StateGraph

from src.domain.models import ConversationSession
from src.graph.state import ConciergeGraphState

Processor = Callable[[ConversationSession, str], Awaitable[str]]


def build_graph(processor: Processor):
    graph = StateGraph(ConciergeGraphState)

    async def concierge_node(state: ConciergeGraphState) -> dict[str, str]:
        reply = await processor(state["session"], state["user_message"])
        return {"reply": reply}

    graph.add_node("concierge", concierge_node)
    graph.set_entry_point("concierge")
    graph.add_edge("concierge", END)
    return graph.compile()
