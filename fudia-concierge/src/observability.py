from __future__ import annotations

import hashlib
import json
import logging
from contextvars import ContextVar, Token
from typing import Any
from uuid import uuid4

_trace_id: ContextVar[str] = ContextVar(
    "fudia_concierge_trace_id",
    default="",
)


def fingerprint(value: str) -> str:
    if not value:
        return ""
    return hashlib.sha256(
        value.encode("utf-8")
    ).hexdigest()[:16]


def begin_trace(seed: str = "") -> Token[str]:
    trace_id = (
        fingerprint(seed)
        if seed
        else uuid4().hex[:16]
    )
    return _trace_id.set(trace_id)


def end_trace(token: Token[str]) -> None:
    _trace_id.reset(token)


def current_trace_id() -> str:
    return _trace_id.get()


def log_event(
    logger: logging.Logger,
    event: str,
    **fields: Any,
) -> None:
    trace_id = current_trace_id()
    payload = {
        "event": event,
        **({"traceId": trace_id} if trace_id else {}),
        **fields,
    }
    logger.info(
        json.dumps(
            payload,
            ensure_ascii=False,
            default=str,
            separators=(",", ":"),
        )
    )
