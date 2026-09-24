from __future__ import annotations

from collections.abc import Awaitable, Callable
from typing import Any

from langgraph.graph import END, StateGraph

from src.domain.intents import AgentIntent
from src.domain.models import ConversationSession
from src.graph.state import ConciergeGraphState

Router = Callable[
    [ConversationSession, str],
    Awaitable[AgentIntent],
]
Processor = Callable[
    [ConversationSession, str],
    Awaitable[str],
]


def build_graph(
    router: Router,
    processors: dict[AgentIntent, Processor],
) -> Any:
    graph = StateGraph(ConciergeGraphState)

    async def route_node(
        state: ConciergeGraphState,
    ) -> dict[str, AgentIntent]:
        intent = await router(
            state["session"],
            state["user_message"],
        )
        return {"intent": intent}

    def specialist_node(
        intent: AgentIntent,
    ) -> Callable[
        [ConciergeGraphState],
        Awaitable[dict[str, str]],
    ]:
        async def run(
            state: ConciergeGraphState,
        ) -> dict[str, str]:
            reply = await processors[intent](
                state["session"],
                state["user_message"],
            )
            return {"reply": reply}

        return run

    graph.add_node("route", route_node)
    graph.add_node(  # type: ignore[arg-type]
        "menu",
        specialist_node("menu"),
    )
    graph.add_node(  # type: ignore[arg-type]
        "order",
        specialist_node("order"),
    )
    graph.add_node(  # type: ignore[arg-type]
        "service",
        specialist_node("service"),
    )
    graph.set_entry_point("route")
    graph.add_conditional_edges(
        "route",
        lambda state: state["intent"],
        {
            "menu": "menu",
            "order": "order",
            "service": "service",
        },
    )
    graph.add_edge("menu", END)
    graph.add_edge("order", END)
    graph.add_edge("service", END)
    return graph.compile()
