"""Reviewer 输出的服务端判定逻辑。"""
from __future__ import annotations


def clamp_score(value: object) -> int:
    try:
        score = int(value)
    except (TypeError, ValueError):
        return 0
    return max(0, min(100, score))


def reviewer_passes(raw_pass: object, score: int, min_pass_score: int) -> bool:
    score_passes = score >= min_pass_score
    if raw_pass is None:
        return score_passes
    if isinstance(raw_pass, bool):
        return raw_pass and score_passes
    if isinstance(raw_pass, str):
        normalized = raw_pass.strip().lower()
        if normalized in {"true", "yes", "1"}:
            return score_passes
        if normalized in {"false", "no", "0"}:
            return False
    return False
