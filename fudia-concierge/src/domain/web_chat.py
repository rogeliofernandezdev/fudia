from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, Field


class WebOrderSummary(BaseModel):
    subtotal: str
    deliveryFee: str | None
    total: str | None
    currencySymbol: str
    fulfillment: Literal["table", "counter", "pickup", "delivery"] | None


class WebAction(BaseModel):
    name: Literal[
        "add_item",
        "add_modified_item",
        "add_combo_item",
        "remove_item",
        "view_cart",
        "prepare_confirmation",
        "confirm_order",
        "set_fulfillment",
        "set_delivery_details",
        "request_bill",
        "request_human",
    ]
    args: dict[str, Any] = Field(default_factory=dict)
    confirmation_ref: str = Field(default="", max_length=100)
