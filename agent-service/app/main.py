"""FastAPI 入口。Go 后端通过 POST /generate-reviews 调用本服务填充评价池。"""
from __future__ import annotations

import logging
import time
from typing import Annotated

from fastapi import FastAPI, Header, HTTPException

from .config import settings
from .internal_auth import check_internal_token
from .schemas import GenerateRequest, GenerateResponse

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
logger = logging.getLogger("agent-service")
app = FastAPI(title="多平台文案生成 Agent 服务", version="0.1.0")


@app.get("/health")
async def health() -> dict:
    return {"status": "ok"}


@app.post("/generate-reviews", response_model=GenerateResponse)
async def generate_reviews(
    req: GenerateRequest,
    x_agent_internal_token: Annotated[
        str | None, Header(alias="X-Agent-Internal-Token")
    ] = None,
    x_generation_task_id: Annotated[
        str | None, Header(alias="X-Generation-Task-ID")
    ] = None,
) -> GenerateResponse:
    started = time.perf_counter()
    ok, status_code, detail = check_internal_token(x_agent_internal_token, settings)
    if not ok:
        logger.warning(
            "agent_generation_auth_failed task_id=%s status=%s detail=%s",
            x_generation_task_id or "",
            status_code,
            detail,
        )
        raise HTTPException(status_code=status_code, detail=detail)
    try:
        from .pipeline import generate

        logger.info(
            "agent_generation_start task_id=%s platform=%s count=%s",
            x_generation_task_id or "",
            req.platform,
            req.count,
        )
        result = await generate(req)
        duration_ms = int((time.perf_counter() - started) * 1000)
        logger.info(
            "agent_generation_success task_id=%s platform=%s requested=%s produced=%s duration_ms=%s",
            x_generation_task_id or "",
            req.platform,
            req.count,
            result.produced,
            duration_ms,
        )
        return result
    except RuntimeError as exc:  # 未配置 key 等
        duration_ms = int((time.perf_counter() - started) * 1000)
        logger.exception(
            "agent_generation_runtime_error task_id=%s platform=%s count=%s duration_ms=%s",
            x_generation_task_id or "",
            req.platform,
            req.count,
            duration_ms,
        )
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    except ValueError as exc:  # 未知平台等
        duration_ms = int((time.perf_counter() - started) * 1000)
        logger.exception(
            "agent_generation_bad_request task_id=%s platform=%s count=%s duration_ms=%s",
            x_generation_task_id or "",
            req.platform,
            req.count,
            duration_ms,
        )
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    except Exception as exc:
        duration_ms = int((time.perf_counter() - started) * 1000)
        logger.exception(
            "agent_generation_unhandled_error task_id=%s platform=%s count=%s duration_ms=%s",
            x_generation_task_id or "",
            req.platform,
            req.count,
            duration_ms,
        )
        raise HTTPException(status_code=500, detail="agent-service 内部错误") from exc


def main() -> None:
    import uvicorn

    uvicorn.run("app.main:app", host=settings.host, port=settings.port, reload=False)


if __name__ == "__main__":
    main()
