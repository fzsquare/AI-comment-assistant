"""平台约束注册表：platform code → PlatformSpec。"""
from __future__ import annotations

from .platforms.base import PlatformSpec
from .platforms.dianping import DIANPING
from .platforms.douyin import DOUYIN
from .platforms.xiaohongshu import XIAOHONGSHU

_REGISTRY = {
    DIANPING.code: DIANPING,
    "meituan": DIANPING,
    XIAOHONGSHU.code: XIAOHONGSHU,
    DOUYIN.code: DOUYIN,
}


def get_spec(platform: str) -> PlatformSpec:
    spec = _REGISTRY.get(platform)
    if spec is None:
        raise ValueError(f"未知平台：{platform}，支持：{list(_REGISTRY)}")
    return spec
