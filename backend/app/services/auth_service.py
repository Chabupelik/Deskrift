from __future__ import annotations

import uuid
from typing import Annotated

import redis.asyncio as aioredis
from fastapi import Depends

from app.core.config import Settings, get_settings
from app.core.security import create_jwt
from app.db.redis import RedisClient
from app.schemas.auth import LoginCheckResponse, LoginInitResponse
from app.schemas.telegram import TelegramUpdate


class AuthService:
    def __init__(
        self,
        redis: aioredis.Redis,
        settings: Settings | None = None,
    ) -> None:
        self.redis = redis
        self.settings = settings or get_settings()

    async def request_login(self, client_ip: str) -> LoginInitResponse:
        login_id = uuid.uuid4().hex
        expires_in = 120
        await self.redis.set(f"login_request:{login_id}", "pending", ex=expires_in)
        return LoginInitResponse(login_id=login_id, expires_in=expires_in)

    async def check_login(self, login_id: str) -> LoginCheckResponse:
        status_val = await self.redis.get(f"login_request:{login_id}")
        if status_val is None:
            return LoginCheckResponse(status="expired")

        if status_val == "approved":
            await self.redis.delete(f"login_request:{login_id}")
            token = create_jwt(self.settings.allowed_telegram_id)
            await self.redis.set(
                f"active_session:{self.settings.allowed_telegram_id}",
                token,
                ex=self.settings.jwt_expire_minutes * 60,
            )
            return LoginCheckResponse(
                status="approved",
                access_token=token,
                token_type="bearer",
            )

        if status_val == "rejected":
            await self.redis.delete(f"login_request:{login_id}")
            return LoginCheckResponse(status="rejected")

        return LoginCheckResponse(status="pending")

    async def process_webhook(self, update: TelegramUpdate) -> None:
        pass


def get_auth_service(
    redis: RedisClient,
) -> AuthService:
    return AuthService(redis, get_settings())


AuthSrv = Annotated[AuthService, Depends(get_auth_service)]
