from __future__ import annotations

from typing import Literal

from pydantic import BaseModel


class LoginInitResponse(BaseModel):
    login_id: str
    expires_in: int


class LoginCheckResponse(BaseModel):
    status: Literal["pending", "approved", "rejected", "expired"]
    access_token: str | None = None
    token_type: str | None = None
