from __future__ import annotations

from pydantic import BaseModel


class AuthRequest(BaseModel):
    init_data: str


class TokenResponse(BaseModel):
    access_token: str
    token_type: str = "bearer"
