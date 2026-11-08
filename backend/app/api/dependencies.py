from __future__ import annotations

from fastapi import Depends, HTTPException, WebSocket, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

from app.core.config import get_settings
from app.core.security import decode_jwt

bearer = HTTPBearer()


def get_current_user(credentials: HTTPAuthorizationCredentials = Depends(bearer)) -> int:
    telegram_id = decode_jwt(credentials.credentials)
    if telegram_id is None:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid token")
    settings = get_settings()
    if telegram_id != settings.allowed_telegram_id:
        raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail="Access denied")
    return telegram_id


async def ws_auth(websocket: WebSocket, token: str) -> int | None:
    """Аутентификация для WebSocket соединений браузера."""
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
    """Аутентификация для WebSocket соединений desktop агента."""
    settings = get_settings()
    if token != settings.pc_secret:
        await websocket.close(code=4001)
        return False
    return True

