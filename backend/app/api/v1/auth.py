from __future__ import annotations

from fastapi import APIRouter, HTTPException

from app.core.config import get_settings
from app.core.security import create_jwt, verify_telegram_init_data
from app.schemas.auth import AuthRequest, TokenResponse

router = APIRouter(prefix="/auth", tags=["auth"])


@router.post("/telegram", response_model=TokenResponse)
async def telegram_auth(body: AuthRequest) -> TokenResponse:
    settings = get_settings()

    user_data = verify_telegram_init_data(body.init_data)
    if user_data is None:
        raise HTTPException(status_code=401, detail="Invalid Telegram data")

    import json as _json

    user_info = _json.loads(user_data.get("user", "{}"))
    telegram_id = user_info.get("id")

    if telegram_id != settings.allowed_telegram_id:
        raise HTTPException(status_code=403, detail="Access denied")

    token = create_jwt(telegram_id)
    return TokenResponse(access_token=token)
