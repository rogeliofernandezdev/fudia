from typing import Literal

AgentIntent = Literal["menu", "order", "service"]
ConciergeRoute = Literal[
    "out_of_scope",
    "menu",
    "order",
    "service",
]

INTENT_MENU: AgentIntent = "menu"
INTENT_ORDER: AgentIntent = "order"
INTENT_SERVICE: AgentIntent = "service"
INTENTS: tuple[AgentIntent, ...] = (
    INTENT_MENU,
    INTENT_ORDER,
    INTENT_SERVICE,
)

ROUTE_OUT_OF_SCOPE: ConciergeRoute = "out_of_scope"
ROUTES: tuple[ConciergeRoute, ...] = (
    ROUTE_OUT_OF_SCOPE,
    INTENT_MENU,
    INTENT_ORDER,
    INTENT_SERVICE,
)
