"""Reproducible Redis Streams throughput baseline; no Meta, LLM or Fudia API calls."""
from __future__ import annotations

import asyncio
import json
import os
import time
from pathlib import Path
from uuid import uuid4

import pytest

from src.domain.models import InboundMessage
from src.infrastructure.queue import RedisInboundQueue
from src.services.queue_worker import QueueWorker


class FastFakeService:
    async def handle_message(self, phone: str, text: str, channel_id: str = "") -> str:
        return "respuesta de prueba"


class FastFakeSender:
    def __init__(self) -> None:
        self.sent = 0

    async def send_text(self, phone: str, message: str) -> None:
        self.sent += 1


@pytest.mark.asyncio
async def test_redis_stream_workers_under_concurrent_load() -> None:
    url = os.getenv("REDIS_URL", "")
    if os.getenv("FUDIA_REDIS_LOAD_TEST") != "1" or not url:
        pytest.skip("set FUDIA_REDIS_LOAD_TEST=1 and REDIS_URL for Redis load")

    rows: list[dict[str, float | int]] = []
    for workers in (1, 8, 32):
        stream = f"fudia:concierge:capacity:{uuid4().hex}"
        queue = RedisInboundQueue(
            url, stream=stream, group="capacity-test",
            max_attempts=2, visibility_timeout_ms=30000,
        )
        count = 256
        try:
            await queue.start()
            for index in range(count):
                ok = await queue.enqueue(InboundMessage(
                    message_id=f"{stream}:{index}",
                    phone=f"519{index:09d}",
                    text="hola",
                    sender_phone_id="meta-capacity",
                ))
                assert ok
            sender = FastFakeSender()
            worker = QueueWorker(
                queue, FastFakeService(), sender,
                max_concurrency=workers, visibility_heartbeat_seconds=1.0,
            )
            started = time.perf_counter()
            task = asyncio.create_task(worker.run())
            try:
                async def completed(
                    active_sender: FastFakeSender = sender,
                    expected: int = count,
                ) -> None:
                    while active_sender.sent < expected:
                        await asyncio.sleep(0.01)

                await asyncio.wait_for(completed(), timeout=30)
            finally:
                await worker.stop()
                await asyncio.wait_for(task, timeout=5)
            elapsed = time.perf_counter() - started
            pending = await queue.client.xpending(stream, queue.group)
            assert sender.sent == count
            assert pending["pending"] == 0
            rps = round(count / elapsed, 1)
            rows.append({
                "workers": workers, "messages": count, "responses": sender.sent,
                "errors": 0, "elapsed_seconds": round(elapsed, 3),
                "messages_per_second": rps,
            })
            print(f"CONCIERGE_CAPACITY workers={workers} messages={count} "
                  f"errors=0 throughput={rps}msg/s elapsed={elapsed:.3f}s")
        finally:
            await queue.client.delete(stream, stream + ":dead")
            await queue.close()

    Path("concierge-capacity.json").write_text(
        json.dumps({
            "scenario": "Real Redis Streams + fake LLM, Meta and Fudia backend",
            "environment": "GitHub Actions runner, Redis 7",
            "results": rows,
        }, indent=2), encoding="utf-8",
    )
