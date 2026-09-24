from __future__ import annotations

from typing import Any
from urllib.parse import quote

import httpx

from src.domain.models import ComboDetail, MenuItem, MenuResponse, OrderResult, TableContext


class FudiaError(RuntimeError):
    def __init__(self, status_code: int, code: str, message: str) -> None:
        super().__init__(message)
        self.status_code = status_code
        self.code = code
        self.message = message


class FudiaClient:
    def __init__(
        self,
        base_url: str,
        api_key: str = "",
        client: httpx.AsyncClient | None = None,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self._external_client = client
        self._client: httpx.AsyncClient | None = client

    def _http(self) -> httpx.AsyncClient:
        if self._client is None:
            self._client = httpx.AsyncClient(timeout=15.0)
        return self._client

    async def close(self) -> None:
        if self._client is not None and self._external_client is None:
            await self._client.aclose()
            self._client = None

    async def _json(self, method: str, path: str, **kwargs: Any) -> dict[str, Any]:
        headers = dict(kwargs.pop("headers", {}))
        if path.startswith("/v1/integrations/concierge/") and self.api_key:
            headers["X-Fudia-Concierge-Key"] = self.api_key
        response = await self._http().request(
            method, self.base_url + path, headers=headers, **kwargs
        )
        try:
            body = response.json()
        except ValueError:
            body = {}
        if not response.is_success:
            raise FudiaError(
                response.status_code,
                str(body.get("code", "fudia_error")),
                str(body.get("message", "Fudia no pudo completar la operación.")),
            )
        return body

    async def resolve_table(self, token: str) -> TableContext:
        body = await self._json("GET", f"/v1/public/tables/{quote(token, safe='')}")
        return TableContext.model_validate(body)

    async def search_menu(
        self, token: str, query: str = "", product_id: str = ""
    ) -> MenuResponse:
        params: dict[str, str] = {}
        if query.strip():
            params["q"] = query.strip()
        if product_id.strip():
            params["productId"] = product_id.strip()
        body = await self._json(
            "GET",
            f"/v1/integrations/concierge/{quote(token, safe='')}/menu",
            params=params,
        )
        return MenuResponse.model_validate(body)

    async def get_product(self, token: str, product_id: str) -> MenuItem | None:
        menu = await self.search_menu(token, product_id=product_id)
        return menu.items[0] if menu.items else None

    async def get_combo(self, token: str, product_id: str) -> ComboDetail:
        body = await self._json(
            "GET",
            (
                f"/v1/integrations/concierge/{quote(token, safe='')}"
                f"/combos/{quote(product_id, safe='')}"
            ),
        )
        return ComboDetail.model_validate(body)

    async def create_order(
        self,
        token: str,
        phone: str,
        lines: list[dict[str, Any]],
        conversation_id: str,
    ) -> OrderResult:
        body = await self._json(
            "POST",
            f"/v1/integrations/concierge/{quote(token, safe='')}/orders",
            json={
                "customerPhone": phone,
                "conversationId": conversation_id,
                "items": lines,
            },
        )
        return OrderResult.model_validate(body)
