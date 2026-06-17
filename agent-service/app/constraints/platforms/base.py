"""平台约束的数据结构。每个平台用一个 PlatformSpec 实例承载手册里的规则。"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import List


@dataclass(frozen=True)
class PlatformSpec:
    code: str
    display_name: str
    # 整篇字数范围（用于生成后校验长度）
    total_min_chars: int
    total_max_chars: int
    # 标签数量范围
    tag_min: int
    tag_max: int
    # 必备细节类型最少数量（手册 4.2）
    min_detail_count: int
    # 渲染进 writer system prompt 的平台结构/风格/标签约束（手册一/二/三部分）
    writer_rules: str
    # 2-3 条高质量范例（取自手册的 ✓示例 思路；few-shot 是质量最大的杠杆）
    few_shots: List[str] = field(default_factory=list)
