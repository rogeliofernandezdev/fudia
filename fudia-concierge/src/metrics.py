from prometheus_client import Counter, Gauge, Histogram

INBOUND_MESSAGES = Counter(
    "fudia_concierge_inbound_messages_total",
    "Inbound WhatsApp messages handled by the webhook.",
    ["result"],
)
SCOPE_DECISIONS = Counter(
    "fudia_concierge_scope_decisions_total",
    "Scope gate decisions.",
    ["decision"],
)
INTENT_ROUTES = Counter(
    "fudia_concierge_intent_routes_total",
    "Intent router decisions.",
    ["intent"],
)
AGENT_REQUESTS = Counter(
    "fudia_concierge_agent_requests_total",
    "Specialist agent completions.",
    ["agent", "result"],
)
AGENT_DURATION = Histogram(
    "fudia_concierge_agent_duration_seconds",
    "Specialist agent processing latency.",
    ["agent"],
)
TOOL_CALLS = Counter(
    "fudia_concierge_tool_calls_total",
    "Tool executions by specialist.",
    ["agent", "tool", "result"],
)
QUEUE_DELIVERIES = Counter(
    "fudia_concierge_queue_deliveries_total",
    "Queue processing outcomes.",
    ["result"],
)
QUEUE_IN_FLIGHT = Gauge(
    "fudia_concierge_queue_in_flight",
    "Queue deliveries currently being processed.",
)
QUEUE_DURATION = Histogram(
    "fudia_concierge_queue_duration_seconds",
    "End-to-end queue delivery processing latency.",
)
RATE_LIMITED = Counter(
    "fudia_concierge_rate_limited_total",
    "Messages rejected by the per-conversation rate limiter.",
)
BUSINESS_ACTIONS = Counter(
    "fudia_concierge_business_actions_total",
    "High-value deterministic operations.",
    ["action", "result"],
)
BACKEND_REQUESTS = Counter(
    "fudia_concierge_backend_requests_total",
    "Requests from Concierge to foods-backend.",
    ["method", "endpoint", "result"],
)
BACKEND_DURATION = Histogram(
    "fudia_concierge_backend_duration_seconds",
    "foods-backend request latency.",
    ["method", "endpoint"],
)
WHATSAPP_OUTBOUND = Counter(
    "fudia_concierge_whatsapp_outbound_total",
    "Outbound WhatsApp send outcomes.",
    ["result"],
)
