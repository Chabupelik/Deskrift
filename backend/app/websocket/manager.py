from __future__ import annotations

import logging

from fastapi import WebSocket

logger = logging.getLogger(__name__)


class ConnectionManager:
    def __init__(self) -> None:
        self.pc: WebSocket | None = None
        self.browser: WebSocket | None = None

    async def connect_pc(self, ws: WebSocket) -> None:
        if self.pc and self.pc != ws:
            logger.info("Closing existing PC connection")
            try:
                await self.pc.close(code=1000)
            except Exception as exc:
                logger.warning("Failed to close previous PC connection: %s", exc)
        await ws.accept()
        self.pc = ws
        logger.info("Desktop agent connected")

    def disconnect_pc(self, ws: WebSocket | None = None) -> None:
        if ws is None or self.pc == ws:
            self.pc = None
            logger.info("Desktop agent disconnected")

    async def connect_browser(self, ws: WebSocket) -> None:
        if self.browser and self.browser != ws:
            logger.info("Closing existing browser connection")
            await self.close_browser()
        await ws.accept()
        self.browser = ws
        logger.info("Browser connected")

    def disconnect_browser(self, ws: WebSocket | None = None) -> None:
        if ws is None or self.browser == ws:
            self.browser = None
            logger.info("Browser disconnected")

    async def close_browser(self) -> None:
        if self.browser:
            try:
                await self.browser.close(code=1000)
            except Exception as exc:
                logger.warning("Failed to close browser websocket cleanly: %s", exc)
            self.browser = None
            logger.info("Browser connection closed")

    async def send_to_browser(self, data: bytes) -> None:
        if self.browser:
            try:
                await self.browser.send_bytes(data)
            except Exception as exc:
                logger.warning("Failed to send bytes to browser: %s", exc)
                self.disconnect_browser()

    async def send_to_pc(self, data: bytes) -> None:
        if self.pc:
            try:
                await self.pc.send_bytes(data)
            except Exception as exc:
                logger.warning("Failed to send bytes to PC: %s", exc)
                self.disconnect_pc()

    @property
    def pc_online(self) -> bool:
        return self.pc is not None


manager = ConnectionManager()
