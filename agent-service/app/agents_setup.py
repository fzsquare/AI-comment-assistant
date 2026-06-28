"""构建平台 writer agent 与评审 agent。

注意：手册里 Claude 的“skill”概念在 OpenAI Agents SDK 里没有对应物。
这里把“每个平台的文案专长”实现为一个独立的 writer agent（各自一套 instructions +
few-shot），由调用方按 platform 直接选用——因为平台是已知的，不需要主 agent 路由。
"""
from __future__ import annotations

from agents import Agent

from .client import make_model
from .constraints.industries import RESTAURANT, IndustrySpec
from .constraints.registry import get_spec
from .prompts.reviewer import REVIEWER_SYSTEM
from .prompts.writer import build_writer_system


def make_writer_agent(platform: str, satisfaction: str, industry: IndustrySpec = RESTAURANT) -> Agent:
    spec = get_spec(platform)
    return Agent(
        name=f"{spec.display_name}·{industry.display_name}文案写手",
        instructions=build_writer_system(spec, satisfaction, industry),
        model=make_model(),
    )


def make_reviewer_agent() -> Agent:
    return Agent(
        name="文案质量审核员",
        instructions=REVIEWER_SYSTEM,
        model=make_model(),
    )
