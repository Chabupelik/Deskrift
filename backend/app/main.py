from __future__ import annotations

import logging
import sys
import time
from contextlib import asynccontextmanager

from fastapi import FastAPI, Request, Response, status
from fastapi.exceptions import RequestValidationError
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from pydantic import ValidationError
from starlette.exceptions import HTTPException as StarletteHTTPException

from app.api.dependencies import get_client_ip
from app.api.v1 import auth, stream
from app.core.config import get_settings
from app.db.redis import RedisClient, close_redis_pool, get_raw_redis_client
from app.schemas.common import ResponseError, ResponseOK

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    handlers=[logging.StreamHandler(sys.stdout)],
)
logger = logging.getLogger("deskrift_api")


@asynccontextmanager
async def lifespan(app: FastAPI):
    try:
        await get_raw_redis_client().ping()
        logger.info("Redis connection established successfully")
    except Exception as e:
        logger.error("Connection check failed: %s", e)
        sys.exit(1)

    yield

    try:
        await close_redis_pool()
        logger.info("Redis connection closed cleanly")
    except Exception as e:
        logger.error("Error during connection teardown: %s", e)


settings = get_settings()

app = FastAPI(
    title="Deskrift API",
    version="1.0.0",
    docs_url="/docs" if settings.show_docs else None,
    redoc_url="/redoc" if settings.show_docs else None,
    openapi_url="/openapi.json" if settings.show_docs else None,
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=False,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.middleware("http")
async def log_requests_middleware(request: Request, call_next):
    if request.url.path in ("/api/ping", "/health"):
        return await call_next(request)

    start_time = time.time()
    response: Response = await call_next(request)
    process_time_ms = (time.time() - start_time) * 1000
    client_ip = get_client_ip(request)
    url_path = (
        f"{request.url.path}?{request.url.query}"
        if request.url.query
        else request.url.path
    )

    logger.info(
        '%s - "%s %s" %s (%.2fms)',
        client_ip,
        request.method,
        url_path,
        response.status_code,
        process_time_ms,
    )
    return response


@app.exception_handler(StarletteHTTPException)
async def http_exception_handler(request: Request, exc: StarletteHTTPException):
    return JSONResponse(
        status_code=exc.status_code,
        content=ResponseError(message=exc.detail).model_dump(),
    )


@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request: Request, exc: RequestValidationError):
    errors = [
        f"[{' -> '.join(str(x) for x in e['loc'])}]: {e['msg']}" for e in exc.errors()
    ]
    return JSONResponse(
        status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
        content=ResponseError(
            message="Ошибка валидации входных данных", details=errors
        ).model_dump(),
    )


@app.exception_handler(ValidationError)
async def pydantic_validation_exception_handler(request: Request, exc: ValidationError):
    errors = [
        f"[{' -> '.join(str(x) for x in e['loc'])}]: {e['msg']}" for e in exc.errors()
    ]
    return JSONResponse(
        status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
        content=ResponseError(
            message="Ошибка валидации входных данных", details=errors
        ).model_dump(),
    )


@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception):
    logger.exception("%s %s: %s", request.method, request.url.path, exc)
    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content=ResponseError(
            message="Внутренняя ошибка сервера. Пожалуйста, попробуйте позже."
        ).model_dump(),
    )


@app.get(
    "/health",
    response_model=ResponseOK[dict],
    summary="Liveness check",
)
@app.get(
    "/api/ping",
    summary="Legacy ping check",
)
async def health_check():
    return ResponseOK(data={"message": "API is alive"})


@app.get(
    "/ready",
    response_model=ResponseOK[dict],
    summary="Readiness check",
)
async def ready_check(
    redis: RedisClient,
):
    try:
        await redis.ping()
        return ResponseOK(data={"redis": "connected"})
    except Exception as e:
        logger.error("[READY CHECK FAILED]: %s", e)
        return JSONResponse(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            content=ResponseError(
                message="Системы не готовы к работе: Redis недоступен"
            ).model_dump(),
        )


PREFIX = "/api/v1"
app.include_router(auth.router, prefix=PREFIX)
app.include_router(stream.router)
