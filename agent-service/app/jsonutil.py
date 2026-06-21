"""容错 JSON 解析。

多数第三方端点不支持严格 json_schema（只支持 json_object），且偶尔会包 ```json
代码块或在前后加说明文字。这里做防御式解析，不依赖严格 schema。
"""
from __future__ import annotations

import json
import re
from typing import Any


def extract_json(text: str) -> Any:
    if not text:
        raise ValueError("空响应")
    t = text.strip()
    # 去掉 ```json ... ``` 代码块围栏
    t = re.sub(r"^```(?:json)?\s*", "", t)
    t = re.sub(r"\s*```$", "", t).strip()

    try:
        return json.loads(t)
    except json.JSONDecodeError:
        pass

    # 退一步：截取第一个平衡的 {...} 或 [...]
    for open_ch, close_ch in (("{", "}"), ("[", "]")):
        start = t.find(open_ch)
        if start == -1:
            continue
        depth = 0
        for i in range(start, len(t)):
            if t[i] == open_ch:
                depth += 1
            elif t[i] == close_ch:
                depth -= 1
                if depth == 0:
                    try:
                        return json.loads(t[start : i + 1])
                    except json.JSONDecodeError:
                        break
    raise ValueError(f"无法从响应中解析 JSON：{text[:200]}")
