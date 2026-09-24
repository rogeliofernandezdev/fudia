from __future__ import annotations

import asyncio
import json
from pathlib import Path
from typing import Any

from openai import AsyncOpenAI

from config.settings import settings
from src.domain.models import ConversationSession
from src.infrastructure.openai_adapter import (
    OpenAIIntentRouter,
    OpenAIScopeClassifier,
)

ROOT = Path(__file__).resolve().parents[1]


def load_cases(name: str) -> list[dict[str, Any]]:
    return json.loads(
        (ROOT / "evals" / name).read_text(
            encoding="utf-8"
        )
    )


async def main() -> None:
    if not settings.openai_api_key:
        raise SystemExit(
            "OPENAI_API_KEY is required for live evals."
        )

    client = AsyncOpenAI(
        api_key=settings.openai_api_key
    )
    scope = OpenAIScopeClassifier(
        client,
        settings.openai_model,
        ROOT / "prompts" / "scope_router.md",
    )
    router = OpenAIIntentRouter(
        client,
        settings.openai_model,
        ROOT / "prompts" / "intent_router.md",
    )
    failures: list[str] = []

    try:
        for case in (
            load_cases("scope_cases.json")
            + load_cases(
                "prompt_injection_cases.json"
            )
        ):
            session = ConversationSession(
                phone="eval",
                qr_token="a" * 32,
            )
            actual = (
                "IN_SCOPE"
                if await scope.is_in_scope(
                    session,
                    str(case["input"]),
                )
                else "OUT_OF_SCOPE"
            )
            if actual != case["expected"]:
                failures.append(
                    f"scope: {case['input']!r}: "
                    f"{actual} != {case['expected']}"
                )

        for case in load_cases(
            "intent_cases.json"
        ):
            session = ConversationSession(
                phone="eval",
                qr_token="a" * 32,
                awaiting_confirmation=bool(
                    case.get(
                        "awaitingConfirmation",
                        False,
                    )
                ),
            )
            actual = await router.route(
                session,
                str(case["input"]),
            )
            if actual != case["expected"]:
                failures.append(
                    f"intent: {case['input']!r}: "
                    f"{actual} != {case['expected']}"
                )
    finally:
        await client.close()

    if failures:
        for failure in failures:
            print(f"FAIL {failure}")
        raise SystemExit(1)

    print("All live Concierge evals passed.")


if __name__ == "__main__":
    asyncio.run(main())
