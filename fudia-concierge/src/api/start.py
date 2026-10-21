from __future__ import annotations

import logging

import httpx
from fastapi import APIRouter, HTTPException, Request
from fastapi.responses import RedirectResponse

from src.observability import log_event
from src.services.ports import FudiaError
from src.services.start_codes import start_message

router = APIRouter(tags=["whatsapp"])
logger = logging.getLogger(__name__)


@router.get("/start/{token}")
async def start_whatsapp(
    token: str,
    request: Request,
) -> RedirectResponse:
    try:
        table = await request.app.state.fudia.resolve_table(
            token
        )
    except FudiaError as exc:
        status_code = (
            404
            if exc.status_code == 404
            else 503
        )
        raise HTTPException(
            status_code=status_code,
            detail=(
                "El QR no está activo o ya no "
                "corresponde a una mesa."
                if status_code == 404
                else "No pudimos validar el QR."
            ),
        ) from exc

    if not table.conciergeEnabled:
        raise HTTPException(
            status_code=404,
            detail=(
                "Fudia Concierge no está habilitado "
                "para este restaurante."
            ),
        )

    # Friendly start message with a short table code; if the shared code
    # store is unavailable the legacy FUDIA:<token> message still works.
    text = ""
    start_codes = getattr(request.app.state, "start_codes", None)
    if start_codes is not None:
        try:
            text = start_message(table.name, await start_codes.issue(token))
        except Exception as exc:
            log_event(
                logger,
                "concierge.start_code.failed",
                errorType=type(exc).__name__,
            )
    try:
        url = await request.app.state.whatsapp.start_url(
            token,
            text,
        )
    except (httpx.HTTPError, RuntimeError) as exc:
        raise HTTPException(
            status_code=503,
            detail=(
                "No pudimos abrir el canal oficial "
                "de WhatsApp."
            ),
        ) from exc

    return RedirectResponse(
        url=url,
        status_code=307,
    )
