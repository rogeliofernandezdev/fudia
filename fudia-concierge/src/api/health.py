from __future__ import annotations

import asyncio

from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse

router = APIRouter()


@router.get("/health")
@router.get("/live")
async def live() -> dict[str, str]:
    return {
        "status": "ok",
        "service": "fudia-concierge",
        "version": "0.3.0",
    }


@router.get("/ready")
async def ready(
    request: Request,
) -> JSONResponse:
    queue = request.app.state.inbound_queue
    store = request.app.state.conversation_store
    fudia = request.app.state.fudia

    results = await asyncio.gather(
        queue.ping(),
        store.ping(),
        fudia.ready(),
        return_exceptions=True,
    )
    checks = {
        "queue": results[0] is True,
        "sessionStore": results[1] is True,
        "foodsBackend": results[2] is True,
    }
    is_ready = all(checks.values())
    return JSONResponse(
        status_code=200 if is_ready else 503,
        content={
            "status": (
                "ready"
                if is_ready
                else "not_ready"
            ),
            "checks": checks,
        },
    )
