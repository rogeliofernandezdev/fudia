import ssl
from typing import Literal
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit

from pydantic import Field, SecretStr, field_validator, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
        hide_input_in_errors=True,
    )

    env: str = "development"
    log_level: str = "INFO"

    fudia_api_url: str = "http://localhost:8080"
    fudia_concierge_api_key: str = ""

    redis_url: str = "redis://localhost:6379/0"
    # Private CA supplied as a mounted file OR an in-memory Secret Manager PEM.
    # Worker pools do not require a secret volume. Never disable TLS checks.
    redis_ca_certfile: str = ""
    redis_ca_cert_pem: SecretStr = Field(default=SecretStr(""), repr=False, exclude=True)
    session_ttl_seconds: int = 43_200
    # Last venue visited by QR: lets a diner whose table service ended order
    # delivery or pickup from that venue (90 days by default).
    customer_memory_ttl_seconds: int = Field(default=7_776_000, ge=86_400)
    # A table context with no concierge order ends after this idle time; an
    # order ends it when foods-backend reports the account finished.
    table_idle_seconds: int = Field(default=10_800, ge=600)
    # Short table code of the QR start message ("Mesa 7 · K7P2"): how long a
    # scan stays valid before the diner sends the prefilled message.
    start_code_ttl_seconds: int = Field(default=600, ge=60, le=86_400)
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
    # Cloud Run sends SIGKILL ~10 s after SIGTERM: drain within that window.
    queue_shutdown_grace_seconds: float = Field(
        default=8.0,
        gt=0,
        le=120,
    )
    # Per-conversation order: how long a delivery waits for its predecessor
    # before it is deferred, and after how long a missing predecessor is
    # skipped so it cannot block the conversation forever.
    queue_turn_wait_seconds: float = Field(default=8.0, gt=0, le=60)
    queue_order_max_wait_seconds: float = Field(default=120.0, gt=0, le=3_600)
    queue_order_ttl_seconds: int = Field(default=604_800, ge=3_600)
    # Retention: acknowledged stream entries and dead letters. Pending or
    # undelivered entries are never trimmed.
    queue_retention_seconds: int = Field(default=86_400, ge=3_600)
    queue_dead_letter_retention_seconds: int = Field(default=1_209_600, ge=86_400)
    queue_maintenance_interval_seconds: float = Field(default=300.0, ge=10)
    # Processing records: finished ones, and ones that may still be needed to
    # recover an operation (must outlive the dead-letter retention).
    processing_record_ttl_seconds: int = Field(default=259_200, ge=3_600)
    processing_pending_record_ttl_seconds: int = Field(default=1_209_600, ge=86_400)
    # Window to wait for a Meta status receipt before resending an uncertain
    # reply once.
    outbound_reconcile_seconds: float = Field(default=30.0, gt=0, le=600)
    # Resends of a reply whose delivery stayed unconfirmed after the window.
    # 1 favours an answer (the text may arrive twice); 0 never duplicates it
    # (the diner may get no answer). Meta offers no idempotent send.
    outbound_max_resends: int = Field(default=1, ge=0, le=3)

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

    @field_validator(
        "whatsapp_token",
        "whatsapp_phone_id",
        "whatsapp_verify_token",
        "whatsapp_app_secret",
        mode="before",
    )
    @classmethod
    def _strip_meta_credentials(cls, value: object) -> object:
        # A secret created with `echo` keeps a trailing newline: the webhook
        # would reject every signature (403) and the verify challenge.
        return value.strip() if isinstance(value, str) else value

    @property
    def redis_connection_url(self) -> str:
        pem = self.redis_ca_cert_pem.get_secret_value()
        if not self.redis_ca_certfile and not pem:
            return self.redis_url
        if self.redis_ca_certfile and pem:
            raise ValueError("Use only one of REDIS_CA_CERTFILE or REDIS_CA_CERT_PEM.")
        parts = urlsplit(self.redis_url)
        if parts.scheme != "rediss":
            raise ValueError("Redis private CA requires rediss:// (TLS).")
        query = dict(parse_qsl(parts.query, keep_blank_values=True))
        if any(key.startswith("ssl_") for key in query):
            raise ValueError("Configure Redis TLS through its private CA setting, not URI overrides.")
        # Validate either source before connecting; the Redis driver loads cadata
        # directly into its SSLContext without writing certificate files.
        if pem:
            ssl.create_default_context(cadata=pem)
            query["ssl_ca_data"] = pem
        else:
            ssl.create_default_context(cafile=self.redis_ca_certfile)
            query["ssl_ca_certs"] = self.redis_ca_certfile
        query.update(
            {
                "ssl_cert_reqs": "required",
                "ssl_check_hostname": "True",
                "socket_connect_timeout": "10",
                "socket_timeout": "10",
            }
        )
        return urlunsplit(parts._replace(query=urlencode(query)))

    @model_validator(mode="after")
    def validate_recovery_windows(self) -> "Settings":
        if self.queue_order_ttl_seconds <= self.session_ttl_seconds:
            raise ValueError(
                "QUEUE_ORDER_TTL_SECONDS must exceed SESSION_TTL_SECONDS: the "
                "QR context compares against per-conversation sequences."
            )
        if self.processing_pending_record_ttl_seconds < self.queue_dead_letter_retention_seconds:
            raise ValueError(
                "PROCESSING_PENDING_RECORD_TTL_SECONDS must cover "
                "QUEUE_DEAD_LETTER_RETENTION_SECONDS so a requeued dead "
                "letter never repeats its effects."
            )
        return self

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
        missing = [name for name, value in required.items() if not str(value).strip()]
        if missing:
            raise ValueError("Missing production settings: " + ", ".join(missing))
        return self


settings = Settings()
