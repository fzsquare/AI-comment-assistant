"""运行配置。所有值可由环境变量覆盖（见 .env.example）。"""
from __future__ import annotations

import os
from dataclasses import dataclass

from dotenv import load_dotenv

load_dotenv()


@dataclass(frozen=True)
class Settings:
    # --- LLM（任意 OpenAI 兼容端点：GPT 代理 / DeepSeek / 其它）---
    api_key: str = os.getenv("LLM_API_KEY", "")
    base_url: str = os.getenv("LLM_BASE_URL", "https://api.openai.com/v1")
    # 用支持 chat/completions + JSON 输出的对话模型（如 gpt-5.4、deepseek-chat）。
    # 不要用纯推理模型（如 deepseek-reasoner/R1）——不支持 JSON 模式，评审循环用不了。
    model: str = os.getenv("LLM_MODEL", "gpt-5.4")

    # --- 质量门槛（对应约束手册 6.1 评分等级）---
    # S=90-100 直接发布 / A=80-89 建议发布 / B=70-79 修改后发布 / C=60-69 重写 / D<60 禁止
    min_pass_score: int = int(os.getenv("MIN_PASS_SCORE", "80"))
    max_revise_rounds: int = int(os.getenv("MAX_REVISE_ROUNDS", "2"))

    # --- 批量生成 ---
    max_concurrency: int = int(os.getenv("MAX_CONCURRENCY", "5"))

    # --- 服务 ---
    host: str = os.getenv("AGENT_HOST", "0.0.0.0")
    port: int = int(os.getenv("AGENT_PORT", "8090"))

    def require_key(self) -> None:
        if not self.api_key:
            raise RuntimeError(
                "未配置 LLM_API_KEY。复制 .env.example 为 .env 并填入端点 key。"
            )


settings = Settings()
