"""运行配置。所有值可由环境变量覆盖（见 .env.example）。"""
from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Mapping

try:
    from dotenv import load_dotenv
except ModuleNotFoundError:  # 测试环境可不安装 python-dotenv，生产仍建议安装 requirements。
    def load_dotenv() -> None:
        return None

load_dotenv()


@dataclass(frozen=True)
class Settings:
    # --- LLM（任意 OpenAI 兼容端点：GPT 代理 / DeepSeek / 其它）---
    api_key: str = ""
    base_url: str = "https://api.openai.com/v1"
    # 用支持 chat/completions + JSON 输出的对话模型（如 gpt-5.4、deepseek-chat）。
    # 不要用纯推理模型（如 deepseek-reasoner/R1）——不支持 JSON 模式，评审循环用不了。
    model: str = "gpt-5.4"

    # --- 质量门槛（对应约束手册 6.1 评分等级）---
    # S=90-100 直接发布 / A=80-89 建议发布 / B=70-79 修改后发布 / C=60-69 重写 / D<60 禁止
    min_pass_score: int = 80
    max_revise_rounds: int = 2

    # --- 批量生成 ---
    max_concurrency: int = 5

    # --- 服务 ---
    host: str = "127.0.0.1"
    port: int = 8090
    internal_token: str = ""

    def require_key(self) -> None:
        if not self.api_key:
            raise RuntimeError(
                "未配置 LLM_API_KEY。复制 .env.example 为 .env 并填入端点 key。"
            )


def _read(env: Mapping[str, str], name: str, default: str) -> str:
    return env.get(name, default)


def _parse_int(
    env: Mapping[str, str],
    name: str,
    default: int,
    errors: list[str],
    *,
    min_value: int | None = None,
    max_value: int | None = None,
) -> int:
    raw = _read(env, name, str(default))
    try:
        value = int(raw)
    except (TypeError, ValueError):
        errors.append(f"{name} 必须是整数，当前值为 {raw!r}")
        return default
    if min_value is not None and value < min_value:
        errors.append(f"{name} 必须 >= {min_value}，当前值为 {value}")
    if max_value is not None and value > max_value:
        errors.append(f"{name} 必须 <= {max_value}，当前值为 {value}")
    return value


def load_settings(environ: Mapping[str, str] | None = None) -> Settings:
    env = os.environ if environ is None else environ
    errors: list[str] = []
    min_pass_score = _parse_int(
        env, "MIN_PASS_SCORE", 80, errors, min_value=0, max_value=100
    )
    max_revise_rounds = _parse_int(
        env, "MAX_REVISE_ROUNDS", 2, errors, min_value=0
    )
    max_concurrency = _parse_int(env, "MAX_CONCURRENCY", 5, errors, min_value=1)
    port = _parse_int(env, "AGENT_PORT", 8090, errors, min_value=1, max_value=65535)
    if errors:
        raise RuntimeError("配置错误：" + "；".join(errors))
    return Settings(
        api_key=_read(env, "LLM_API_KEY", ""),
        base_url=_read(env, "LLM_BASE_URL", "https://api.openai.com/v1"),
        model=_read(env, "LLM_MODEL", "gpt-5.4"),
        min_pass_score=min_pass_score,
        max_revise_rounds=max_revise_rounds,
        max_concurrency=max_concurrency,
        host=_read(env, "AGENT_HOST", "127.0.0.1"),
        port=port,
        internal_token=_read(env, "AGENT_INTERNAL_TOKEN", ""),
    )


settings = load_settings()
