"""Dedicated Redis Stream worker runtime.

Run separately from the HTTP ingress to make scaling policies independent.
Requires an always-running process with CPU available even when idle.
"""
from __future__ import annotations

import asyncio

from config.settings import settings
from src.main import create_app


async def run_worker() -> None:
    cfg = settings.model_copy(update={"worker_enabled": True})
    app = create_app(cfg)
    async with app.router.lifespan_context(app):
        task = app.state.queue_worker_task
        if task is None:
            raise RuntimeError("The queue worker was not started")
        await task


def main() -> None:
    asyncio.run(run_worker())


if __name__ == "__main__":
    main()
