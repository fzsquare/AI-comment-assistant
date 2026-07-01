"""LLM 客户端 + OpenAI Agents SDK 模型封装。

要点（任意 OpenAI 兼容端点接入）：
- 多数第三方端点只实现 /chat/completions，不实现 OpenAI 的 Responses API，
  所以用 OpenAIChatCompletionsModel（走 chat completions），不能用默认那条。
- 第三方端点没有 OpenAI 的 tracing，必须在导入 agents 前设置
  OPENAI_AGENTS_DISABLE_TRACING，否则 SDK 可能初始化 trace exporter。
"""
from __future__ import annotations

import os

# 必须在导入 agents 前设置；否则 SDK 可能先初始化 tracing exporter。
os.environ.setdefault("OPENAI_AGENTS_DISABLE_TRACING", "true")

from agents import OpenAIChatCompletionsModel
from openai import AsyncOpenAI, DefaultAsyncHttpxClient

from .config import settings

_llm_client = AsyncOpenAI(
    base_url=settings.base_url,
    api_key=settings.api_key or "placeholder",  # 启动期允许空，调用前再校验
    http_client=DefaultAsyncHttpxClient(trust_env=False),
)


def make_model() -> OpenAIChatCompletionsModel:
    """每个 agent 用一个模型实例，统一指向配置的 LLM 端点。"""
    return OpenAIChatCompletionsModel(
        model=settings.model,
        openai_client=_llm_client,
    )
