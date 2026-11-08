from __future__ import annotations

import uuid
from typing import Annotated

import redis.asyncio as aioredis
from fastapi import Depends

from app.core.config import Settings, get_settings
from app.core.security import create_jwt
from app.db.redis import RedisClient
from app.integrations.telegram import telegram_client
from app.schemas.auth import LoginCheckResponse, LoginInitResponse
from app.schemas.telegram import TelegramUpdate
from app.websocket.manager import manager


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

        await telegram_client.send_login_request(
            chat_id=self.settings.allowed_telegram_id,
            client_ip=client_ip,
            login_id=login_id,
        )

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
        if not update.callback_query:
            return

        cq = update.callback_query
        if cq.from_user.id != self.settings.allowed_telegram_id:
            await telegram_client.answer_callback_query(
                callback_query_id=cq.id,
                text="Доступ запрещен",
            )
            return

        if not cq.data:
            return

        chat_id = (
            cq.message.chat.id if cq.message else self.settings.allowed_telegram_id
        )
        message_id = cq.message.message_id if cq.message else None

        if cq.data.startswith("approve:"):
            login_id = cq.data.split(":", 1)[1]
            current_status = await self.redis.get(f"login_request:{login_id}")
            if current_status == "pending":
                await self.redis.set(f"login_request:{login_id}", "approved", ex=120)
                await telegram_client.answer_callback_query(
                    callback_query_id=cq.id,
                    text="Вход разрешен",
                )
                if message_id:
                    await telegram_client.notify_login_approved(
                        chat_id=chat_id,
                        message_id=message_id,
                        telegram_id=self.settings.allowed_telegram_id,
                    )
            else:
                await telegram_client.answer_callback_query(
                    callback_query_id=cq.id,
                    text="Запрос устарел или уже обработан",
                )

        elif cq.data.startswith("reject:"):
            login_id = cq.data.split(":", 1)[1]
            await self.redis.set(f"login_request:{login_id}", "rejected", ex=60)
            await telegram_client.answer_callback_query(
                callback_query_id=cq.id,
                text="Вход отклонен",
            )
            if message_id:
                await telegram_client.notify_login_rejected(
                    chat_id=chat_id,
                    message_id=message_id,
                )

        elif cq.data.startswith("revoke:"):
            user_id = cq.data.split(":", 1)[1]
            await self.redis.delete(f"active_session:{user_id}")
            manager.disconnect_browser()
            await telegram_client.answer_callback_query(
                callback_query_id=cq.id,
                text="Сессия завершена",
            )
            if message_id:
                await telegram_client.notify_session_revoked(
                    chat_id=chat_id,
                    message_id=message_id,
                )


def get_auth_service(
    redis: RedisClient,
) -> AuthService:
    return AuthService(redis, get_settings())


AuthSrv = Annotated[AuthService, Depends(get_auth_service)]
