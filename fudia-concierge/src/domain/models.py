from __future__ import annotations

from decimal import Decimal
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
    whatsappPhone: str = ""


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


class ConversationSession(BaseModel):
    phone: str
    conversation_id: str = Field(default_factory=lambda: uuid4().hex)
    qr_token: str | None = None
    table: TableContext | None = None
    cart: list[CartLine] = Field(default_factory=list)
    awaiting_confirmation: bool = False
    last_order_id: str | None = None
    messages: list[ChatMessage] = Field(default_factory=list)


class OrderResult(BaseModel):
    id: str
    code: str
    status: str
    total: Decimal
