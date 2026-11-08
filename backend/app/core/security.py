from __future__ import annotations

import logging
from datetime import UTC, datetime, timedelta

import jwt

from app.core.config import get_settings

logger = logging.getLogger(__name__)


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
    except jwt.PyJWTError as exc:
        logger.warning("JWT validation failed: %s", exc)
        return None
    except Exception as exc:
        logger.exception("Unexpected error decoding JWT: %s", exc)
        return None
