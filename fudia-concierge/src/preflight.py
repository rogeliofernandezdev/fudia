"""Read-only release checks executed as a Cloud Run Job, never from CI's network.

Does not consume Streams, send WhatsApp messages, or create orders. Vertex
requests use a synthetic prompt and incur a small inference charge.
"""

from __future__ import annotations

import asyncio
from pathlib import Path

import httpx
import redis.asyncio as redis
from google.genai import types

from config.settings import Settings, settings
from src.infrastructure.vertex_adapter import build_vertex_client


async def check_redis(cfg: Settings) -> None:
    if not cfg.redis_ca_certfile or not cfg.redis_url.startswith("rediss://"):
        raise RuntimeError("Preflight requires Redis TLS and its mounted CA")
    client = redis.from_url(cfg.redis_connection_url, decode_responses=True)
    try:
        if not await client.ping():
            raise RuntimeError("Redis PING failed")
        policy = await client.config_get("maxmemory-policy")
        cluster = await client.info("cluster")
        if policy.get("maxmemory-policy") != "noeviction":
            raise RuntimeError("Redis requires maxmemory-policy=noeviction")
        if str(cluster.get("cluster_enabled")) != "0":
            raise RuntimeError("Concierge requires non-cluster Redis")
    finally:
        await client.aclose()


async def check_http(cfg: Settings) -> None:
    async with httpx.AsyncClient(timeout=15, follow_redirects=False) as client:
        health = await client.get(f"{cfg.fudia_api_url}/health")
        if health.status_code != 200 or health.json().get("service") != "foods-backend":
            raise RuntimeError("Backend health check failed")
        # An intentionally nonexistent QR tests authentication without business effects.
        response = await client.get(
            f"{cfg.fudia_api_url}/v1/integrations/concierge/preflight-invalid-qr/menu",
            headers={"X-Fudia-Concierge-Key": cfg.fudia_concierge_api_key},
        )
        if response.status_code != 404 or response.json().get("code") != "table_not_found":
            raise RuntimeError("Backend integration key check failed")
        meta = await client.get(
            f"https://graph.facebook.com/{cfg.whatsapp_graph_version}/{cfg.whatsapp_phone_id}",
            params={"fields": "id,display_phone_number"},
            headers={"Authorization": f"Bearer {cfg.whatsapp_token}"},
        )
        if meta.status_code != 200 or str(meta.json().get("id")) != cfg.whatsapp_phone_id:
            raise RuntimeError("Meta phone/token/version check failed")


async def check_vertex(cfg: Settings) -> None:
    client = build_vertex_client(
        project=cfg.google_cloud_project,
        location=cfg.google_cloud_location,
        timeout_seconds=cfg.llm_timeout_seconds,
        max_retries=cfg.llm_max_retries,
    )
    try:
        for model in dict.fromkeys((cfg.gemini_model_lite, cfg.gemini_model_flash)):
            response = await client.aio.models.generate_content(
                model=model,
                contents="Responde únicamente OK. Prueba técnica sin datos de clientes.",
                config=types.GenerateContentConfig(
                    max_output_tokens=128,
                    automatic_function_calling=types.AutomaticFunctionCallingConfig(disable=True),
                ),
            )
            if not response.candidates:
                raise RuntimeError("Vertex returned no candidates")
    finally:
        await client.aio.aclose()


async def run(cfg: Settings) -> None:
    # No .env dependency, private key file or conversation data is required.
    if cfg.llm_provider != "vertex" or not Path(cfg.redis_ca_certfile).is_file():
        raise RuntimeError("Invalid GCP preflight configuration")
    await check_redis(cfg)
    print("Preflight: Redis AUTH/TLS, noeviction and non-cluster OK.")
    await check_http(cfg)
    print("Preflight: backend integration and Meta phone access OK.")
    await check_vertex(cfg)


def main() -> None:
    try:
        asyncio.run(run(settings))
    except Exception:
        # Provider exception strings may contain URLs, identifiers or credentials.
        print("Concierge preflight failed: check Redis/TLS, backend key, Meta and Vertex permissions.")
        raise SystemExit(1) from None
    print("Concierge preflight passed: Redis/TLS, backend integration, Meta and both Vertex models.")


if __name__ == "__main__":
    main()
