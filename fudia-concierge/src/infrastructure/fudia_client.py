from __future__ import annotations

import asyncio
import time
from typing import Any
from urllib.parse import quote

import httpx

from src.domain.models import (
    BillSummary,
    ComboDetail,
    MenuItem,
    MenuResponse,
    ModifierConfig,
    OrderResult,
    TableContext,
)
from src.metrics import BACKEND_DURATION, BACKEND_REQUESTS
from src.observability import current_trace_id


class FudiaError(RuntimeError):
    def __init__(
        self,
        status_code: int,
        code: str,
        message: str,
    ) -> None:
        super().__init__(message)
        self.status_code = status_code
        self.code = code
        self.message = message


def _endpoint_label(path: str) -> str:
    if "/menu" in path:
        return "menu"
    if "/orders" in path:
        return "orders"
    if "/bill" in path:
        return "bill"
    if "/handoffs" in path:
        return "handoffs"
    if "/combos/" in path:
        return "combos"
    if "/modifiers" in path:
        return "modifiers"
    if "/v1/public/tables/" in path:
        return "table"
    return "other"


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
            self._client = httpx.AsyncClient(
                timeout=httpx.Timeout(
                    15.0,
                    connect=3.0,
                )
            )
        return self._client

    async def close(self) -> None:
        if (
            self._client is not None
            and self._external_client is None
        ):
            await self._client.aclose()
            self._client = None

    async def _json(
        self,
        method: str,
        path: str,
        *,
        retryable: bool = False,
        **kwargs: Any,
    ) -> dict[str, Any]:
        headers = dict(
            kwargs.pop("headers", {})
        )
        trace_id = current_trace_id()
        if trace_id:
            headers["X-Correlation-ID"] = trace_id
        started = time.perf_counter()
        endpoint = _endpoint_label(path)
        if (
            path.startswith(
                "/v1/integrations/concierge/"
            )
            and self.api_key
        ):
            headers[
                "X-Fudia-Concierge-Key"
            ] = self.api_key

        safe_retry = (
            retryable
            or method.upper() == "GET"
        )
        attempts = 3 if safe_retry else 1
        response: httpx.Response | None = None

        for attempt in range(attempts):
            try:
                response = await self._http().request(
                    method,
                    self.base_url + path,
                    headers=headers,
                    **kwargs,
                )
            except httpx.TransportError as exc:
                if attempt + 1 >= attempts:
                    raise FudiaError(
                        503,
                        "fudia_unreachable",
                        (
                            "No pudimos comunicarnos "
                            "con Fudia."
                        ),
                    ) from exc
                await asyncio.sleep(
                    0.2 * (2**attempt)
                )
                continue

            should_retry = (
                response.status_code == 429
                or response.status_code >= 500
            )
            if (
                should_retry
                and attempt + 1 < attempts
            ):
                retry_after = response.headers.get(
                    "Retry-After",
                    "",
                )
                try:
                    delay = min(
                        float(retry_after),
                        5.0,
                    )
                except ValueError:
                    delay = 0.2 * (2**attempt)
                await asyncio.sleep(delay)
                continue
            break

        if response is None:
            raise FudiaError(
                503,
                "fudia_unreachable",
                "No pudimos comunicarnos con Fudia.",
            )

        try:
            body = response.json()
        except ValueError:
            body = {}

        duration = time.perf_counter() - started
        if not response.is_success:
            BACKEND_REQUESTS.labels(
                method=method.upper(),
                endpoint=endpoint,
                result="error",
            ).inc()
            BACKEND_DURATION.labels(
                method=method.upper(),
                endpoint=endpoint,
            ).observe(duration)
            raise FudiaError(
                response.status_code,
                str(
                    body.get(
                        "code",
                        "fudia_error",
                    )
                ),
                str(
                    body.get(
                        "message",
                        (
                            "Fudia no pudo completar "
                            "la operación."
                        ),
                    )
                ),
            )
        BACKEND_REQUESTS.labels(
            method=method.upper(),
            endpoint=endpoint,
            result="success",
        ).inc()
        BACKEND_DURATION.labels(
            method=method.upper(),
            endpoint=endpoint,
        ).observe(duration)
        return body

    async def ready(self) -> bool:
        try:
            await self._http().get(
                self.base_url + "/",
                timeout=2.0,
            )
            return True
        except httpx.TransportError:
            return False

    async def resolve_table(
        self,
        token: str,
    ) -> TableContext:
        body = await self._json(
            "GET",
            (
                "/v1/public/tables/"
                f"{quote(token, safe='')}"
            ),
        )
        return TableContext.model_validate(body)

    async def search_menu(
        self,
        token: str,
        query: str = "",
        product_id: str = "",
    ) -> MenuResponse:
        params: dict[str, str] = {}
        if query.strip():
            params["q"] = query.strip()
        if product_id.strip():
            params[
                "productId"
            ] = product_id.strip()
        body = await self._json(
            "GET",
            (
                "/v1/integrations/concierge/"
                f"{quote(token, safe='')}/menu"
            ),
            params=params,
        )
        return MenuResponse.model_validate(body)

    async def get_product(
        self,
        token: str,
        product_id: str,
    ) -> MenuItem | None:
        menu = await self.search_menu(
            token,
            product_id=product_id,
        )
        return (
            menu.items[0]
            if menu.items
            else None
        )

    async def get_combo(
        self,
        token: str,
        product_id: str,
    ) -> ComboDetail:
        body = await self._json(
            "GET",
            (
                "/v1/integrations/concierge/"
                f"{quote(token, safe='')}"
                "/combos/"
                f"{quote(product_id, safe='')}"
            ),
        )
        return ComboDetail.model_validate(body)

    async def get_modifiers(
        self,
        token: str,
        product_id: str,
    ) -> ModifierConfig:
        body = await self._json(
            "GET",
            (
                "/v1/integrations/concierge/"
                f"{quote(token, safe='')}"
                "/products/"
                f"{quote(product_id, safe='')}"
                "/modifiers"
            ),
        )
        return ModifierConfig.model_validate(body)

    async def request_handoff(
        self,
        token: str,
        phone: str,
        conversation_id: str,
        reason: str,
    ) -> dict[str, Any]:
        return await self._json(
            "POST",
            (
                "/v1/integrations/concierge/"
                f"{quote(token, safe='')}"
                "/handoffs"
            ),
            json={
                "customerPhone": phone,
                "conversationId": conversation_id,
                "reason": reason,
            },
        )

    async def get_handoff_status(
        self,
        token: str,
        conversation_id: str,
    ) -> dict[str, Any]:
        return await self._json(
            "GET",
            (
                "/v1/integrations/concierge/"
                f"{quote(token, safe='')}"
                "/handoffs/"
                f"{quote(conversation_id, safe='')}"
            ),
        )

    async def create_order(
        self,
        token: str,
        phone: str,
        lines: list[dict[str, Any]],
        conversation_id: str,
        request_id: str,
    ) -> OrderResult:
        body = await self._json(
            "POST",
            (
                "/v1/integrations/concierge/"
                f"{quote(token, safe='')}/orders"
            ),
            retryable=True,
            json={
                "customerPhone": phone,
                "conversationId": conversation_id,
                "requestId": request_id,
                "items": lines,
            },
        )
        return OrderResult.model_validate(body)

    async def request_bill(
        self,
        token: str,
        phone: str,
        conversation_id: str,
    ) -> BillSummary:
        body = await self._json(
            "POST",
            (
                "/v1/integrations/concierge/"
                f"{quote(token, safe='')}/bill"
            ),
            json={
                "customerPhone": phone,
                "conversationId": conversation_id,
            },
        )
        return BillSummary.model_validate(body)
