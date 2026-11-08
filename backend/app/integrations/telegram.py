from __future__ import annotations

import logging
from typing import Any

import httpx

from app.core.config import get_settings

logger = logging.getLogger(__name__)


class TelegramClient:
    def __init__(self, token: str | None = None) -> None:
        self.token = token or get_settings().telegram_bot_token
        self.base_url = f"https://api.telegram.org/bot{self.token}"

    async def send_message(
        self,
        chat_id: int,
        text: str,
        reply_markup: dict[str, Any] | None = None,
    ) -> int | None:
        if not self.token:
            return None

        url = f"{self.base_url}/sendMessage"
        payload: dict[str, Any] = {
            "chat_id": chat_id,
            "text": text,
            "parse_mode": "HTML",
        }
        if reply_markup:
            payload["reply_markup"] = reply_markup

        try:
            async with httpx.AsyncClient(timeout=5.0) as client:
                res = await client.post(url, json=payload)
                data = res.json()
                if data.get("ok"):
                    return data.get("result", {}).get("message_id")
                logger.error("Telegram send_message failed: %s", data)
        except Exception as e:
            logger.error("Telegram request error: %s", e)
        return None

    async def edit_message_text(
        self,
        chat_id: int,
        message_id: int,
        text: str,
        reply_markup: dict[str, Any] | None = None,
    ) -> None:
        if not self.token:
            return

        url = f"{self.base_url}/editMessageText"
        payload: dict[str, Any] = {
            "chat_id": chat_id,
            "message_id": message_id,
            "text": text,
            "parse_mode": "HTML",
        }
        if reply_markup is not None:
            payload["reply_markup"] = reply_markup

        try:
            async with httpx.AsyncClient(timeout=5.0) as client:
                await client.post(url, json=payload)
        except Exception as e:
            logger.error("Telegram edit_message_text error: %s", e)

    async def answer_callback_query(
        self,
        callback_query_id: str,
        text: str | None = None,
    ) -> None:
        if not self.token:
            return

        url = f"{self.base_url}/answerCallbackQuery"
        payload: dict[str, Any] = {
            "callback_query_id": callback_query_id,
        }
        if text:
            payload["text"] = text

        try:
            async with httpx.AsyncClient(timeout=5.0) as client:
                await client.post(url, json=payload)
        except Exception as e:
            logger.error("Telegram answer_callback_query error: %s", e)

    async def send_login_request(
        self,
        chat_id: int,
        client_ip: str,
        login_id: str,
    ) -> int | None:
        text = (
            "<b>🔔 Попытка входа в Deskrift</b>\n\n"
            f"<b>IP адрес:</b> <code>{client_ip}</code>\n\n"
            "Разрешить доступ к рабочему столу?"
        )
        reply_markup = {
            "inline_keyboard": [
                [
                    {
                        "text": "✅ Разрешить",
                        "callback_data": f"approve:{login_id}",
                    },
                    {
                        "text": "❌ Отклонить",
                        "callback_data": f"reject:{login_id}",
                    },
                ]
            ]
        }
        return await self.send_message(chat_id, text, reply_markup)

    async def notify_login_approved(
        self,
        chat_id: int,
        message_id: int,
        telegram_id: int,
    ) -> None:
        text = "<b>✅ Вход разрешен</b>"
        reply_markup = {
            "inline_keyboard": [
                [
                    {
                        "text": "🛑 Завершить сессию",
                        "callback_data": f"revoke:{telegram_id}",
                    }
                ]
            ]
        }
        await self.edit_message_text(chat_id, message_id, text, reply_markup)

    async def notify_login_rejected(
        self,
        chat_id: int,
        message_id: int,
    ) -> None:
        text = "<b>❌ Вход отклонен</b>"
        await self.edit_message_text(chat_id, message_id, text, {"inline_keyboard": []})

    async def notify_session_revoked(
        self,
        chat_id: int,
        message_id: int,
    ) -> None:
        text = "<b>🛑 Сессия завершена</b>"
        await self.edit_message_text(chat_id, message_id, text, {"inline_keyboard": []})


telegram_client = TelegramClient()
