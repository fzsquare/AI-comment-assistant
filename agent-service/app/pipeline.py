"""文案生成主流程：选专家 → 生成 → 硬过滤 + 评审打分 → 不达标重写循环 → 批量并发。

自评循环用代码编排（不靠模型自己调工具），对第三方端点更确定、更稳。
"""
from __future__ import annotations

import asyncio
import logging
from typing import List, Tuple

from agents import Agent, Runner

from .agents_setup import make_reviewer_agent, make_writer_agent
from .config import settings
from .content_normalizer import normalize_generated_content
from .constraints.banned_words import find_hard_violations
from .constraints.platforms.base import PlatformSpec
from .constraints.registry import get_spec
from .jsonutil import extract_json
from .prompts.reviewer import build_reviewer_user
from .prompts.writer import build_revise_user, build_writer_user
from .reviewer_logic import clamp_score, reviewer_passes
from .schemas import GenerateRequest, GenerateResponse, ReviewItem


def _grade_from_score(score: int) -> str:
    if score >= 90:
        return "S"
    if score >= 80:
        return "A"
    if score >= 70:
        return "B"
    if score >= 60:
        return "C"
    return "D"


async def _run_writer(agent: Agent, user_input: str, platform: str) -> Tuple[str, List[str]]:
    result = await Runner.run(agent, user_input)
    out = result.final_output or ""
    try:
        data = extract_json(out)
        title = str(data.get("title", "")).strip()
        content = str(data.get("content", "")).strip()
        tags = [str(t).strip() for t in (data.get("tags") or []) if str(t).strip()]
        content = normalize_generated_content(platform, content, title=title)
        if content:
            return content, tags
    except ValueError:
        pass
    # 解析失败兜底：把整段当正文，tags 留空
    return normalize_generated_content(platform, out), []


async def _run_reviewer(
    agent: Agent,
    spec: PlatformSpec,
    satisfaction: str,
    content: str,
    store_name: str,
    keywords: List[str],
) -> Tuple[int, str, bool, List[str]]:
    user = build_reviewer_user(spec, satisfaction, content, store_name, keywords)
    result = await Runner.run(agent, user)
    try:
        data = extract_json(result.final_output or "")
        score = clamp_score(data.get("score", 0))
        raw_grade = str(data.get("grade", "")).strip().upper()[:1]
        grade = raw_grade if raw_grade in {"S", "A", "B", "C", "D"} else _grade_from_score(score)
        passed = reviewer_passes(data.get("pass"), score, settings.min_pass_score)
        issues = [str(i).strip() for i in (data.get("issues") or []) if str(i).strip()]
        return score, grade, passed, issues
    except (ValueError, TypeError):
        return 0, "D", False, ["质量审核解析失败，请重写得更真实、细节更具体"]


async def _generate_one(
    writer: Agent, reviewer: Agent, spec: PlatformSpec, req: GenerateRequest, index: int
) -> ReviewItem:
    user = build_writer_user(spec, req.store, req.keywords, req.satisfaction, index)
    best: ReviewItem | None = None
    best_clean = False  # best 是否无硬违规

    for round_ in range(settings.max_revise_rounds + 1):
        content, tags = await _run_writer(writer, user, req.platform)
        score, grade, passed, issues = await _run_reviewer(
            reviewer, spec, req.satisfaction, content, req.store.store_name, req.keywords
        )

        # 硬过滤：命中高风险禁用词 → 直接判不合规，强制重写
        hits = find_hard_violations(content, req.platform)
        if hits:
            passed = False
            score = min(score, 59)
            grade = "D"
            issues = [f"命中高风险禁用词：{'、'.join(hits)}，必须删改"] + issues

        item = ReviewItem(
            content=content, tags=tags, score=score, grade=grade, revisions=round_
        )
        if passed:
            return item

        # 选 best：优先无违规，其次高分（避免兜底返回一条带禁用词的文案）
        clean = not hits
        if best is None or (clean, item.score) > (best_clean, best.score):
            best = item
            best_clean = clean

        user = build_revise_user(content, issues)

    assert best is not None
    return best


async def generate(req: GenerateRequest) -> GenerateResponse:
    settings.require_key()
    spec = get_spec(req.platform)
    writer = make_writer_agent(req.platform, req.satisfaction)
    reviewer = make_reviewer_agent()

    sem = asyncio.Semaphore(settings.max_concurrency)

    async def worker(i: int) -> ReviewItem:
        async with sem:
            return await _generate_one(writer, reviewer, spec, req, i)

    results = await asyncio.gather(
        *[worker(i) for i in range(req.count)], return_exceptions=True
    )
    items: List[ReviewItem] = []
    for i, r in enumerate(results):
        if isinstance(r, ReviewItem):
            items.append(r)
        else:
            # 单条失败不影响整批；记录原因便于排查（填池可由 Go 侧后续补量）
            logging.warning("第 %d 条生成失败：%r", i, r)
    if not items:
        raise RuntimeError("本批次未生成任何评价，请检查模型服务或输入约束。")
    return GenerateResponse(
        platform=req.platform,
        requested=req.count,
        produced=len(items),
        items=items,
    )
