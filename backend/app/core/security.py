from __future__ import annotations

import hashlib
import hmac
import time
from datetime import UTC, datetime, timedelta

import jwt

from app.core.config import get_settings


def verify_telegram_init_data(init_data: str) -> dict | None:
    """Верифицирует Telegram WebApp initData и возвращает данные пользователя."""
    settings = get_settings()

    params = dict(pair.split("=", 1) for pair in init_data.split("&") if "=" in pair)
    received_hash = params.pop("hash", None)
    if not received_hash:
        return None

    data_check_string = "\n".join(f"{k}={v}" for k, v in sorted(params.items()))
    secret_key = hmac.new(
        b"WebAppData", settings.telegram_bot_token.encode(), hashlib.sha256
    ).digest()
    computed_hash = hmac.new(
        secret_key, data_check_string.encode(), hashlib.sha256
    ).hexdigest()

    if not hmac.compare_digest(computed_hash, received_hash):
        return None

    # Проверяем что initData не старше 24 часов
    auth_date = int(params.get("auth_date", 0))
    if time.time() - auth_date > 86400:
        return None

    return params


def create_jwt(telegram_id: int) -> str:
    settings = get_settings()
    payload = {
        "sub": str(telegram_id),
        "exp": datetime.now(UTC) + timedelta(minutes=settings.jwt_expire_minutes),
    }
    return jwt.encode(payload, settings.jwt_secret, algorithm="HS256")


def decode_jwt(token: str) -> int | None:
    settings = get_settings()
    try:
        payload = jwt.decode(token, settings.jwt_secret, algorithms=["HS256"])
        return int(payload["sub"])
    except Exception:
        return None
