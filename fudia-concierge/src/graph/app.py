from __future__ import annotations

from collections.abc import Awaitable, Callable
from typing import Any

from langgraph.graph import END, StateGraph

from src.domain.intents import AgentIntent, ConciergeRoute
from src.domain.models import ConversationSession
from src.graph.state import ConciergeGraphState

Router = Callable[
    [ConversationSession, str],
    Awaitable[ConciergeRoute],
]
Processor = Callable[
    [ConversationSession, str],
    Awaitable[str],
]


def build_graph(
    router: Router,
    processors: dict[AgentIntent, Processor],
    out_of_scope_reply: str,
) -> Any:
    graph = StateGraph(ConciergeGraphState)

    async def route_node(
        state: ConciergeGraphState,
    ) -> dict[str, ConciergeRoute]:
        route = await router(
            state["session"],
            state["user_message"],
        )
        return {"intent": route}

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

    async def out_of_scope_node(
        state: ConciergeGraphState,
    ) -> dict[str, str]:
        return {"reply": out_of_scope_reply}

    graph.add_node("route", route_node)
    menu_node: Any = specialist_node("menu")
    order_node: Any = specialist_node("order")
    service_node: Any = specialist_node("service")
    graph.add_node("menu", menu_node)
    graph.add_node("order", order_node)
    graph.add_node("service", service_node)
    graph.add_node("out_of_scope", out_of_scope_node)
    graph.set_entry_point("route")
    graph.add_conditional_edges(
        "route",
        lambda state: state["intent"],
        {
            "menu": "menu",
            "order": "order",
            "service": "service",
            "out_of_scope": "out_of_scope",
        },
    )
    graph.add_edge("menu", END)
    graph.add_edge("order", END)
    graph.add_edge("service", END)
    graph.add_edge("out_of_scope", END)
    return graph.compile()
