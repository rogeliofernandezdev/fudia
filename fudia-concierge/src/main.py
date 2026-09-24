from __future__ import annotations

import logging
from pathlib import Path

from fastapi import FastAPI
from openai import AsyncOpenAI

from config.settings import Settings, settings
from src.api.health import router as health_router
from src.api.webhook import router as webhook_router
from src.infrastructure.dedup import RedisMessageDeduplicator
from src.infrastructure.fudia_client import FudiaClient
from src.infrastructure.openai_adapter import (
    OpenAIIntentRouter,
    OpenAIScopeClassifier,
    OpenAIToolAgent,
)
from src.infrastructure.state_store import RedisConversationStore
from src.infrastructure.tool_schemas import (
    MENU_TOOL_SCHEMAS,
    ORDER_TOOL_SCHEMAS,
    SERVICE_TOOL_SCHEMAS,
)
from src.infrastructure.whatsapp_adapter import MetaWhatsAppAdapter
from src.services.concierge_service import ConciergeService


def create_app(
    app_settings: Settings | None = None,
) -> FastAPI:
    cfg = app_settings or settings
    logging.basicConfig(
        level=getattr(
            logging,
            cfg.log_level.upper(),
            logging.INFO,
        )
    )

    app = FastAPI(
        title="Fudia Concierge",
        version="0.2.0",
    )
    prompts = (
        Path(__file__).resolve().parents[1]
        / "prompts"
    )
    openai_client = AsyncOpenAI(
        api_key=cfg.openai_api_key
    )
    fudia = FudiaClient(
        cfg.fudia_api_url,
        cfg.fudia_concierge_api_key,
    )
    store = RedisConversationStore(
        cfg.redis_url,
        cfg.session_ttl_seconds,
    )
    deduplicator = RedisMessageDeduplicator(
        cfg.redis_url
    )
    whatsapp = MetaWhatsAppAdapter(
        cfg.whatsapp_token,
        cfg.whatsapp_phone_id,
        cfg.whatsapp_graph_version,
    )

    scope = OpenAIScopeClassifier(
        openai_client,
        cfg.openai_model,
        prompts / "scope_router.md",
    )
    intent_router = OpenAIIntentRouter(
        openai_client,
        cfg.openai_model,
        prompts / "intent_router.md",
    )
    agents = {
        "menu": OpenAIToolAgent(
            openai_client,
            cfg.openai_model,
            prompts / "system.md",
            prompts / "menu_agent.md",
            MENU_TOOL_SCHEMAS,
            "menu",
        ),
        "order": OpenAIToolAgent(
            openai_client,
            cfg.openai_model,
            prompts / "system.md",
            prompts / "order_agent.md",
            ORDER_TOOL_SCHEMAS,
            "order",
        ),
        "service": OpenAIToolAgent(
            openai_client,
            cfg.openai_model,
            prompts / "system.md",
            prompts / "service_agent.md",
            SERVICE_TOOL_SCHEMAS,
            "service",
        ),
    }

    app.state.settings = cfg
    app.state.concierge_service = ConciergeService(
        store,
        fudia,
        scope,
        intent_router,
        agents,  # type: ignore[arg-type]
    )
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
        await openai_client.close()

    return app


app = create_app()
