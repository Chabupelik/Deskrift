from __future__ import annotations

from fastapi import APIRouter, WebSocket

from app.api.dependencies import WSBrowserAuth, WSPCAuth
from app.services.stream_service import StreamSrv

router = APIRouter(tags=["stream"])


@router.websocket("/ws/pc")
async def pc_websocket(
    websocket: WebSocket,
    _: WSPCAuth,
    stream_service: StreamSrv,
) -> None:
    await stream_service.handle_pc_stream(websocket)


@router.websocket("/ws/browser")
async def browser_websocket(
    websocket: WebSocket,
    session: WSBrowserAuth,
    stream_service: StreamSrv,
) -> None:
    await stream_service.handle_browser_stream(websocket, session)
