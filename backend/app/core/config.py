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
    jwt_expire_minutes: int = 60 * 24 * 7

    pc_secret: str = ""

    redis_host: str = "redis"
    redis_port: int = 6379
    redis_db: int = 0

    show_docs: bool = False

    @property
    def redis_url(self) -> str:
        return f"redis://{self.redis_host}:{self.redis_port}/{self.redis_db}"


@lru_cache
def get_settings() -> Settings:
    return Settings()
