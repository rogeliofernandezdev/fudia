from __future__ import annotations

import asyncio
import json
from pathlib import Path
from typing import Any

from config.settings import settings
from src.domain.models import ConversationSession
from src.infrastructure.ai_factory import build_ai

ROOT = Path(__file__).resolve().parents[1]


def load_cases(name: str) -> list[dict[str, Any]]:
    return json.loads(
        (ROOT / "evals" / name).read_text(
            encoding="utf-8"
        )
    )


async def main() -> None:
    if settings.llm_provider == "vertex" and not settings.google_cloud_project:
        raise SystemExit(
            "GOOGLE_CLOUD_PROJECT and Application Default Credentials are "
            "required for live Vertex evals."
        )
    if settings.llm_provider == "openai" and not settings.openai_api_key:
        raise SystemExit("OPENAI_API_KEY is required for live OpenAI evals.")

    stack = build_ai(settings, ROOT / "prompts")
    tiered: Any = stack.router
    # Every tier that can answer routing must pass the same cases.
    routers = [("lite", tiered.primary)]
    if tiered.fallback is not None:
        routers.append(("flash", tiered.fallback))
    failures: list[str] = []

    try:
        for tier, router in routers:
            print(f"Evaluating {stack.provider} router [{tier}] {router.model}")
            failures.extend(await evaluate(router, tier))
    finally:
        await stack.close()

    if failures:
        for failure in failures:
            print(f"FAIL {failure}")
        raise SystemExit(1)

    print("All live Concierge evals passed.")


async def evaluate(router: Any, tier: str) -> list[str]:
    failures: list[str] = []
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
        route = await router.route(
            session,
            str(case["input"]),
        )
        actual = (
            "OUT_OF_SCOPE"
            if route == "out_of_scope"
            else "IN_SCOPE"
        )
        if actual != case["expected"]:
            failures.append(
                f"[{tier}] scope: {case['input']!r}: "
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
                f"[{tier}] intent: {case['input']!r}: "
                f"{actual} != {case['expected']}"
            )
    return failures


if __name__ == "__main__":
    asyncio.run(main())
