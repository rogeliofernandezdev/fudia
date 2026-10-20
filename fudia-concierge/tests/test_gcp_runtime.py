import asyncio
import ssl
from contextlib import asynccontextmanager
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import AsyncMock
from urllib.parse import parse_qs, urlsplit

import httpx
import pytest
import redis.asyncio as redis
from redis.exceptions import ResponseError

from config.settings import Settings
from src import preflight


def test_configuration_errors_do_not_expose_secret_values() -> None:
    with pytest.raises(ValueError) as error:
        Settings(_env_file=None, env="production", fudia_concierge_api_key="private-test-service-key")
    assert "private-test-service-key" not in str(error.value)


def test_private_ca_uses_verified_tls_and_preserves_credentials(monkeypatch: pytest.MonkeyPatch) -> None:
    calls: list[str] = []
    monkeypatch.setattr(ssl, "create_default_context", lambda *, cafile: calls.append(cafile))
    cfg = Settings(
        _env_file=None, redis_url="rediss://:a%2Fb%40c@10.0.0.1:6378/0", redis_ca_certfile="ca.pem"
    )
    url = cfg.redis_connection_url
    assert calls == ["ca.pem"]
    assert urlsplit(url).netloc == urlsplit(cfg.redis_url).netloc
    options = parse_qs(urlsplit(url).query)
    assert options["ssl_cert_reqs"] == ["required"]
    assert options["ssl_check_hostname"] == ["True"]
    client = redis.from_url(url)
    kwargs = client.connection_pool.connection_kwargs
    assert kwargs["password"] == "a/b@c"
    assert kwargs["ssl_ca_certs"] == "ca.pem"
    assert kwargs["ssl_check_hostname"] is True


def test_private_ca_rejects_plaintext_and_uri_overrides() -> None:
    for uri in ("redis://10.0.0.1:6379/0", "rediss://10.0.0.1:6378/0?ssl_cert_reqs=none"):
        cfg = Settings(_env_file=None, redis_url=uri, redis_ca_certfile="ca.pem")
        with pytest.raises(ValueError):
            _ = cfg.redis_connection_url
    with pytest.raises((FileNotFoundError, ssl.SSLError)):
        _ = Settings(
            _env_file=None, redis_url="rediss://10.0.0.1:6378", redis_ca_certfile="missing-ca.pem"
        ).redis_connection_url
    cfg = Settings(_env_file=None)
    assert cfg.redis_connection_url == cfg.redis_url  # Local Redis remains supported.


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "policy,cluster,ok", [("noeviction", 0, True), ("allkeys-lru", 0, False), ("noeviction", 1, False)]
)
async def test_preflight_checks_policy_and_cluster_without_queue_mutations(
    monkeypatch: pytest.MonkeyPatch,
    policy: str,
    cluster: int,
    ok: bool,
) -> None:
    client = SimpleNamespace(
        ping=AsyncMock(return_value=True),
        config_get=AsyncMock(side_effect=ResponseError("CONFIG is blocked by Memorystore")),
        info=AsyncMock(
            side_effect=lambda section: {
                "memory": {"maxmemory_policy": policy},
                "cluster": {"cluster_enabled": cluster},
            }[section]
        ),
        aclose=AsyncMock(),
    )
    monkeypatch.setattr(ssl, "create_default_context", lambda **kwargs: None)
    monkeypatch.setattr(preflight.redis, "from_url", lambda *args, **kwargs: client)
    cfg = Settings(_env_file=None, redis_url="rediss://:password@10.0.0.1:6378", redis_ca_certfile="ca.pem")
    if ok:
        await preflight.check_redis(cfg)
    else:
        with pytest.raises(RuntimeError):
            await preflight.check_redis(cfg)
    client.aclose.assert_awaited_once()
    client.config_get.assert_not_called()
    assert [call.args for call in client.info.await_args_list] == [("memory",), ("cluster",)]


@pytest.mark.asyncio
async def test_preflight_http_is_read_only_and_checks_integration_and_meta(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    requests: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        requests.append(request)
        if request.url.path == "/health":
            return httpx.Response(200, json={"service": "foods-backend"})
        if "integrations" in request.url.path:
            assert request.headers["X-Fudia-Concierge-Key"] == "test-key"
            return httpx.Response(404, json={"code": "table_not_found"})
        assert request.headers["Authorization"] == "Bearer test-token"
        assert "access_token" not in request.url.params
        return httpx.Response(200, json={"id": "123456"})

    factory = httpx.AsyncClient
    monkeypatch.setattr(
        preflight.httpx,
        "AsyncClient",
        lambda **kwargs: factory(transport=httpx.MockTransport(handler), **kwargs),
    )
    await preflight.check_http(
        Settings(
            _env_file=None,
            fudia_api_url="https://api.example.com",
            fudia_concierge_api_key="test-key",
            whatsapp_token="test-token",
            whatsapp_graph_version="v26.0",
            whatsapp_phone_id="123456",
        )
    )
    assert len(requests) == 3
    assert all(request.method == "GET" for request in requests)


@pytest.mark.asyncio
async def test_preflight_invokes_both_models_without_tools(monkeypatch: pytest.MonkeyPatch) -> None:
    generate = AsyncMock(return_value=SimpleNamespace(candidates=[object()]))
    close = AsyncMock()
    client = SimpleNamespace(
        aio=SimpleNamespace(models=SimpleNamespace(generate_content=generate), aclose=close)
    )
    monkeypatch.setattr(preflight, "build_vertex_client", lambda **kwargs: client)
    cfg = Settings(_env_file=None)
    await preflight.check_vertex(cfg)
    assert [call.kwargs["model"] for call in generate.await_args_list] == [
        cfg.gemini_model_lite,
        cfg.gemini_model_flash,
    ]
    assert all(not call.kwargs["config"].tools for call in generate.await_args_list)
    close.assert_awaited_once()


@pytest.mark.asyncio
async def test_worker_shutdown_drains_on_signal_event(monkeypatch: pytest.MonkeyPatch) -> None:
    from src import worker

    stop = asyncio.Event()
    drained: list[bool] = []
    task: asyncio.Task[None] | None = None

    @asynccontextmanager
    async def lifespan(app):
        nonlocal task
        task = asyncio.create_task(asyncio.sleep(60))
        app.state.queue_worker_task = task
        try:
            yield
        finally:
            drained.append(True)
            task.cancel()
            await asyncio.gather(task, return_exceptions=True)

    app = SimpleNamespace(state=SimpleNamespace(), router=SimpleNamespace(lifespan_context=lifespan))
    monkeypatch.setattr(worker, "create_app", lambda cfg: app)
    run = asyncio.create_task(worker.run_worker(stop))
    while task is None:
        await asyncio.sleep(0)
    stop.set()
    await asyncio.wait_for(run, 1)
    assert drained == [True]


def test_preflight_does_not_print_provider_error_credentials(
    monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    def failing(coroutine) -> None:
        coroutine.close()
        raise RuntimeError("rediss://secret-password@redis")

    monkeypatch.setattr(preflight.asyncio, "run", failing)
    with pytest.raises(SystemExit):
        preflight.main()
    assert "secret-password" not in capsys.readouterr().out


def test_all_runtime_redis_clients_and_dlq_use_same_tls_configuration() -> None:
    main = (Path(__file__).parents[1] / "src/application.py").read_text()
    dlq = (Path(__file__).parents[1] / "scripts/dlq.py").read_text()
    assert "redis_url = cfg.redis_connection_url" in main
    assert "cfg.redis_url," not in main
    assert "settings.redis_url" not in dlq
