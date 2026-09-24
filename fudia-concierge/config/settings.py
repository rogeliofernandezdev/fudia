from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8", extra="ignore")

    env: str = "development"
    log_level: str = "INFO"
    fudia_api_url: str = "http://localhost:8080"
    fudia_concierge_api_key: str = ""
    redis_url: str = "redis://localhost:6379/0"
    session_ttl_seconds: int = 43_200

    openai_api_key: str = ""
    openai_model: str = "gpt-5.6-luna"

    whatsapp_token: str = ""
    whatsapp_phone_id: str = ""
    whatsapp_verify_token: str = ""
    whatsapp_app_secret: str = ""
    whatsapp_graph_version: str = ""


settings = Settings()
