from __future__ import annotations

import logging

from fastapi import FastAPI

from config.settings import Settings, settings
from src.api.health import router as health_router
from src.api.webhook import router as webhook_router
from src.infrastructure.dedup import RedisMessageDeduplicator
from src.infrastructure.fudia_client import FudiaClient
from src.infrastructure.openai_adapter import OpenAIConciergeEngine
from src.infrastructure.state_store import RedisConversationStore
from src.infrastructure.whatsapp_adapter import MetaWhatsAppAdapter
from src.services.concierge_service import ConciergeService


def create_app(app_settings: Settings | None = None) -> FastAPI:
    cfg = app_settings or settings
    logging.basicConfig(level=getattr(logging, cfg.log_level.upper(), logging.INFO))

    app = FastAPI(title="Fudia Concierge", version="0.1.0")
    fudia = FudiaClient(cfg.fudia_api_url, cfg.fudia_concierge_api_key)
    store = RedisConversationStore(cfg.redis_url, cfg.session_ttl_seconds)
    deduplicator = RedisMessageDeduplicator(cfg.redis_url)
    engine = OpenAIConciergeEngine(cfg.openai_api_key, cfg.openai_model)
    whatsapp = MetaWhatsAppAdapter(
        cfg.whatsapp_token,
        cfg.whatsapp_phone_id,
        cfg.whatsapp_graph_version,
    )

    app.state.settings = cfg
    app.state.concierge_service = ConciergeService(store, fudia, engine)
    app.state.whatsapp = whatsapp
    app.state.deduplicator = deduplicator

    app.include_router(health_router)
    app.include_router(webhook_router)

    @app.on_event("shutdown")
    async def shutdown() -> None:
        await fudia.close()
        await store.close()
        await deduplicator.close()
        await whatsapp.close()

    return app


app = create_app()
