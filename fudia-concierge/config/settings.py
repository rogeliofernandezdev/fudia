from pydantic import Field, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    env: str = "development"
    log_level: str = "INFO"

    fudia_api_url: str = "http://localhost:8080"
    fudia_concierge_api_key: str = ""

    redis_url: str = "redis://localhost:6379/0"
    session_ttl_seconds: int = 43_200
    queue_stream: str = "fudia:concierge:inbound"
    queue_group: str = "fudia-concierge"
    queue_consumer: str = ""
    queue_max_attempts: int = 5
    queue_visibility_timeout_ms: int = 30_000
    queue_worker_concurrency: int = Field(
        default=8,
        ge=1,
        le=100,
    )
    queue_shutdown_grace_seconds: float = Field(
        default=15.0,
        gt=0,
        le=120,
    )

    conversation_lock_ttl_seconds: int = 30
    conversation_lock_wait_seconds: float = 8.0
    rate_limit_messages: int = 30
    rate_limit_window_seconds: int = 60
    max_message_chars: int = 2_000

    openai_api_key: str = ""
    openai_model: str = "gpt-5.6-luna"

    whatsapp_token: str = ""
    whatsapp_phone_id: str = ""
    whatsapp_verify_token: str = ""
    whatsapp_app_secret: str = ""
    whatsapp_graph_version: str = ""

    @model_validator(mode="after")
    def validate_production_secrets(self) -> "Settings":
        if self.env.lower() != "production":
            return self

        required = {
            "FUDIA_API_URL": self.fudia_api_url,
            "FUDIA_CONCIERGE_API_KEY": self.fudia_concierge_api_key,
            "REDIS_URL": self.redis_url,
            "OPENAI_API_KEY": self.openai_api_key,
            "WHATSAPP_TOKEN": self.whatsapp_token,
            "WHATSAPP_VERIFY_TOKEN": self.whatsapp_verify_token,
            "WHATSAPP_APP_SECRET": self.whatsapp_app_secret,
            "WHATSAPP_GRAPH_VERSION": self.whatsapp_graph_version,
        }
        missing = [
            name
            for name, value in required.items()
            if not str(value).strip()
        ]
        if missing:
            raise ValueError(
                "Missing production settings: "
                + ", ".join(missing)
            )
        return self


settings = Settings()
