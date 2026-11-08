from __future__ import annotations

import json
import logging
from typing import Annotated

from fastapi import Depends, WebSocket, WebSocketDisconnect

from app.schemas.stream import WSBrowserSession
from app.services.auth_service import AuthService, AuthSrv
from app.websocket.manager import ConnectionManager, manager

logger = logging.getLogger(__name__)


class StreamService:
    def __init__(
        self,
        conn_manager: ConnectionManager,
        auth_service: AuthService,
    ) -> None:
        self.manager = conn_manager
        self.auth_service = auth_service

    async def handle_pc_stream(self, websocket: WebSocket) -> None:
        await self.manager.connect_pc(websocket)
        if self.manager.browser_online:
            await self.manager.send_text_to_pc(json.dumps({"type": "start_stream"}))
            await self.manager.send_text_to_browser(
                json.dumps({"type": "pc_status", "online": True})
            )
        else:
            await self.manager.send_text_to_pc(json.dumps({"type": "stop_stream"}))

        try:
            while True:
                data = await websocket.receive_bytes()
                await self.manager.send_to_browser(data)
        except WebSocketDisconnect as exc:
            logger.info("PC websocket disconnected: code=%s", exc.code)
        except Exception as exc:
            logger.exception("Unexpected error in PC websocket: %s", exc)
        finally:
            self.manager.disconnect_pc(websocket)
            if self.manager.browser_online:
                await self.manager.send_text_to_browser(
                    json.dumps({"type": "pc_status", "online": False})
                )

    async def handle_browser_stream(
        self,
        websocket: WebSocket,
        session: WSBrowserSession,
    ) -> None:
        await self.manager.connect_browser(websocket)
        if self.manager.pc_online:
            await self.manager.send_text_to_browser(
                json.dumps({"type": "pc_status", "online": True})
            )
            await self.manager.send_text_to_pc(json.dumps({"type": "start_stream"}))
        else:
            await self.manager.send_text_to_browser(
                json.dumps({"type": "pc_status", "online": False})
            )

        try:
            while True:
                data = await websocket.receive_text()
                await self.manager.send_to_pc(data.encode())
        except WebSocketDisconnect as exc:
            logger.info(
                "Browser websocket disconnected: code=%s, user=%s",
                exc.code,
                session.telegram_id,
            )
        except Exception as exc:
            logger.exception(
                "Unexpected error in browser websocket for user %s: %s",
                session.telegram_id,
                exc,
            )
        finally:
            self.manager.disconnect_browser(websocket)
            await self.auth_service.revoke_token(session.token, session.telegram_id)
            if self.manager.pc_online:
                await self.manager.send_text_to_pc(json.dumps({"type": "stop_stream"}))


def get_stream_service(
    auth_service: AuthSrv,
) -> StreamService:
    return StreamService(manager, auth_service)


StreamSrv = Annotated[StreamService, Depends(get_stream_service)]
