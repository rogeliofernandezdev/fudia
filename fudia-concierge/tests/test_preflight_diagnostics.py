"""Offline diagnostics tests: immediate async doubles need no sockets/event loop."""

import errno
import json
import socket
import ssl
from collections.abc import Coroutine
from pathlib import Path
from types import SimpleNamespace
from typing import Any
from unittest.mock import AsyncMock

import httpx
import pytest
from google.auth.exceptions import DefaultCredentialsError
from google.genai.errors import APIError
from redis.exceptions import AuthenticationError, ResponseError
from redis.exceptions import ConnectionError as RedisConnectionError

from config.settings import Settings
from src import preflight

PRIVATE = "private-password-token-phone-qr"


def complete_immediate(coroutine: Coroutine[Any, Any, Any]) -> Any:
    """Run only coroutines backed by immediate test doubles, never live I/O."""
    try:
        coroutine.send(None)
    except StopIteration as done:
        return done.value
    finally:
        coroutine.close()
    raise AssertionError("Test double unexpectedly suspended for I/O")


def records(capsys: pytest.CaptureFixture[str]) -> list[dict[str, Any]]:
    captured = capsys.readouterr()
    assert PRIVATE not in captured.out + captured.err
    assert "Traceback" not in captured.out + captured.err
    return [json.loads(line) for line in captured.out.splitlines()]


@pytest.mark.parametrize(
    "error,reason",
    [
        (AuthenticationError(PRIVATE), "redis_auth_failed"),
        (RedisConnectionError(PRIVATE), "redis_connection_failed"),
        (ResponseError(PRIVATE), "redis_command_rejected"),
        (FileNotFoundError(PRIVATE), "ca_missing"),
        (PermissionError(PRIVATE), "ca_unreadable"),
        (ssl.SSLError(PRIVATE), "tls_handshake_failed"),
        (socket.gaierror(PRIVATE), "dns_failed"),
        (TimeoutError(PRIVATE), "timeout"),
        (httpx.ReadTimeout(PRIVATE), "timeout"),
        (httpx.ConnectError(PRIVATE), "network_failed"),
        (OSError(errno.ECONNREFUSED, PRIVATE), "network_failed"),
        (DefaultCredentialsError(PRIVATE), "vertex_credentials_failed"),
        (ValueError(PRIVATE), "configuration_invalid"),
        (RuntimeError(PRIVATE), "unexpected_error"),
    ],
)
def test_error_classification_never_prints_provider_messages(
    error: Exception, reason: str, capsys: pytest.CaptureFixture[str]
) -> None:
    with pytest.raises(preflight.PreflightFailure) as failed, preflight.check_stage("redis"):
        raise error
    preflight.emit_check(failed.value.stage, "failed", failed.value)
    result = records(capsys)
    assert result[0]["status"] == "started"
    assert result[-1]["stage"] == "redis"
    assert result[-1]["reason"] == reason
    assert result[-1]["severity"] == "ERROR"
    assert "passed" not in [record["status"] for record in result]


def test_wrapped_tls_failure_keeps_safe_verification_code(capsys: pytest.CaptureFixture[str]) -> None:
    tls = ssl.SSLCertVerificationError(PRIVATE)
    tls.verify_code = 64
    wrapped = RedisConnectionError(f"rediss://:{PRIVATE}@private-redis")
    wrapped.__context__ = tls
    failure = preflight.classify_failure("redis", wrapped)
    preflight.emit_check(failure.stage, "failed", failure)
    result = records(capsys)[0]
    assert result["reason"] == "tls_certificate_invalid"
    assert result["tls_verify_code"] == 64
    assert "private-redis" not in json.dumps(result)
    tls.__context__ = wrapped  # Defensive: malformed exception chains must terminate.
    assert preflight.classify_failure("redis", wrapped).reason == "tls_certificate_invalid"


@pytest.mark.parametrize("status", [400, 401, 403, 404, 429, 503])
def test_vertex_errors_include_only_numeric_status(status: int, capsys: pytest.CaptureFixture[str]) -> None:
    error = APIError(status, {"error": {"message": PRIVATE, "details": [{"token": PRIVATE}]}})
    failure = preflight.classify_failure("vertex_flash", error)
    preflight.emit_check(failure.stage, "failed", failure)
    result = records(capsys)[0]
    assert result["reason"] == "vertex_request_failed"
    assert result["http_status"] == status
    assert result["stage"] == "vertex_flash"


def setup_checks(
    monkeypatch: pytest.MonkeyPatch,
) -> tuple[SimpleNamespace, list[httpx.Response], SimpleNamespace]:
    cfg = Settings(  # type: ignore[call-arg]  # BaseSettings accepts _env_file; generated signature omits it.
        _env_file=None,
        redis_url=f"rediss://:{PRIVATE}@private-redis:6378/0",
        redis_ca_certfile="mounted-ca.pem",
        fudia_concierge_api_key=PRIVATE,
        whatsapp_token=PRIVATE,
        whatsapp_phone_id="123456",
        whatsapp_graph_version="v26.0",
    )
    monkeypatch.setattr(preflight, "settings", cfg)
    monkeypatch.setattr(preflight.asyncio, "run", complete_immediate)
    monkeypatch.setattr(Path, "is_file", lambda path: True)
    monkeypatch.setattr(ssl, "create_default_context", lambda **kwargs: None)
    redis_client = SimpleNamespace(
        ping=AsyncMock(return_value=True),
        config_get=AsyncMock(return_value={"maxmemory-policy": "noeviction"}),
        info=AsyncMock(return_value={"cluster_enabled": 0}),
        aclose=AsyncMock(),
    )
    monkeypatch.setattr(preflight.redis, "from_url", lambda *args, **kwargs: redis_client)
    responses = [
        httpx.Response(200, json={"service": "foods-backend"}),
        httpx.Response(404, json={"code": "table_not_found"}),
        httpx.Response(200, json={"id": "123456"}),
    ]

    class HTTPClient:
        async def __aenter__(self) -> "HTTPClient":
            return self

        async def __aexit__(self, *args: object) -> None:
            pass

        async def get(self, *args: object, **kwargs: object) -> httpx.Response:
            return responses.pop(0)

    monkeypatch.setattr(preflight.httpx, "AsyncClient", lambda **kwargs: HTTPClient())
    vertex = SimpleNamespace(
        aio=SimpleNamespace(
            models=SimpleNamespace(
                generate_content=AsyncMock(return_value=SimpleNamespace(candidates=[object()]))
            ),
            aclose=AsyncMock(),
        )
    )
    monkeypatch.setattr(preflight, "build_vertex_client", lambda **kwargs: vertex)
    return redis_client, responses, vertex


def test_success_reports_each_gate_and_completes(
    monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    redis_client, responses, vertex = setup_checks(monkeypatch)
    preflight.main()
    result = records(capsys)
    assert [record["stage"] for record in result if record["status"] == "passed"] == [
        "configuration",
        "redis",
        "backend_health",
        "backend_integration",
        "meta_phone",
        "vertex_client",
        "vertex_lite",
        "vertex_flash",
        "complete",
    ]
    assert not responses
    redis_client.aclose.assert_awaited_once()
    vertex.aio.aclose.assert_awaited_once()


@pytest.mark.parametrize(
    "index,stage,status",
    [(0, "backend_health", 503), (1, "backend_integration", 401), (2, "meta_phone", 403)],
)
def test_http_failure_pinpoints_endpoint_and_stops_before_vertex(
    index: int, stage: str, status: int, monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    _, responses, vertex = setup_checks(monkeypatch)
    responses[index] = httpx.Response(status, json={"error": {"message": PRIVATE}})
    with pytest.raises(SystemExit) as exited:
        preflight.main()
    assert exited.value.code == 1
    failure = records(capsys)[-1]
    assert failure["stage"] == stage
    assert failure["reason"] == "http_status_unexpected"
    assert failure["http_status"] == status
    assert failure["hint"] == preflight.HTTP_HINTS[stage]
    vertex.aio.models.generate_content.assert_not_called()


def test_redis_failure_stops_http_and_models(
    monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    redis_client, responses, vertex = setup_checks(monkeypatch)
    redis_client.ping.side_effect = AuthenticationError(PRIVATE)
    with pytest.raises(SystemExit):
        preflight.main()
    result = records(capsys)
    assert result[-1]["stage"] == "redis"
    assert result[-1]["reason"] == "redis_auth_failed"
    assert len(responses) == 3
    vertex.aio.models.generate_content.assert_not_called()
    redis_client.aclose.assert_awaited_once()


@pytest.mark.parametrize("index,stage", [(0, "vertex_lite"), (1, "vertex_flash")])
def test_vertex_failure_reports_model_tier_without_provider_payload(
    index: int, stage: str, monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    _, _, vertex = setup_checks(monkeypatch)
    vertex.aio.models.generate_content.side_effect = [
        *[SimpleNamespace(candidates=[object()]) for _ in range(index)],
        APIError(403, {"error": {"message": PRIVATE}}),
    ]
    with pytest.raises(SystemExit):
        preflight.main()
    failure = records(capsys)[-1]
    assert failure["stage"] == stage
    assert failure["http_status"] == 403
    vertex.aio.aclose.assert_awaited_once()


def test_missing_ca_is_a_configuration_failure(
    monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    redis_client, _, _ = setup_checks(monkeypatch)
    monkeypatch.setattr(Path, "is_file", lambda path: False)
    with pytest.raises(SystemExit):
        preflight.main()
    failure = records(capsys)[-1]
    assert failure["stage"] == "configuration"
    assert failure["reason"] == "ca_missing"
    redis_client.ping.assert_not_called()


@pytest.mark.parametrize("invalid_json", [False, True])
def test_bad_http_content_never_logs_response_body(
    invalid_json: bool, monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    _, responses, vertex = setup_checks(monkeypatch)
    responses[0] = (
        httpx.Response(200, content=PRIVATE)
        if invalid_json
        else httpx.Response(200, json={"service": PRIVATE})
    )
    with pytest.raises(SystemExit):
        preflight.main()
    failure = records(capsys)[-1]
    assert failure["stage"] == "backend_health"
    assert failure["reason"] == "response_invalid"
    vertex.aio.models.generate_content.assert_not_called()


@pytest.mark.parametrize(
    "policy,cluster,reason",
    [("allkeys-lru", 0, "redis_policy_invalid"), ("noeviction", 1, "redis_cluster_enabled")],
)
def test_redis_safety_gates_remain_fail_closed(
    policy: str,
    cluster: int,
    reason: str,
    monkeypatch: pytest.MonkeyPatch,
    capsys: pytest.CaptureFixture[str],
) -> None:
    redis_client, responses, vertex = setup_checks(monkeypatch)
    redis_client.config_get.return_value = {"maxmemory-policy": policy}
    redis_client.info.return_value = {"cluster_enabled": cluster}
    with pytest.raises(SystemExit):
        preflight.main()
    failure = records(capsys)[-1]
    assert failure["stage"] == "redis"
    assert failure["reason"] == reason
    assert len(responses) == 3
    vertex.aio.models.generate_content.assert_not_called()
