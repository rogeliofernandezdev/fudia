"""Local visual QA only: test restaurant, no real model, orders or credentials.

Run uvicorn tests.web_chat_preview:app --host 127.0.0.1 --port 8019.
Use a separate Next instance with FUDIA_CONCIERGE_URL=http://localhost:8019
and FUDIA_CONCIERGE_API_KEY=local-web-chat-review-key-at-least-32-chars.
"""

from __future__ import annotations

import asyncio

from fastapi import FastAPI

from config.settings import Settings
from src.api.web_chat import router
from src.infrastructure.concurrency import RedisConversationLock
from src.infrastructure.queue import RedisInboundQueue
from src.infrastructure.state_store import RedisConversationStore
from src.infrastructure.whatsapp_providers import WhatsAppProviders
from src.services.concierge_service import ConciergeService
from src.services.queue_worker import QueueWorker
from tests.test_web_chat import Registry


class PreviewRestaurant(Registry):
    async def get_order_status(self, token, order_id, conversation_id=""):
        from src.domain.models import OrderStatus

        return OrderStatus(
            orderId=order_id,
            code="PED-PRUEBA",
            channel="whatsapp",
            status="nuevo",
            total="32.50",
            active=True,
        )

    async def resolve_table(self, token: str):
        return (await super().resolve_table(token)).model_copy(
            update={
                "organizationName": "Restaurante de prueba",
                "locationName": "Local de prueba",
            }
        )


class PreviewAgent:
    async def reply(self, *args):
        return (
            "Puedes elegir un plato de la carta o revisar tu pedido. "
            "Esta es una conversación de prueba local."
        )


app = FastAPI()
app.include_router(router)
cfg = Settings(
    _env_file=None,
    web_chat_url="http://localhost:5195",
    redis_url="redis://localhost:56379/13",
    fudia_concierge_api_key="local-web-chat-review-key-at-least-32-chars",
)
backend = PreviewRestaurant()
transport = WhatsAppProviders(cfg, backend)
store = RedisConversationStore(cfg.redis_connection_url, cfg.session_ttl_seconds)
queue = RedisInboundQueue(cfg.redis_connection_url, stream="fudia:web-preview", group="web-preview")
service = ConciergeService(
    store, backend, PreviewAgent(), conversation_lock=RedisConversationLock(cfg.redis_connection_url, 30, 5)
)
worker = QueueWorker(queue, service, transport, records=store)
app.state.settings = cfg
app.state.whatsapp = transport
app.state.fudia = backend
app.state.inbound_queue = queue
app.state.conversation_store = store


@app.on_event("startup")
async def startup():
    app.state.worker_task = asyncio.create_task(worker.run())


@app.on_event("shutdown")
async def shutdown():
    await worker.stop()
    app.state.worker_task.cancel()
    await transport.close()
    await store.close()
    await queue.close()
