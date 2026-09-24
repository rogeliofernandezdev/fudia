from typing import Literal

AgentIntent = Literal["menu", "order", "service"]

INTENT_MENU: AgentIntent = "menu"
INTENT_ORDER: AgentIntent = "order"
INTENT_SERVICE: AgentIntent = "service"
INTENTS: tuple[AgentIntent, ...] = (
    INTENT_MENU,
    INTENT_ORDER,
    INTENT_SERVICE,
)
