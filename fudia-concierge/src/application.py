from __future__ import annotations

import asyncio
import contextlib
import logging
from pathlib import Path

from fastapi import FastAPI

from config.settings import Settings, settings
from src.api.health import router as health_router
from src.api.start import router as start_router
from src.api.webhook import router as webhook_router
from src.infrastructure.ai_factory import build_ai
from src.infrastructure.concurrency import RedisConversationLock
from src.infrastructure.fudia_client import FudiaClient
from src.infrastructure.queue import RedisInboundQueue
from src.infrastructure.rate_limit import RedisRateLimiter
from src.infrastructure.start_codes import RedisTableStartCodes
from src.infrastructure.state_store import RedisConversationStore
from src.infrastructure.whatsapp_adapter import MetaWhatsAppAdapter
from src.services.concierge_service import ConciergeService
from src.services.queue_worker import QueueWorker


def create_app(
    app_settings: Settings | None = None,
) -> FastAPI:
    cfg = app_settings or settings
    redis_url = cfg.redis_connection_url
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
    ai = build_ai(cfg, prompts)
    fudia = FudiaClient(
        cfg.fudia_api_url,
        cfg.fudia_concierge_api_key,
    )
    store = RedisConversationStore(
        redis_url,
        cfg.session_ttl_seconds,
        record_ttl_seconds=cfg.processing_record_ttl_seconds,
        pending_record_ttl_seconds=(
            cfg.processing_pending_record_ttl_seconds
        ),
        memory_ttl_seconds=cfg.customer_memory_ttl_seconds,
    )
    conversation_lock = RedisConversationLock(
        redis_url,
        cfg.conversation_lock_ttl_seconds,
        cfg.conversation_lock_wait_seconds,
    )
    start_codes = RedisTableStartCodes(
        redis_url,
        cfg.start_code_ttl_seconds,
    )
    rate_limiter = RedisRateLimiter(
        redis_url,
        cfg.rate_limit_messages,
        cfg.rate_limit_window_seconds,
    )
    inbound_queue = RedisInboundQueue(
        redis_url,
        stream=cfg.queue_stream,
        group=cfg.queue_group,
        consumer=cfg.queue_consumer,
        max_attempts=cfg.queue_max_attempts,
        visibility_timeout_ms=(
            cfg.queue_visibility_timeout_ms
        ),
        order_ttl_seconds=cfg.queue_order_ttl_seconds,
        order_max_wait_seconds=cfg.queue_order_max_wait_seconds,
        retention_seconds=cfg.queue_retention_seconds,
        dead_letter_retention_seconds=(
            cfg.queue_dead_letter_retention_seconds
        ),
    )
    whatsapp = MetaWhatsAppAdapter(
        cfg.whatsapp_token,
        cfg.whatsapp_phone_id,
        cfg.whatsapp_graph_version,
    )

    service = ConciergeService(
        store,
        fudia,
        ai.router,
        ai.agents,
        conversation_lock=conversation_lock,
        rate_limiter=rate_limiter,
        start_codes=start_codes,
        max_message_chars=cfg.max_message_chars,
        lock_heartbeat_seconds=max(
            0.1,
            cfg.conversation_lock_ttl_seconds / 6.0,
        ),
        lock_ttl_seconds=cfg.conversation_lock_ttl_seconds,
        table_idle_seconds=cfg.table_idle_seconds,
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
        records=store,
        turn_wait_seconds=cfg.queue_turn_wait_seconds,
        reconcile_seconds=cfg.outbound_reconcile_seconds,
        max_resends=cfg.outbound_max_resends,
        maintenance_interval_seconds=(
            cfg.queue_maintenance_interval_seconds
        ),
    )

    app.state.settings = cfg
    app.state.ai_provider = ai.provider
    app.state.ai_models = ai.models
    app.state.concierge_service = service
    app.state.whatsapp = whatsapp
    app.state.inbound_queue = inbound_queue
    app.state.fudia = fudia
    app.state.start_codes = start_codes
    app.state.conversation_store = store
    app.state.queue_worker = worker
    app.state.queue_worker_task = None

    app.include_router(health_router)
    app.include_router(start_router)
    app.include_router(webhook_router)

    @app.on_event("startup")
    async def startup() -> None:
        await inbound_queue.start()
        if cfg.worker_enabled:
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
        await start_codes.close()
        await fudia.close()
        await store.close()
        await whatsapp.close()
        await ai.close()

    return app
