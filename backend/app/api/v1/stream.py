from __future__ import annotations

from fastapi import APIRouter, Query, WebSocket, WebSocketDisconnect

from app.api.dependencies import ws_auth, ws_pc_auth
from app.websocket.manager import manager

router = APIRouter(tags=["stream"])


@router.websocket("/ws/pc")
async def pc_websocket(websocket: WebSocket, token: str = Query(...)):
    """WebSocket для desktop агента (Go стример)."""
    if not await ws_pc_auth(websocket, token):
        return

    await manager.connect_pc(websocket)
    try:
        while True:
            # Получаем H.264 chunks от PC и пересылаем браузеру
            data = await websocket.receive_bytes()
            await manager.send_to_browser(data)
    except WebSocketDisconnect:
        manager.disconnect_pc()


@router.websocket("/ws/browser")
async def browser_websocket(websocket: WebSocket, token: str = Query(...)):
    """WebSocket для браузера (плеер)."""
    if not await ws_auth(websocket, token):
        return

    await manager.connect_browser(websocket)
    try:
        while True:
            # Получаем input события от браузера и пересылаем PC
            data = await websocket.receive_text()
            await manager.send_to_pc(data.encode())
    except WebSocketDisconnect:
        manager.disconnect_browser()
