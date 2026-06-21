"""LLM 客户端 + OpenAI Agents SDK 模型封装。

要点（任意 OpenAI 兼容端点接入）：
- 多数第三方端点只实现 /chat/completions，不实现 OpenAI 的 Responses API，
  所以用 OpenAIChatCompletionsModel（走 chat completions），不能用默认那条。
- 第三方端点没有 OpenAI 的 tracing，必须 set_tracing_disabled(True)，
  否则 SDK 会尝试把 trace 传回 OpenAI 并报缺 key。
"""
from __future__ import annotations

from agents import OpenAIChatCompletionsModel, set_tracing_disabled
from openai import AsyncOpenAI

from .config import settings

# 关闭 tracing —— 第三方 key 用不了 OpenAI 的 trace 上报
set_tracing_disabled(True)

_llm_client = AsyncOpenAI(
    base_url=settings.base_url,
    api_key=settings.api_key or "placeholder",  # 启动期允许空，调用前再校验
)


def make_model() -> OpenAIChatCompletionsModel:
    """每个 agent 用一个模型实例，统一指向配置的 LLM 端点。"""
    return OpenAIChatCompletionsModel(
        model=settings.model,
        openai_client=_llm_client,
    )
