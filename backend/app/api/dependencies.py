from __future__ import annotations

from typing import Annotated

from fastapi import Depends, HTTPException, Request, WebSocket, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from starlette.requests import HTTPConnection

from app.core.config import get_settings
from app.core.security import decode_jwt
from app.db.redis import RedisClient

bearer = HTTPBearer()


def get_client_ip(request: HTTPConnection) -> str:
    cf_ip = request.headers.get("cf-connecting-ip")
    if cf_ip:
        return cf_ip.strip()

    x_real_ip = request.headers.get("x-real-ip")
    if x_real_ip:
        return x_real_ip.strip()

    x_forwarded_for = request.headers.get("x-forwarded-for")
    if x_forwarded_for:
        return x_forwarded_for.split(",")[0].strip()

    if request.client and request.client.host:
        return request.client.host

    return "unknown"


def get_current_user(
    credentials: HTTPAuthorizationCredentials = Depends(bearer),
) -> int:
    telegram_id = decode_jwt(credentials.credentials)
    if telegram_id is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid token"
        )
    settings = get_settings()
    if telegram_id != settings.allowed_telegram_id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN, detail="Access denied"
        )
    return telegram_id


CurrentUser = Annotated[int, Depends(get_current_user)]


async def ws_auth(websocket: WebSocket, token: str) -> int | None:
    telegram_id = decode_jwt(token)
    if telegram_id is None:
        await websocket.close(code=4001)
        return None
    settings = get_settings()
    if telegram_id != settings.allowed_telegram_id:
        await websocket.close(code=4003)
        return None
    return telegram_id


async def ws_pc_auth(websocket: WebSocket, token: str) -> bool:
    settings = get_settings()
    if token != settings.pc_secret:
        await websocket.close(code=4001)
        return False
    return True


class RateLimiter:
    def __init__(self, max_requests: int = 60, window_seconds: int = 60) -> None:
        self.max_requests = max_requests
        self.window_seconds = window_seconds

    async def __call__(
        self,
        request: Request,
        redis: RedisClient,
    ) -> None:
        ip = get_client_ip(request)
        path = request.url.path
        key = f"rate_limit:{ip}:{path}"

        async with redis.pipeline(transaction=True) as pipe:
            pipe.incr(key)
            pipe.expire(key, self.window_seconds, nx=True)
            res = await pipe.execute()

        current_requests = res[0]

        if current_requests > self.max_requests:
            raise HTTPException(
                status_code=status.HTTP_429_TOO_MANY_REQUESTS,
                detail="Слишком много запросов. Подождите, попробуйте позже.",
            )


rate_limiter = RateLimiter(max_requests=60, window_seconds=60)
RateLimit = Annotated[None, Depends(rate_limiter)]

auth_rate_limiter = RateLimiter(max_requests=10, window_seconds=60)
AuthRateLimit = Annotated[None, Depends(auth_rate_limiter)]
