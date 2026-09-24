import hashlib
import hmac

from src.api.webhook import _incoming_text_messages


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
    assert _incoming_text_messages(payload) == [("m1", "51999999999", "hola")]


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
