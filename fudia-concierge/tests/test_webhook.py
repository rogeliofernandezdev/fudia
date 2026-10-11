import hashlib
import hmac

from src.api.webhook import _incoming_text_messages, _matches_configured_phone_id


def test_extracts_only_text_messages() -> None:
    payload = {
        "entry": [
            {
                "changes": [
                    {
                        "value": {
                            "messages": [
                                {
                                    "id": "m1",
                                    "from": "51999999999",
                                    "type": "text",
                                    "text": {"body": "hola"},
                                },
                                {"id": "m2", "from": "51999999999", "type": "image"},
                            ]
                        }
                    }
                ]
            }
        ]
    }
    assert _incoming_text_messages(payload) == [
        ("m1", "51999999999", "hola", "")
    ]


def test_extracts_inbound_whatsapp_number_context() -> None:
    payload = {
        "entry": [
            {
                "changes": [
                    {
                        "value": {
                            "metadata": {
                                "display_phone_number": "+51 987 654 321",
                                "phone_number_id": "meta-phone-123",
                            },
                            "messages": [
                                {
                                    "id": "m1",
                                    "from": "51999999999",
                                    "type": "text",
                                    "text": {"body": "hola"},
                                }
                            ],
                        }
                    }
                ]
            }
        ]
    }
    assert _incoming_text_messages(payload) == [
        (
            "m1",
            "51999999999",
            "hola",
            "meta-phone-123",
        )
    ]


def test_validates_meta_signature() -> None:
    from src.api.webhook import _valid_meta_signature

    payload = b'{"entry":[]}'
    secret = "test-secret"
    signature = "sha256=" + hmac.new(
        secret.encode("utf-8"), payload, hashlib.sha256
    ).hexdigest()

    assert _valid_meta_signature(payload, signature, secret)
    assert not _valid_meta_signature(payload, "sha256=bad", secret)
    assert not _valid_meta_signature(payload, signature, "")


def test_accepts_only_configured_global_whatsapp_phone_id() -> None:
    assert _matches_configured_phone_id(
        "meta-phone-123",
        "meta-phone-123",
    )
    assert not _matches_configured_phone_id(
        "meta-phone-other",
        "meta-phone-123",
    )
    assert _matches_configured_phone_id(
        "meta-phone-123",
        "",
    )


def test_interactive_whatsapp_replies_are_normalized_to_chat_text() -> None:
    from src.api.webhook import _incoming_text_messages

    messages = [
        {"id": "m-button", "from": "51999999999", "type": "button",
         "button": {"text": "Confirmar"}},
        {"id": "m-quick", "from": "51999999999", "type": "interactive",
         "interactive": {"type": "button_reply",
                         "button_reply": {"id": "yes", "title": "Sí"}}},
        {"id": "m-list", "from": "51999999999", "type": "interactive",
         "interactive": {"type": "list_reply",
                         "list_reply": {"id": "item", "title": "Ver carta"}}},
        {"id": "m-image", "from": "51999999999", "type": "image"},
    ]
    payload = {"entry": [{"changes": [{"value": {
        "metadata": {"phone_number_id": "meta-123"}, "messages": messages,
    }}]}]}
    assert _incoming_text_messages(payload) == [
        ("m-button", "51999999999", "Confirmar", "meta-123"),
        ("m-quick", "51999999999", "Sí", "meta-123"),
        ("m-list", "51999999999", "Ver carta", "meta-123"),
    ]


def test_malformed_meta_event_is_not_a_worker_exception() -> None:
    from src.api.webhook import _incoming_text_messages

    for bad in [
        {"entry": None},
        {"entry": ["not-object"]},
        {"entry": [{"changes": [None, {"value": 4}]}]},
        {"entry": [{"changes": [{"value": {"messages": [[], None, {"type": "text", "text": []}]}}]}]},
    ]:
        assert _incoming_text_messages(bad) == []


def test_webhook_rejects_oversized_signed_payload_before_enqueue() -> None:
    from types import SimpleNamespace

    from fastapi import FastAPI
    from fastapi.testclient import TestClient

    from src.api.webhook import router

    app = FastAPI()
    app.state.settings = SimpleNamespace(
        webhook_max_payload_bytes=1024, whatsapp_app_secret="test-secret",
    )
    app.include_router(router)
    with TestClient(app) as client:
        response = client.post(
            "/webhook",
            content=b"x" * 1025,
            headers={"Content-Length": "1025"},
        )
    assert response.status_code == 413
