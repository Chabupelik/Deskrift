from __future__ import annotations

from fastapi import APIRouter

from app.api.dependencies import AuthRateLimit, ClientIP, TelegramWebhookSecret
from app.schemas.auth import LoginCheckResponse, LoginInitResponse
from app.schemas.common import StatusResponse
from app.schemas.telegram import TelegramUpdate
from app.services.auth_service import AuthSrv

router = APIRouter(prefix="/auth", tags=["auth"])


@router.post("/login", response_model=LoginInitResponse)
async def request_login(
    auth_service: AuthSrv,
    client_ip: ClientIP,
    _: AuthRateLimit,
) -> LoginInitResponse:
    return await auth_service.request_login(client_ip)


@router.get("/login/{login_id}", response_model=LoginCheckResponse)
async def check_login(
    login_id: str,
    client_ip: ClientIP,
    auth_service: AuthSrv,
) -> LoginCheckResponse:
    return await auth_service.check_login(login_id, client_ip)


@router.post("/telegram/webhook", response_model=StatusResponse)
async def telegram_webhook(
    update: TelegramUpdate,
    auth_service: AuthSrv,
    _: TelegramWebhookSecret,
) -> StatusResponse:
    await auth_service.process_webhook(update)
    return StatusResponse()
