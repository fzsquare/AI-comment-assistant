"""内部调用认证。只给 Go backend 调用 agent-service 使用。"""
from __future__ import annotations

import hmac

from .config import Settings


def check_internal_token(
    provided_token: str | None, settings: Settings
) -> tuple[bool, int, str]:
    expected = settings.internal_token
    if not expected:
        return False, 503, "AGENT_INTERNAL_TOKEN 未配置"
    if not provided_token:
        return False, 401, "缺少内部认证 token"
    if not hmac.compare_digest(provided_token, expected):
        return False, 401, "内部认证 token 无效"
    return True, 200, "ok"
