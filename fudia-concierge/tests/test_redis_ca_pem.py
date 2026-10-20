"""Validate worker TLS configuration entirely offline, without certificate files."""

import ssl
from pathlib import Path
from typing import Any
from urllib.parse import parse_qs, urlsplit

import certifi
import pytest
import redis.asyncio as redis
from redis.asyncio.connection import RedisSSLContext

from config.settings import Settings


def pem_settings(**overrides: Any) -> Settings:
    options: dict[str, Any] = {
        "_env_file": None,
        "redis_url": "rediss://:a%2Fb%40c@10.0.0.1:6378/0",
        "redis_ca_certfile": "",
        "redis_ca_cert_pem": "private-ca-test-value",
    }
    return Settings(**(options | overrides))


@pytest.fixture
def public_ca_pem() -> str:
    # A public CA from the pinned local trust bundle, not a deployed secret.
    bundle = Path(certifi.where()).read_text(encoding="utf-8")
    begin = bundle.index("-----BEGIN CERTIFICATE-----")
    end = bundle.index("-----END CERTIFICATE-----", begin) + len("-----END CERTIFICATE-----")
    return bundle[begin:end] + "\n"


def test_worker_pem_is_loaded_into_real_tls_context_without_files(public_ca_pem: str) -> None:
    cfg = pem_settings(redis_ca_cert_pem=public_ca_pem)
    url = cfg.redis_connection_url
    assert urlsplit(url).netloc == urlsplit(cfg.redis_url).netloc
    options = parse_qs(urlsplit(url).query)
    assert options["ssl_ca_data"] == [public_ca_pem]
    assert "ssl_ca_certs" not in options
    assert options["ssl_cert_reqs"] == ["required"]
    assert options["ssl_check_hostname"] == ["True"]

    # Use the installed driver's URL parser and SSL context implementation, not
    # mocks: this catches unsupported kwargs, newline or URL encoding problems.
    client = redis.from_url(url)
    kwargs = client.connection_pool.connection_kwargs
    assert kwargs["password"] == "a/b@c"
    assert kwargs["ssl_ca_data"] == public_ca_pem
    context = RedisSSLContext(
        ca_data=kwargs["ssl_ca_data"],
        cert_reqs=kwargs["ssl_cert_reqs"],
        check_hostname=kwargs["ssl_check_hostname"],
    ).get()
    assert context.verify_mode == ssl.CERT_REQUIRED
    assert context.check_hostname is True
    assert context.get_ca_certs()


def test_pem_never_appears_in_settings_repr_or_serialization() -> None:
    cfg = pem_settings()
    assert "private-ca-test-value" not in repr(cfg)
    assert "redis_ca_cert_pem" not in cfg.model_dump()
    assert "private-ca-test-value" not in cfg.model_dump_json()


@pytest.mark.parametrize(
    "uri",
    [
        "redis://10.0.0.1:6379/0",
        "rediss://10.0.0.1:6378/0?ssl_cert_reqs=none",
        "rediss://10.0.0.1:6378/0?ssl_check_hostname=false",
        "rediss://10.0.0.1:6378/0?ssl_ca_data=other-ca",
        "rediss://10.0.0.1:6378/0?ssl_ca_certs=other-ca.pem",
    ],
)
def test_pem_keeps_plaintext_and_tls_overrides_rejected(uri: str) -> None:
    with pytest.raises(ValueError):
        _ = pem_settings(redis_url=uri).redis_connection_url


def test_pem_and_file_are_mutually_exclusive() -> None:
    with pytest.raises(ValueError, match="only one"):
        _ = pem_settings(redis_ca_certfile="mounted-ca.pem").redis_connection_url


def test_invalid_pem_fails_without_exposing_its_content() -> None:
    with pytest.raises((ssl.SSLError, ValueError)) as error:
        _ = pem_settings().redis_connection_url
    assert "private-ca-test-value" not in str(error.value)


def test_pem_can_be_injected_as_environment_secret(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("REDIS_CA_CERT_PEM", "private-ca-test-value")
    options: dict[str, Any] = {"_env_file": None, "redis_ca_certfile": ""}
    cfg = Settings(**options)
    assert cfg.redis_ca_cert_pem.get_secret_value() == "private-ca-test-value"
    assert "private-ca-test-value" not in repr(cfg)
