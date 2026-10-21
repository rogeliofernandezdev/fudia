from __future__ import annotations

from decimal import Decimal
from typing import Any, Literal
from uuid import uuid4

from pydantic import BaseModel, Field


class ChatMessage(BaseModel):
    role: str
    content: str


class TableContext(BaseModel):
    name: str
    seats: int
    zone: str = ""
    organizationName: str
    locationName: str
    conciergeEnabled: bool = False


class MenuItem(BaseModel):
    productId: str
    name: str
    description: str = ""
    price: Decimal
    categoryName: str | None = None
    imageUrl: str | None = None
    status: str = "available"
    isCombo: bool = False
    hasModifiers: bool = False
    # Recomendado por el local en la carta digital; foods-backend lo envía primero.
    recommended: bool = False
    # Porciones o unidades que quedan hoy; None si el producto no controla cantidad.
    remaining: Decimal | None = None


class MenuResponse(BaseModel):
    items: list[MenuItem] = Field(default_factory=list)
    currencySymbol: str = ""


class ComboOption(BaseModel):
    productId: str
    name: str
    surcharge: Decimal
    available: bool


class ComboGroup(BaseModel):
    id: str
    name: str
    required: bool
    minSelections: int
    maxSelections: int
    options: list[ComboOption] = Field(default_factory=list)


class ComboDetail(BaseModel):
    id: str
    name: str
    description: str = ""
    price: Decimal
    imageUrl: str | None = None
    groups: list[ComboGroup] = Field(default_factory=list)


class ModifierOption(BaseModel):
    id: str
    name: str
    surcharge: Decimal


class ModifierGroup(BaseModel):
    id: str
    name: str
    required: bool
    minSelections: int
    maxSelections: int
    options: list[ModifierOption] = Field(default_factory=list)


class ModifierConfig(BaseModel):
    productId: str
    groups: list[ModifierGroup] = Field(default_factory=list)


class CartSelection(BaseModel):
    group_id: str
    group_name: str
    product_id: str
    name: str
    surcharge: Decimal


class CartModifier(BaseModel):
    group_id: str
    group_name: str
    option_id: str
    name: str
    surcharge: Decimal


class CartLine(BaseModel):
    product_id: str
    name: str
    quantity: float
    unit_price: Decimal
    note: str = ""
    item_type: str = "product"
    selections: list[CartSelection] = Field(default_factory=list)
    modifiers: list[CartModifier] = Field(default_factory=list)


class GeoPoint(BaseModel):
    """Ubicación compartida por WhatsApp (pin)."""

    latitude: float = Field(ge=-90, le=90)
    longitude: float = Field(ge=-180, le=180)
    name: str = ""
    address: str = ""


Fulfillment = Literal["delivery", "pickup"]
SessionMode = Literal["table", "delivery"]


class DeliveryDetails(BaseModel):
    """Datos de entrega de un pedido sin mesa.

    `fee`/`distance_km` vienen de la cotización de foods-backend para el pin
    compartido; nunca del texto del cliente ni del modelo.
    """

    fulfillment: Fulfillment | None = None
    customer_name: str = ""
    address: str = ""
    reference: str = ""
    location: GeoPoint | None = None
    covered: bool = False
    distance_km: Decimal | None = None
    fee: Decimal | None = None
    max_distance_km: Decimal | None = None
    currency_symbol: str = ""


class VenueContext(BaseModel):
    organizationName: str
    locationName: str


class CustomerMemory(BaseModel):
    """Último local que el cliente visitó por QR. Sobrevive a la sesión para
    tomar pedidos de delivery/recojo de ese local cuando ya no está en mesa."""

    qr_token: str
    venue: VenueContext
    customer_name: str = ""


class DeliveryTier(BaseModel):
    maxDistanceKm: Decimal
    fee: Decimal


class DeliveryOptions(BaseModel):
    locationName: str
    locationAddress: str = ""
    locationPhone: str = ""
    currencySymbol: str = ""
    pickupAvailable: bool = True
    deliveryAvailable: bool = False
    maxDistanceKm: Decimal = Decimal("0")
    tiers: list[DeliveryTier] = Field(default_factory=list)


class DeliveryQuote(BaseModel):
    covered: bool
    distanceKm: Decimal
    fee: Decimal
    maxDistanceKm: Decimal
    currencySymbol: str = ""


class OrderStatus(BaseModel):
    orderId: str
    code: str
    channel: str
    status: str
    subtotal: Decimal = Decimal("0")
    deliveryFee: Decimal = Decimal("0")
    total: Decimal
    paymentStatus: str = ""
    active: bool


class ConversationSession(BaseModel):
    phone: str
    channel_key: str = "default"
    conversation_id: str = Field(default_factory=lambda: uuid4().hex)
    qr_token: str | None = None
    table: TableContext | None = None
    # "table": atención en mesa abierta por QR. "delivery": la atención en
    # mesa terminó y el bot toma pedidos de delivery o recojo del mismo local.
    mode: SessionMode = "table"
    venue: VenueContext | None = None
    delivery: DeliveryDetails = Field(default_factory=DeliveryDetails)
    last_activity_at: float = 0.0
    cart: list[CartLine] = Field(default_factory=list)
    awaiting_confirmation: bool = False
    pending_order_request_id: str | None = None
    last_order_id: str | None = None
    handoff_pending: bool = False
    def memory(self) -> CustomerMemory | None:
        """What outlives the session: the venue of the last QR context."""
        if not self.qr_token or self.venue is None:
            return None
        return CustomerMemory(
            qr_token=self.qr_token,
            venue=self.venue,
            customer_name=self.delivery.customer_name,
        )

    # Queue sequence of the QR message that opened the current context.
    # Messages enqueued before it never apply to this table.
    context_sequence: int = 0
    messages: list[ChatMessage] = Field(default_factory=list)


class BillItem(BaseModel):
    id: str
    productId: str = ""
    name: str
    qty: Decimal
    unitPrice: Decimal
    note: str = ""
    itemType: str = "product"


class BillSummary(BaseModel):
    orderId: str
    code: str
    tableName: str
    status: str
    currencySymbol: str
    items: list[BillItem] = Field(default_factory=list)
    total: Decimal
    paidAmount: Decimal
    remainingAmount: Decimal
    paymentStatus: str


class OrderResult(BaseModel):
    id: str
    code: str
    status: str
    total: Decimal


class InboundMessage(BaseModel):
    message_id: str
    phone: str
    text: str
    sender_phone_id: str = ""
    # Pin de WhatsApp; `text` lleva entonces una descripción para el historial.
    location: GeoPoint | None = None
    # Set once when the message enters the queue; survives retries and the
    # dead letter.
    received_at: float = 0.0
    # Per-conversation order assigned atomically by Redis at enqueue; filled
    # in on delivery (0 for direct, unqueued calls).
    sequence: int = 0


EffectKind = Literal[
    "cart",
    "confirmation",
    "delivery",
    "create_order",
    "request_handoff",
    "request_bill",
]
EffectStatus = Literal["pending", "done", "failed", "unknown"]
ProcessingStage = Literal[
    "processing",
    "effects_applied",
    "effect_pending",
    "reply_ready",
    "sending",
    "uncertain",
    "sent",
    "failed",
    "superseded",
]
# Stages after which the model never runs again for the same message.
PREPARED_STAGES: frozenset[str] = frozenset(
    {
        "reply_ready",
        "sending",
        "uncertain",
        "sent",
        "failed",
        "superseded",
    }
)
TERMINAL_STAGES: frozenset[str] = frozenset(
    {"sent", "failed", "superseded"}
)


class EffectRecord(BaseModel):
    """One state change performed while processing a message.

    `payload` keeps what is needed to replay a remote operation with the same
    idempotency key; `result` keeps only what a recovery reply needs.
    """

    kind: EffectKind
    status: EffectStatus = "done"
    request_id: str = ""
    payload: dict[str, Any] = Field(default_factory=dict)
    result: dict[str, Any] = Field(default_factory=dict)
    error_code: str = ""


class ProcessingRecord(BaseModel):
    """Recovery log of one inbound message, keyed by its Meta message id."""

    message_id: str
    conversation: str = ""
    # Context (QR) the prepared reply belongs to.
    conversation_id: str = ""
    stage: ProcessingStage = "processing"
    attempts: int = 0
    effects: list[EffectRecord] = Field(default_factory=list)
    reply: str = ""
    send_attempts: int = 0
    resends: int = 0
    last_send_at: float = 0.0
    created_at: float = 0.0
    updated_at: float = 0.0
    last_error: str = ""

    @property
    def prepared(self) -> bool:
        return self.stage in PREPARED_STAGES

    @property
    def terminal(self) -> bool:
        return self.stage in TERMINAL_STAGES

    def pending_effect(self) -> EffectRecord | None:
        for effect in reversed(self.effects):
            if effect.status == "pending":
                return effect
        return None
