from __future__ import annotations

import asyncio
import contextlib
import logging
from pathlib import Path

from fastapi import FastAPI
from openai import AsyncOpenAI

from config.settings import Settings, settings
from src.api.health import router as health_router
from src.api.webhook import router as webhook_router
from src.infrastructure.concurrency import RedisConversationLock
from src.infrastructure.fudia_client import FudiaClient
from src.infrastructure.openai_adapter import (
    OpenAIIntentRouter,
    OpenAIScopeClassifier,
    OpenAIToolAgent,
)
from src.infrastructure.queue import RedisInboundQueue
from src.infrastructure.rate_limit import RedisRateLimiter
from src.infrastructure.state_store import RedisConversationStore
from src.infrastructure.tool_schemas import (
    MENU_TOOL_SCHEMAS,
    ORDER_TOOL_SCHEMAS,
    SERVICE_TOOL_SCHEMAS,
)
from src.infrastructure.whatsapp_adapter import MetaWhatsAppAdapter
from src.services.concierge_service import ConciergeService
from src.services.queue_worker import QueueWorker


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
        version="0.3.0",
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
    conversation_lock = RedisConversationLock(
        cfg.redis_url,
        cfg.conversation_lock_ttl_seconds,
        cfg.conversation_lock_wait_seconds,
    )
    rate_limiter = RedisRateLimiter(
        cfg.redis_url,
        cfg.rate_limit_messages,
        cfg.rate_limit_window_seconds,
    )
    inbound_queue = RedisInboundQueue(
        cfg.redis_url,
        stream=cfg.queue_stream,
        group=cfg.queue_group,
        consumer=cfg.queue_consumer,
        max_attempts=cfg.queue_max_attempts,
        visibility_timeout_ms=(
            cfg.queue_visibility_timeout_ms
        ),
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

    service = ConciergeService(
        store,
        fudia,
        scope,
        intent_router,
        agents,  # type: ignore[arg-type]
        conversation_lock=conversation_lock,
        rate_limiter=rate_limiter,
        max_message_chars=cfg.max_message_chars,
    )
    worker = QueueWorker(
        inbound_queue,
        service,
        whatsapp,
        max_concurrency=cfg.queue_worker_concurrency,
        visibility_heartbeat_seconds=max(
            0.1,
            cfg.queue_visibility_timeout_ms / 3000.0,
        ),
    )

    app.state.settings = cfg
    app.state.concierge_service = service
    app.state.whatsapp = whatsapp
    app.state.inbound_queue = inbound_queue
    app.state.fudia = fudia
    app.state.conversation_store = store
    app.state.queue_worker = worker
    app.state.queue_worker_task = None

    app.include_router(health_router)
    app.include_router(webhook_router)

    @app.on_event("startup")
    async def startup() -> None:
        await inbound_queue.start()
        app.state.queue_worker_task = (
            asyncio.create_task(worker.run())
        )

    @app.on_event("shutdown")
    async def shutdown() -> None:
        await worker.stop()
        task = app.state.queue_worker_task
        if task is not None:
            done, _ = await asyncio.wait(
                {task},
                timeout=(
                    cfg.queue_shutdown_grace_seconds
                ),
            )
            if task not in done:
                task.cancel()
                with contextlib.suppress(
                    asyncio.CancelledError
                ):
                    await task
        await inbound_queue.close()
        await conversation_lock.close()
        await rate_limiter.close()
        await fudia.close()
        await store.close()
        await whatsapp.close()
        await openai_client.close()

    return app


app = create_app()
