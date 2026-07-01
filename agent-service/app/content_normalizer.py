"""Normalize generated review text before scoring and returning it."""
from __future__ import annotations

import re

TITLE_LABEL_RE = re.compile(
    r"^\s*(?:[#*\-\s]*)?(?:[【\[]\s*(?:标题|题目|评论标题|小标题|title)\s*[】\]]|"
    r"(?:标题|题目|评论标题|小标题|title)\s*[:：])\s*(?P<title>.*)$",
    re.IGNORECASE,
)


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
