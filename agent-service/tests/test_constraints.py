"""约束层单元测试：禁用词硬过滤 + 容错 JSON 解析。

可直接跑：  python tests/test_constraints.py
或用 pytest：python -m pytest tests/
（纯 stdlib，不依赖 openai-agents / openai，能在无 key 环境跑。）
"""
from __future__ import annotations

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from app.constraints.banned_words import find_hard_violations  # noqa: E402
from app.jsonutil import extract_json  # noqa: E402


def test_hard_violations_catches_real_banned():
    assert "全网最低" in find_hard_violations("这家全网最低价", "dianping")
    assert "加微信" in find_hard_violations("加微信领券", "dianping")
    assert "领券" in find_hard_violations("加微信领券", "dianping")


def test_hard_violations_no_false_positive_on_diyi_ci():
    # 回归：系统人设“第一次来”不能被“第一”误伤（评审子代理发现的 P1）
    assert find_hard_violations("第一次来这家店，体验不错", "dianping") == []


def test_hard_violations_no_false_positive_on_common_words():
    assert find_hard_violations("最近和朋友来的，老板赠送了小菜", "dianping") == []


def test_platform_specific_medical_word():
    assert "祛痘" in find_hard_violations("这个能祛痘", "xiaohongshu")
    # 点评平台不带小红书的医疗词表
    assert find_hard_violations("这个能祛痘", "dianping") == []


def test_extract_json_plain():
    assert extract_json('{"content":"x","tags":["a"]}') == {"content": "x", "tags": ["a"]}


def test_extract_json_fenced():
    raw = '好的：\n```json\n{"content":"x","tags":[]}\n```'
    assert extract_json(raw) == {"content": "x", "tags": []}


def test_extract_json_with_trailing_text():
    raw = '{"score": 88, "grade": "A"} 以上是结果'
    assert extract_json(raw) == {"score": 88, "grade": "A"}


def test_extract_json_array():
    assert extract_json("前缀 [1, 2, 3]") == [1, 2, 3]


def test_extract_json_raises_on_garbage():
    try:
        extract_json("完全不是 json")
    except ValueError:
        return
    raise AssertionError("应抛 ValueError")


def _run_all() -> int:
    fns = [v for k, v in sorted(globals().items()) if k.startswith("test_") and callable(v)]
    failed = 0
    for fn in fns:
        try:
            fn()
            print(f"PASS  {fn.__name__}")
        except Exception as exc:  # noqa: BLE001
            failed += 1
            print(f"FAIL  {fn.__name__}: {exc!r}")
    print(f"\n{len(fns) - failed}/{len(fns)} passed")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(_run_all())
