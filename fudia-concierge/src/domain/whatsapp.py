"""Transport references and verified identity; no provider credentials."""

from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field

WhatsAppProvider = Literal["META_CLOUD_API", "WHATSMEOW", "FUDIA_WEB"]


class ChannelRoute(BaseModel):
    routingKey: str
    countryCode: str
    provider: WhatsAppProvider = "META_CLOUD_API"
    providerKey: str
    phoneNumber: str
    active: bool = False
    legacy: bool = False


class VerifiedCustomer(BaseModel):
    provider: WhatsAppProvider = "META_CLOUD_API"
    subject: str
    verifiedPhone: str = ""


class RestaurantOption(BaseModel):
    organizationId: str
    locationId: str
    organizationName: str
    locationName: str
    visitCount: int = 0
    qrToken: str = ""


class RestaurantPage(BaseModel):
    items: list[RestaurantOption] = Field(default_factory=list)
    hasMore: bool = False
    offset: int = 0


class RestaurantSelection(BaseModel):
    state: Literal["AWAITING_RESTAURANT_SELECTION"] = "AWAITING_RESTAURANT_SELECTION"
    options: list[RestaurantOption] = Field(default_factory=list)
    only_visited: bool = True
    offset: int = 0
    has_more: bool = False
