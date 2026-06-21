"""FastAPI 入口。Go 后端通过 POST /generate-reviews 调用本服务填充评价池。"""
from __future__ import annotations

from fastapi import FastAPI, HTTPException

from .config import settings
from .pipeline import generate
from .schemas import GenerateRequest, GenerateResponse

app = FastAPI(title="多平台文案生成 Agent 服务", version="0.1.0")


@app.get("/health")
async def health() -> dict:
    return {
        "status": "ok",
        "model": settings.model,
        "base_url": settings.base_url,
        "key_configured": bool(settings.api_key),
        "min_pass_score": settings.min_pass_score,
    }


@app.post("/generate-reviews", response_model=GenerateResponse)
async def generate_reviews(req: GenerateRequest) -> GenerateResponse:
    try:
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
