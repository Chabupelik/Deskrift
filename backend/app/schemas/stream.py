from __future__ import annotations

from typing import Literal

from pydantic import BaseModel


class InputEvent(BaseModel):
    type: Literal["mousemove", "mousedown", "mouseup", "keydown", "keyup"]
    x: int | None = None
    y: int | None = None
    button: Literal["left", "right"] | None = None
    key: int | None = None


class StreamStatusResponse(BaseModel):
    pc_online: bool
    quality: str
