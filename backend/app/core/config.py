from __future__ import annotations

from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    telegram_bot_token: str = ""
    allowed_telegram_id: int = 0

    jwt_secret: str = ""
    jwt_expire_minutes: int = 60 * 24 * 7  # 7 дней

    pc_secret: str = ""

    show_docs: bool = False


@lru_cache
def get_settings() -> Settings:
    return Settings()
