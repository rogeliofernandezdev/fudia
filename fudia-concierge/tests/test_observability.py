import logging

from src.observability import (
    begin_trace,
    current_trace_id,
    end_trace,
    fingerprint,
    log_event,
)


def test_trace_context_is_added_and_reset(
    caplog,
) -> None:
    logger = logging.getLogger(
        "fudia-concierge-test"
    )
    token = begin_trace("wamid.123")
    expected = fingerprint("wamid.123")

    try:
        assert current_trace_id() == expected
        with caplog.at_level(logging.INFO):
            log_event(
                logger,
                "test.event",
                result="ok",
            )
        assert expected in caplog.text
        assert "test.event" in caplog.text
    finally:
        end_trace(token)

    assert current_trace_id() == ""
