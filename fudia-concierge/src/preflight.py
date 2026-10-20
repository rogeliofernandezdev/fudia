"""Read-only release checks executed as a Cloud Run Job, never from CI's network.

Does not consume Streams, send WhatsApp messages, or create orders. Vertex
requests use a synthetic prompt and incur a small inference charge.
"""

from __future__ import annotations

import asyncio
import errno
import json
import socket
import ssl
from collections.abc import Iterator
from contextlib import contextmanager
from pathlib import Path

import httpx
import redis.asyncio as redis
from google.auth.exceptions import DefaultCredentialsError, RefreshError
from google.genai import types
from google.genai.errors import APIError
from redis.exceptions import AuthenticationError, ResponseError
from redis.exceptions import ConnectionError as RedisConnectionError
from redis.exceptions import TimeoutError as RedisTimeoutError

from config.settings import Settings, settings
from src.infrastructure.vertex_adapter import build_vertex_client

HINTS = {
    "configuration_invalid": "Revisa proveedor Vertex, TLS de Redis y montaje del certificado CA.",
    "ca_missing": "Revisa la ruta y el montaje del secreto que contiene el certificado CA.",
    "ca_unreadable": "Revisa los permisos de lectura del archivo CA montado.",
    "tls_certificate_invalid": "Revisa CA, vigencia e identidad del servidor; no desactives TLS.",
    "tls_handshake_failed": "Revisa el puerto TLS y el certificado CA de Redis.",
    "redis_auth_failed": "Revisa AUTH y la codificación de la contraseña en REDIS_URL.",
    "redis_connection_failed": "Revisa IP, puerto, VPC, subred y acceso privado a Redis.",
    "redis_command_rejected": "Revisa que Redis permita CONFIG GET e INFO, sin cambiar datos.",
    "redis_ping_failed": "Redis no confirmó PING.",
    "redis_policy_invalid": "Configura maxmemory-policy=noeviction en Redis.",
    "redis_cluster_enabled": "Este despliegue requiere Redis sin cluster.",
    "timeout": "La dependencia no respondió a tiempo; revisa red y disponibilidad.",
    "dns_failed": "No se pudo resolver el nombre de la dependencia; revisa red y DNS.",
    "network_failed": "Revisa conectividad y disponibilidad de la dependencia.",
    "http_status_unexpected": "Revisa el estado HTTP junto con la etapa que falló.",
    "response_invalid": "La respuesta no tiene el formato o contenido esperado para esta etapa.",
    "vertex_credentials_failed": "Revisa ADC y la identidad runtime del worker.",
    "vertex_request_failed": "Revisa acceso al modelo, proyecto, región y cuotas de Vertex.",
    "vertex_empty_response": "Vertex respondió sin candidatos; revisa el acceso al modelo.",
    "unexpected_error": "La etapa falló por una causa no clasificada; no se registró el mensaje privado.",
}
HTTP_HINTS = {
    "backend_health": "Revisa disponibilidad y acceso HTTP al backend GCP.",
    "backend_integration": "Revisa que backend y Concierge usen la misma clave y versión del secreto.",
    "meta_phone": "Revisa WHATSAPP_TOKEN, permisos sobre WHATSAPP_PHONE_ID y WHATSAPP_GRAPH_VERSION.",
}


class PreflightFailure(RuntimeError):
    """Only allowlisted diagnostics, never provider messages, URLs or payloads."""

    def __init__(
        self, stage: str, reason: str, *, http_status: int | None = None, tls_verify_code: int | None = None
    ) -> None:
        super().__init__("Preflight check failed")
        self.stage = stage
        self.reason = reason
        self.http_status = http_status
        self.tls_verify_code = tls_verify_code


def classify_failure(stage: str, error: Exception) -> PreflightFailure:
    # Redis/http clients can wrap the actual TLS/DNS failure. Inspect types only:
    # str(error), repr(error), response bodies and traceback text may leak secrets.
    chain: list[BaseException] = []
    current: BaseException | None = error
    while current is not None and all(current is not item for item in chain):
        chain.append(current)
        current = current.__cause__ or current.__context__
    for cause in chain:
        if isinstance(cause, ssl.SSLCertVerificationError):
            code = getattr(cause, "verify_code", None)
            return PreflightFailure(
                stage, "tls_certificate_invalid", tls_verify_code=code if type(code) is int else None
            )
    cases = (
        (ssl.SSLError, "tls_handshake_failed"),
        (AuthenticationError, "redis_auth_failed"),
        (FileNotFoundError, "ca_missing"),
        (PermissionError, "ca_unreadable"),
        (socket.gaierror, "dns_failed"),
        ((TimeoutError, RedisTimeoutError, httpx.TimeoutException), "timeout"),
        ((DefaultCredentialsError, RefreshError), "vertex_credentials_failed"),
        (RedisConnectionError, "redis_connection_failed"),
        (ResponseError, "redis_command_rejected"),
        (httpx.HTTPStatusError, "http_status_unexpected"),
        (httpx.RequestError, "network_failed"),
        (json.JSONDecodeError, "response_invalid"),
        (ValueError, "configuration_invalid"),
    )
    for types_, reason in cases:
        if any(isinstance(cause, types_) for cause in chain):
            status = error.response.status_code if isinstance(error, httpx.HTTPStatusError) else None
            return PreflightFailure(stage, reason, http_status=status)
    for cause in chain:
        if isinstance(cause, APIError):
            code = cause.code
            return PreflightFailure(
                stage,
                "vertex_request_failed",
                http_status=code if type(code) is int and 100 <= code <= 599 else None,
            )
        if isinstance(cause, OSError) and cause.errno in (
            errno.ECONNREFUSED,
            errno.ENETUNREACH,
            errno.EHOSTUNREACH,
        ):
            return PreflightFailure(stage, "network_failed")
    return PreflightFailure(stage, "unexpected_error")


def emit_check(stage: str, status: str, failure: PreflightFailure | None = None) -> None:
    record: dict[str, str | int] = {
        "event": "concierge.preflight",
        "stage": stage,
        "status": status,
        "severity": "ERROR" if failure else "INFO",
    }
    if failure:
        hint = HINTS[failure.reason]
        if failure.reason == "http_status_unexpected":
            hint = HTTP_HINTS.get(stage, hint)
        record.update(reason=failure.reason, hint=hint)
        if failure.http_status is not None:
            record["http_status"] = failure.http_status
        if failure.tls_verify_code is not None:
            record["tls_verify_code"] = failure.tls_verify_code
    print(json.dumps(record, ensure_ascii=False), flush=True)


@contextmanager
def check_stage(stage: str) -> Iterator[None]:
    emit_check(stage, "started")
    try:
        yield
    except PreflightFailure:
        raise
    except Exception as error:
        raise classify_failure(stage, error) from None
    emit_check(stage, "passed")


async def check_redis(cfg: Settings) -> None:
    with check_stage("redis"):
        if not cfg.redis_ca_certfile or not cfg.redis_url.startswith("rediss://"):
            raise PreflightFailure("redis", "configuration_invalid")
        client = redis.from_url(cfg.redis_connection_url, decode_responses=True)
        try:
            if not await client.ping():
                raise PreflightFailure("redis", "redis_ping_failed")
            policy = await client.config_get("maxmemory-policy")
            cluster = await client.info("cluster")
            if policy.get("maxmemory-policy") != "noeviction":
                raise PreflightFailure("redis", "redis_policy_invalid")
            if str(cluster.get("cluster_enabled")) != "0":
                raise PreflightFailure("redis", "redis_cluster_enabled")
        finally:
            await client.aclose()


async def check_http(cfg: Settings) -> None:
    async with httpx.AsyncClient(timeout=15, follow_redirects=False) as client:
        with check_stage("backend_health"):
            health = await client.get(f"{cfg.fudia_api_url}/health")
            if health.status_code != 200:
                raise PreflightFailure(
                    "backend_health", "http_status_unexpected", http_status=health.status_code
                )
            if health.json().get("service") != "foods-backend":
                raise PreflightFailure("backend_health", "response_invalid")
        with check_stage("backend_integration"):
            # An intentionally nonexistent QR tests authentication without business effects.
            response = await client.get(
                f"{cfg.fudia_api_url}/v1/integrations/concierge/preflight-invalid-qr/menu",
                headers={"X-Fudia-Concierge-Key": cfg.fudia_concierge_api_key},
            )
            if response.status_code != 404:
                raise PreflightFailure(
                    "backend_integration", "http_status_unexpected", http_status=response.status_code
                )
            if response.json().get("code") != "table_not_found":
                raise PreflightFailure("backend_integration", "response_invalid")
        with check_stage("meta_phone"):
            meta = await client.get(
                f"https://graph.facebook.com/{cfg.whatsapp_graph_version}/{cfg.whatsapp_phone_id}",
                params={"fields": "id,display_phone_number"},
                headers={"Authorization": f"Bearer {cfg.whatsapp_token}"},
            )
            if meta.status_code != 200:
                raise PreflightFailure("meta_phone", "http_status_unexpected", http_status=meta.status_code)
            if str(meta.json().get("id")) != cfg.whatsapp_phone_id:
                raise PreflightFailure("meta_phone", "response_invalid")


async def check_vertex(cfg: Settings) -> None:
    with check_stage("vertex_client"):
        client = build_vertex_client(
            project=cfg.google_cloud_project,
            location=cfg.google_cloud_location,
            timeout_seconds=cfg.llm_timeout_seconds,
            max_retries=cfg.llm_max_retries,
        )
    try:
        for index, model in enumerate(dict.fromkeys((cfg.gemini_model_lite, cfg.gemini_model_flash))):
            stage = "vertex_lite" if index == 0 else "vertex_flash"
            with check_stage(stage):
                response = await client.aio.models.generate_content(
                    model=model,
                    contents="Responde únicamente OK. Prueba técnica sin datos de clientes.",
                    config=types.GenerateContentConfig(
                        max_output_tokens=128,
                        automatic_function_calling=types.AutomaticFunctionCallingConfig(disable=True),
                    ),
                )
                if not response.candidates:
                    raise PreflightFailure(stage, "vertex_empty_response")
    finally:
        await client.aio.aclose()


async def run(cfg: Settings) -> None:
    # No .env dependency, private key file or conversation data is required.
    with check_stage("configuration"):
        if (
            cfg.llm_provider != "vertex"
            or not cfg.redis_ca_certfile
            or not cfg.redis_url.startswith("rediss://")
        ):
            raise PreflightFailure("configuration", "configuration_invalid")
        if not Path(cfg.redis_ca_certfile).is_file():
            raise PreflightFailure("configuration", "ca_missing")
    await check_redis(cfg)
    await check_http(cfg)
    await check_vertex(cfg)


def main() -> None:
    try:
        asyncio.run(run(settings))
    except Exception as error:
        failure = error if isinstance(error, PreflightFailure) else classify_failure("startup", error)
        emit_check(failure.stage, "failed", failure)
        raise SystemExit(1) from None
    emit_check("complete", "passed")


if __name__ == "__main__":
    main()
