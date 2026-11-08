from __future__ import annotations

from fastapi import WebSocket
import logging

logger = logging.getLogger(__name__)


class ConnectionManager:
    def __init__(self) -> None:
        self.pc: WebSocket | None = None
        self.browser: WebSocket | None = None

    async def connect_pc(self, ws: WebSocket) -> None:
        await ws.accept()
        self.pc = ws
        logger.info("Desktop agent connected")

    def disconnect_pc(self) -> None:
        self.pc = None
        logger.info("Desktop agent disconnected")

    async def connect_browser(self, ws: WebSocket) -> None:
        await ws.accept()
        self.browser = ws
        logger.info("Browser connected")

    def disconnect_browser(self) -> None:
        self.browser = None
        logger.info("Browser disconnected")

    async def send_to_browser(self, data: bytes) -> None:
        if self.browser:
            try:
                await self.browser.send_bytes(data)
            except Exception:
                self.disconnect_browser()

    async def send_to_pc(self, data: bytes) -> None:
        if self.pc:
            try:
                await self.pc.send_bytes(data)
            except Exception:
                self.disconnect_pc()

    @property
    def pc_online(self) -> bool:
        return self.pc is not None


manager = ConnectionManager()

