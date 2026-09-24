from pathlib import Path
from typing import Any

import pytest

from src.domain.models import ChatMessage, ConversationSession
from src.infrastructure.openai_adapter import OpenAIConciergeEngine


class FakeScopeResponse:
    def __init__(self, output_text: str) -> None:
        self.output_text = output_text


class FakeResponses:
    def __init__(self, output_text: str) -> None:
        self.output_text = output_text
        self.calls: list[dict[str, Any]] = []

    async def create(self, **kwargs: Any) -> FakeScopeResponse:
        self.calls.append(kwargs)
        return FakeScopeResponse(self.output_text)


class FakeOpenAI:
    def __init__(self, output_text: str) -> None:
        self.responses = FakeResponses(output_text)


def build_engine(tmp_path: Path, output_text: str) -> tuple[OpenAIConciergeEngine, FakeOpenAI]:
    prompt = tmp_path / "system.md"
    prompt.write_text("restaurant ordering only", encoding="utf-8")
    client = FakeOpenAI(output_text)
    engine = OpenAIConciergeEngine(
        api_key="test",
        model="test-model",
        prompt_path=prompt,
        client=client,  # type: ignore[arg-type]
    )
    return engine, client


@pytest.mark.asyncio
async def test_scope_classifier_rejects_out_of_scope_label(tmp_path: Path) -> None:
    engine, client = build_engine(tmp_path, "OUT_OF_SCOPE")
    session = ConversationSession(
        phone="51999999999",
        qr_token="a" * 32,
        messages=[
            ChatMessage(role="assistant", content="¿Qué deseas pedir?"),
        ],
    )

    allowed = await engine.is_in_scope(session, "¿Quién fue Napoleón?")

    assert allowed is False
    assert len(client.responses.calls) == 1
    assert client.responses.calls[0]["model"] == "test-model"
    assert "Current customer message" in client.responses.calls[0]["input"]


@pytest.mark.asyncio
async def test_scope_classifier_only_accepts_exact_in_scope_label(tmp_path: Path) -> None:
    session = ConversationSession(phone="51999999999", qr_token="a" * 32)

    engine, _ = build_engine(tmp_path, "IN_SCOPE")
    assert await engine.is_in_scope(session, "Quiero una bebida") is True

    ambiguous_engine, _ = build_engine(tmp_path, "IN_SCOPE porque parece válido")
    assert await ambiguous_engine.is_in_scope(session, "Hazme una tarea") is False
