"""ASGI entrypoint; the worker imports the factory without creating an HTTP app."""

from src.application import create_app

app = create_app()
