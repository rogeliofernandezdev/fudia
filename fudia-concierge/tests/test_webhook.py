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
