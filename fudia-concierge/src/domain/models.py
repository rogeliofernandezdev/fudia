from __future__ import annotations

from decimal import Decimal

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


class MenuItem(BaseModel):
    productId: str
    name: str
    description: str = ""
    price: Decimal
    categoryName: str | None = None
    imageUrl: str | None = None
    status: str = "available"
    isCombo: bool = False


class MenuResponse(BaseModel):
    items: list[MenuItem] = Field(default_factory=list)
    currencySymbol: str = ""


class CartLine(BaseModel):
    product_id: str
    name: str
    quantity: float
    unit_price: Decimal
    note: str = ""


class ConversationSession(BaseModel):
    phone: str
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
