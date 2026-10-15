from typing import Literal

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
    # Split HTTP ingress and queue processing into independently scalable roles.
    # True preserves the existing single-container development deployment.
    worker_enabled: bool = True
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
    webhook_max_payload_bytes: int = Field(default=262_144, ge=1024, le=1_048_576)

    # Model provider. Vertex AI (Gemini) is the default; OpenAI stays optional.
    llm_provider: Literal["vertex", "openai"] = "vertex"
    # Per-call budget for a single model request, and the tool-loop ceiling.
    llm_timeout_seconds: float = Field(default=20.0, gt=0, le=120)
    llm_max_retries: int = Field(default=2, ge=0, le=5)
    llm_max_output_tokens: int = Field(default=1_024, ge=64, le=8_192)
    llm_max_tool_rounds: int = Field(default=8, ge=1, le=16)

    # Vertex AI uses Application Default Credentials (Cloud Run service
    # account or `gcloud auth application-default login`); no API key.
    google_cloud_project: str = ""
    google_cloud_location: str = "global"
    # Flash-Lite: routing, menu questions and service requests.
    # Flash: order building and confirmation, and escalation from Flash-Lite.
    gemini_model_lite: str = "gemini-3.5-flash-lite"
    gemini_model_flash: str = "gemini-3.5-flash"

    openai_api_key: str = ""
    openai_model: str = "gpt-5.6-luna"
    # Lighter OpenAI tier; empty means "use OPENAI_MODEL for every task".
    openai_model_lite: str = ""

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
            "WHATSAPP_TOKEN": self.whatsapp_token,
            "WHATSAPP_PHONE_ID": self.whatsapp_phone_id,
            "WHATSAPP_VERIFY_TOKEN": self.whatsapp_verify_token,
            "WHATSAPP_APP_SECRET": self.whatsapp_app_secret,
            "WHATSAPP_GRAPH_VERSION": self.whatsapp_graph_version,
        }
        if self.llm_provider == "vertex":
            required["GOOGLE_CLOUD_PROJECT"] = self.google_cloud_project
            required["GOOGLE_CLOUD_LOCATION"] = self.google_cloud_location
            required["GEMINI_MODEL_LITE"] = self.gemini_model_lite
            required["GEMINI_MODEL_FLASH"] = self.gemini_model_flash
        else:
            required["OPENAI_API_KEY"] = self.openai_api_key
            required["OPENAI_MODEL"] = self.openai_model
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
