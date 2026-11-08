from __future__ import annotations

from fastapi import APIRouter, Query, WebSocket, WebSocketDisconnect

from app.api.dependencies import ws_auth, ws_pc_auth
from app.db.redis import RedisClient
from app.services.auth_service import AuthSrv
from app.websocket.manager import manager

router = APIRouter(tags=["stream"])


@router.websocket("/ws/pc")
async def pc_websocket(
    websocket: WebSocket,
    token: str = Query(...),
) -> None:
    if not await ws_pc_auth(websocket, token):
        return

    await manager.connect_pc(websocket)
    try:
        while True:
            data = await websocket.receive_bytes()
            await manager.send_to_browser(data)
    except WebSocketDisconnect:
        pass
    finally:
        manager.disconnect_pc()


@router.websocket("/ws/browser")
async def browser_websocket(
    websocket: WebSocket,
    redis: RedisClient,
    auth_service: AuthSrv,
    token: str = Query(...),
) -> None:
    telegram_id = await ws_auth(websocket, token, redis)
    if telegram_id is None:
        return

    await manager.connect_browser(websocket)
    try:
        while True:
            data = await websocket.receive_text()
            await manager.send_to_pc(data.encode())
    except WebSocketDisconnect:
        pass
    finally:
        manager.disconnect_browser()
        await auth_service.revoke_token(token, telegram_id)
