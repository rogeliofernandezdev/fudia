import json
from pathlib import Path

import pytest
from prometheus_client import generate_latest
from pydantic import ValidationError

from config.settings import Settings
from src.domain.models import ConversationSession
from src.metrics import INBOUND_MESSAGES
from src.services.tool_registry import ToolRegistry

ROOT = Path(__file__).resolve().parents[1]


class NoopFudia:
    pass


@pytest.mark.asyncio
async def test_specialists_cannot_call_foreign_tools() -> None:
    registry = ToolRegistry(
        NoopFudia(),  # type: ignore[arg-type]
        ConversationSession(
            phone="51999999999",
            qr_token="a" * 32,
        ),
        "test",
    )

    menu_denied = await registry.execute_for(
        "menu",
        "confirm_order",
        {},
    )
    order_denied = await registry.execute_for(
        "order",
        "request_bill",
        {},
    )
    service_denied = await registry.execute_for(
        "service",
        "add_item",
        {},
    )

    assert (
        menu_denied["code"]
        == "tool_not_allowed_for_agent"
    )
    assert (
        order_denied["code"]
        == "tool_not_allowed_for_agent"
    )
    assert (
        service_denied["code"]
        == "tool_not_allowed_for_agent"
    )


def test_eval_corpus_has_required_coverage() -> None:
    scope = json.loads(
        (
            ROOT
            / "evals"
            / "scope_cases.json"
        ).read_text(encoding="utf-8")
    )
    intent = json.loads(
        (
            ROOT
            / "evals"
            / "intent_cases.json"
        ).read_text(encoding="utf-8")
    )
    injection = json.loads(
        (
            ROOT
            / "evals"
            / "prompt_injection_cases.json"
        ).read_text(encoding="utf-8")
    )

    assert len(scope) >= 10
    assert {
        case["expected"]
        for case in scope
    } == {"IN_SCOPE", "OUT_OF_SCOPE"}
    assert {
        case["expected"]
        for case in intent
    } == {"menu", "order", "service"}
    assert len(injection) >= 5
    assert any(
        case["expected"] == "IN_SCOPE"
        for case in injection
    )
    assert any(
        case["expected"] == "OUT_OF_SCOPE"
        for case in injection
    )


def test_production_settings_fail_fast_without_secrets() -> None:
    with pytest.raises(
        ValidationError,
        match="Missing production settings",
    ):
        Settings(
            _env_file=None,
            env="production",
            fudia_concierge_api_key="",
            openai_api_key="",
            whatsapp_token="",
            whatsapp_verify_token="",
            whatsapp_app_secret="",
            whatsapp_graph_version="",
        )


def test_prometheus_registry_contains_concierge_metrics() -> None:
    INBOUND_MESSAGES.labels(
        result="test"
    ).inc()
    payload = generate_latest().decode("utf-8")
    assert (
        "fudia_concierge_inbound_messages_total"
        in payload
    )
    assert (
        "fudia_concierge_queue_deliveries_total"
        in payload
    )
    assert (
        "fudia_concierge_queue_in_flight"
        in payload
    )


def test_whatsapp_number_is_resolved_from_meta_not_app_config() -> None:
    settings_source = (
        ROOT / "config" / "settings.py"
    ).read_text(encoding="utf-8")
    env_example = (
        ROOT / ".env.example"
    ).read_text(encoding="utf-8")
    adapter = (
        ROOT
        / "src"
        / "infrastructure"
        / "whatsapp_adapter.py"
    ).read_text(encoding="utf-8")

    assert "FUDIA_WHATSAPP_PHONE" not in settings_source
    assert "FUDIA_WHATSAPP_PHONE" not in env_example
    assert "WHATSAPP_PHONE_ID" in env_example
    assert "display_phone_number" in adapter
