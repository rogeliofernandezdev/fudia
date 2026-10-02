from __future__ import annotations

import httpx
from fastapi import APIRouter, HTTPException, Request
from fastapi.responses import RedirectResponse

from src.infrastructure.fudia_client import FudiaError

router = APIRouter(tags=["whatsapp"])


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

    try:
        url = await request.app.state.whatsapp.start_url(
            token
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
