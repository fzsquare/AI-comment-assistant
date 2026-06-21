"""FastAPI 入口。Go 后端通过 POST /generate-reviews 调用本服务填充评价池。"""
from __future__ import annotations

from typing import Annotated

from fastapi import FastAPI, Header, HTTPException

from .config import settings
from .internal_auth import check_internal_token
from .schemas import GenerateRequest, GenerateResponse

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
) -> GenerateResponse:
    ok, status_code, detail = check_internal_token(x_agent_internal_token, settings)
    if not ok:
        raise HTTPException(status_code=status_code, detail=detail)
    try:
        from .pipeline import generate

        return await generate(req)
    except RuntimeError as exc:  # 未配置 key 等
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    except ValueError as exc:  # 未知平台等
        raise HTTPException(status_code=400, detail=str(exc)) from exc


def main() -> None:
    import uvicorn

    uvicorn.run("app.main:app", host=settings.host, port=settings.port, reload=False)


if __name__ == "__main__":
    main()
