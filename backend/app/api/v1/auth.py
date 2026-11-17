from __future__ import annotations

import asyncio
from collections.abc import AsyncGenerator

from fastapi import APIRouter
from fastapi.responses import StreamingResponse

from app.api.dependencies import (
    AuthRateLimit,
    ClientIP,
    RateLimit,
    TelegramWebhookSecret,
)
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
    _: RateLimit,
) -> LoginCheckResponse:
    return await auth_service.check_login(login_id, client_ip)


@router.get("/login/{login_id}/events")
async def login_status_events(
    login_id: str,
    client_ip: ClientIP,
    auth_service: AuthSrv,
    _: RateLimit,
) -> StreamingResponse:
    """
    Server-Sent Events stream for login status.
    Pushes a JSON event every ~500 ms until the request is approved,
    rejected or expired — then closes the stream.
    Replaces client-side HTTP polling.
    """

    async def generator() -> AsyncGenerator[str, None]:
        try:
            redis = auth_service.redis
            deadline = asyncio.get_event_loop().time() + 125  # slightly over 120 s TTL

            while asyncio.get_event_loop().time() < deadline:
                raw = await redis.get(f"login_request:{login_id}")

                if raw is None:
                    yield f"data: {LoginCheckResponse(status='expired').model_dump_json()}\n\n"
                    return

                if raw in ("approved", "rejected"):
                    # Consume the state: creates session / clears keys / returns token
                    result = await auth_service.check_login(login_id, client_ip)
                    yield f"data: {result.model_dump_json()}\n\n"
                    return

                # Still pending — push heartbeat and wait
                yield f"data: {LoginCheckResponse(status='pending').model_dump_json()}\n\n"
                await asyncio.sleep(0.5)

            yield f"data: {LoginCheckResponse(status='expired').model_dump_json()}\n\n"
        except (asyncio.CancelledError, GeneratorExit):
            pass

    return StreamingResponse(
        generator(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",  # prevent Nginx from buffering SSE
            "Connection": "keep-alive",
        },
    )


@router.post("/telegram/webhook", response_model=StatusResponse)
async def telegram_webhook(
    update: TelegramUpdate,
    auth_service: AuthSrv,
    _: TelegramWebhookSecret,
) -> StatusResponse:
    await auth_service.process_webhook(update)
    return StatusResponse()
