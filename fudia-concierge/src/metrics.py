from prometheus_client import Counter, Gauge, Histogram

INBOUND_MESSAGES = Counter(
    "fudia_concierge_inbound_messages_total",
    "Inbound WhatsApp messages handled by the webhook.",
    ["result"],
)
SCOPE_DECISIONS = Counter(
    "fudia_concierge_scope_decisions_total",
    "Scope decisions produced by the unified router.",
    ["decision"],
)
INTENT_ROUTES = Counter(
    "fudia_concierge_intent_routes_total",
    "Routing decisions produced by the unified router.",
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
MODEL_REQUESTS = Counter(
    "fudia_concierge_model_requests_total",
    "Model calls by role, model tier and outcome.",
    ["role", "tier", "result"],
)
MODEL_TOKENS = Counter(
    "fudia_concierge_model_tokens_total",
    "Model tokens consumed by role, tier and direction.",
    ["role", "tier", "direction"],
)
MODEL_ESCALATIONS = Counter(
    "fudia_concierge_model_escalations_total",
    "Light-model failures and whether the stronger model took over.",
    ["agent", "result"],
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
LOCK_LOST = Counter(
    "fudia_concierge_lock_lost_total",
    "Messages interrupted because the conversation lease was lost.",
)
PROCESSING_RECOVERIES = Counter(
    "fudia_concierge_processing_recoveries_total",
    "Messages answered from persisted effects instead of re-running the model.",
    ["effect", "result"],
)
MESSAGES_SUPERSEDED = Counter(
    "fudia_concierge_messages_superseded_total",
    "Messages or replies dropped because a newer QR context replaced them.",
    ["phase"],
)
OUTBOUND_REPLIES = Counter(
    "fudia_concierge_outbound_replies_total",
    "Prepared reply transitions in the outbound phase.",
    ["state"],
)
QUEUE_ORDER_GAPS = Counter(
    "fudia_concierge_queue_order_gaps_total",
    "Messages processed without their predecessor after the order wait limit.",
)
QUEUE_PENDING = Gauge(
    "fudia_concierge_queue_pending",
    "Delivered but unacknowledged stream entries (same value on every replica).",
)
QUEUE_DEAD_LETTER = Gauge(
    "fudia_concierge_queue_dead_letter",
    "Entries in the dead-letter stream (same value on every replica).",
)
QUEUE_STREAM_LENGTH = Gauge(
    "fudia_concierge_queue_stream_length",
    "Entries retained in the inbound stream (same value on every replica).",
)
QUEUE_TRIMMED = Counter(
    "fudia_concierge_queue_trimmed_total",
    "Stream entries removed by the retention policy.",
    ["stream"],
)
