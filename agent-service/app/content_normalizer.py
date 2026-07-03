"""Normalize generated review text before scoring and returning it."""
from __future__ import annotations

import re

TITLE_LABEL_RE = re.compile(
    r"^\s*(?:[#*\-\s]*)?(?:[【\[]\s*(?:标题|题目|评论标题|小标题|title)\s*[】\]]|"
    r"(?:标题|题目|评论标题|小标题|title)\s*[:：])\s*(?P<title>.*)$",
    re.IGNORECASE,
)
SPEND_RE = re.compile(
    r"(?:人均|客单价|总价|消费|花了|花费)\s*(?:大概|差不多|不到|约|大约|在)?\s*"
    r"(?:[¥￥]?\s*\d+(?:\.\d+)?\s*(?:元|块|rmb|RMB)?|[一二三四五六七八九十百]+(?:元|块)?)",
    re.IGNORECASE,
)
MONEY_RE = re.compile(r"[¥￥]\s*\d+(?:\.\d+)?|\d+(?:\.\d+)?\s*(?:元|块|rmb|RMB)")


def _strip_title_label(line: str) -> tuple[bool, str]:
    matched = TITLE_LABEL_RE.match(line)
    if not matched:
        return False, line.strip()
    return True, matched.group("title").strip()


def normalize_generated_content(platform: str, content: str, title: str = "") -> str:
    """Keep titles only for Xiaohongshu; strip explicit title labels everywhere."""
    normalized = content.replace("\r\n", "\n").replace("\r", "\n").strip()
    title_matched, clean_title = _strip_title_label(title)
    if title_matched:
        title = clean_title
    else:
        title = title.strip()

    if platform == "xiaohongshu":
        lines = normalized.splitlines()
        if lines:
            matched, first_line = _strip_title_label(lines[0])
            if matched:
                lines[0] = first_line
                normalized = "\n".join(lines).strip()

        if title:
            first_non_empty = next((line.strip() for line in normalized.splitlines() if line.strip()), "")
            if first_non_empty != title:
                normalized = f"{title}\n\n{normalized}" if normalized else title
        return normalized.strip()

    lines = normalized.splitlines()
    while lines and not lines[0].strip():
        lines.pop(0)
    if lines:
        matched, _ = _strip_title_label(lines[0])
        if matched:
            lines = lines[1:]
    return "\n".join(lines).strip()


def find_natural_review_violations(content: str, store_name: str) -> list[str]:
    """Find content that makes generated text read unlike a normal user review."""
    text = content or ""
    violations: list[str] = []
    name = (store_name or "").strip()
    if name and name in text:
        violations.append(f"正文出现店名：{name}。正常用户评论不要写店铺名。")

    spend_hits = []
    for pattern in (SPEND_RE, MONEY_RE):
        for match in pattern.finditer(text):
            hit = match.group(0).strip()
            if hit and hit not in spend_hits:
                spend_hits.append(hit)
    if "人均" in text and not any("人均" in hit for hit in spend_hits):
        spend_hits.append("人均")
    if spend_hits:
        violations.append(
            "正文出现人均花费或具体消费金额："
            + "、".join(spend_hits[:4])
            + "。正常用户评论不要写人均、总价或具体花费。"
        )
    return violations
