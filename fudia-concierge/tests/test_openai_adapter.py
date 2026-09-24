from pathlib import Path
from typing import Any

import pytest

from src.domain.models import ChatMessage, ConversationSession
from src.infrastructure.openai_adapter import (
    OpenAIIntentRouter,
    OpenAIScopeClassifier,
)


class FakeResponse:
    def __init__(self, output_text: str) -> None:
        self.output_text = output_text


class FakeResponses:
    def __init__(self, output_text: str) -> None:
        self.output_text = output_text
        self.calls: list[dict[str, Any]] = []

    async def create(self, **kwargs: Any) -> FakeResponse:
        self.calls.append(kwargs)
        return FakeResponse(self.output_text)


class FakeOpenAI:
    def __init__(self, output_text: str) -> None:
        self.responses = FakeResponses(output_text)


def prompt(
    tmp_path: Path,
    name: str,
    content: str,
) -> Path:
    path = tmp_path / name
    path.write_text(content, encoding="utf-8")
    return path


@pytest.mark.asyncio
async def test_scope_classifier_is_fail_closed(
    tmp_path: Path,
) -> None:
    client = FakeOpenAI("OUT_OF_SCOPE")
    classifier = OpenAIScopeClassifier(
        client,  # type: ignore[arg-type]
        "test-model",
        prompt(
            tmp_path,
            "scope.md",
            "scope classification only",
        ),
    )
    session = ConversationSession(
        phone="51999999999",
        qr_token="a" * 32,
        messages=[
            ChatMessage(
                role="assistant",
                content="¿Qué deseas pedir?",
            ),
        ],
    )

    assert (
        await classifier.is_in_scope(
            session,
            "¿Quién fue Napoleón?",
        )
        is False
    )
    assert (
        client.responses.calls[0]["instructions"]
        == "scope classification only"
    )

    ambiguous = FakeOpenAI(
        "IN_SCOPE porque parece válido"
    )
    ambiguous_classifier = OpenAIScopeClassifier(
        ambiguous,  # type: ignore[arg-type]
        "test-model",
        prompt(
            tmp_path,
            "scope2.md",
            "scope",
        ),
    )
    assert (
        await ambiguous_classifier.is_in_scope(
            session,
            "hazme una tarea",
        )
        is False
    )


@pytest.mark.asyncio
async def test_intent_router_accepts_only_known_intents(
    tmp_path: Path,
) -> None:
    session = ConversationSession(
        phone="51999999999",
        qr_token="a" * 32,
    )
    menu_client = FakeOpenAI("menu")
    router = OpenAIIntentRouter(
        menu_client,  # type: ignore[arg-type]
        "test-model",
        prompt(
            tmp_path,
            "intent.md",
            "route",
        ),
    )
    assert (
        await router.route(
            session,
            "¿Qué bebidas tienen?",
        )
        == "menu"
    )

    invalid_client = FakeOpenAI("anything")
    invalid_router = OpenAIIntentRouter(
        invalid_client,  # type: ignore[arg-type]
        "test-model",
        prompt(
            tmp_path,
            "intent2.md",
            "route",
        ),
    )
    assert (
        await invalid_router.route(
            session,
            "quiero dos",
        )
        == "order"
    )
