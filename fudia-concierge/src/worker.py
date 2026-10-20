"""Dedicated Redis Stream worker runtime.

Run separately from the HTTP ingress to make scaling policies independent.
Requires an always-running process with CPU available even when idle.
"""

from __future__ import annotations

import asyncio
import contextlib
import signal

from config.settings import settings
from src.application import create_app


async def run_worker(stop_event: asyncio.Event | None = None) -> None:
    cfg = settings.model_copy(update={"worker_enabled": True})
    app = create_app(cfg)
    stop = stop_event or asyncio.Event()
    loop = asyncio.get_running_loop()
    registered: list[signal.Signals] = []
    if stop_event is None:
        for signum in (signal.SIGTERM, signal.SIGINT):
            try:
                loop.add_signal_handler(signum, stop.set)
                registered.append(signum)
            except NotImplementedError:
                # Windows local runs retain asyncio's default Ctrl+C handling.
                pass
    try:
        async with app.router.lifespan_context(app):
            task = app.state.queue_worker_task
            if task is None:
                raise RuntimeError("The queue worker was not started")
            waiter = asyncio.create_task(stop.wait())
            try:
                done, _ = await asyncio.wait({task, waiter}, return_when=asyncio.FIRST_COMPLETED)
                if task in done:
                    await task  # Propagate worker failures: Cloud Run must restart it.
                    raise RuntimeError("The queue worker stopped unexpectedly")
                # Exiting lifespan invokes stop + bounded drain before closing clients.
            finally:
                waiter.cancel()
                with contextlib.suppress(asyncio.CancelledError):
                    await waiter
    finally:
        for signum in registered:
            loop.remove_signal_handler(signum)


def main() -> None:
    asyncio.run(run_worker())


if __name__ == "__main__":
    main()
