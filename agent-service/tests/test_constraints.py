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
from app.constraints.registry import get_spec  # noqa: E402
from app.content_normalizer import normalize_generated_content  # noqa: E402
from app.config import Settings, load_settings  # noqa: E402
from app.internal_auth import check_internal_token  # noqa: E402
from app.jsonutil import extract_json  # noqa: E402
from app.reviewer_logic import reviewer_passes  # noqa: E402

try:
    from app.schemas import GenerateRequest, ReviewItem  # noqa: E402
except ModuleNotFoundError as exc:
    if exc.name != "pydantic":
        raise
    GenerateRequest = None
    ReviewItem = None


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


def test_meituan_platform_uses_dianping_constraints():
    if GenerateRequest is None:
        return
    req = GenerateRequest(
        store={"store_name": "示例餐厅"},
        keywords=[],
        platform="meituan",
        count=1,
    )
    assert req.platform == "meituan"
    assert get_spec("meituan").display_name == "大众点评"


def test_non_xiaohongshu_generated_content_strips_explicit_title():
    content = "标题：适合朋友聚餐的小店\n\n上周和朋友过去吃饭，环境挺舒服，服务也比较自然。"
    assert normalize_generated_content("dianping", content) == "上周和朋友过去吃饭，环境挺舒服，服务也比较自然。"


def test_non_xiaohongshu_generated_content_keeps_normal_sentence_with_title_word():
    content = "标题取得普通一点反而真实，上周和朋友过去吃饭，环境挺舒服，服务也比较自然。"
    assert normalize_generated_content("dianping", content) == content


def test_xiaohongshu_generated_content_keeps_title_without_label():
    content = "标题：人均70挖到宝藏小店\n\n上周和朋友过去吃饭，环境挺舒服，服务也比较自然。"
    assert normalize_generated_content("xiaohongshu", content) == "人均70挖到宝藏小店\n\n上周和朋友过去吃饭，环境挺舒服，服务也比较自然。"


def test_xiaohongshu_generated_content_can_prepend_json_title():
    assert (
        normalize_generated_content(
            "xiaohongshu",
            "上周和朋友过去吃饭，环境挺舒服，服务也比较自然。",
            title="人均70挖到宝藏小店",
        )
        == "人均70挖到宝藏小店\n\n上周和朋友过去吃饭，环境挺舒服，服务也比较自然。"
    )


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


def test_internal_token_missing_configuration_returns_503():
    settings = Settings(internal_token="")
    ok, status, detail = check_internal_token("anything", settings)
    assert ok is False
    assert status == 503
    assert "AGENT_INTERNAL_TOKEN" in detail


def test_internal_token_rejects_missing_or_wrong_header():
    settings = Settings(internal_token="expected-token")
    assert check_internal_token(None, settings)[:2] == (False, 401)
    assert check_internal_token("wrong-token", settings)[:2] == (False, 401)


def test_internal_token_accepts_exact_match():
    settings = Settings(internal_token="expected-token")
    assert check_internal_token("expected-token", settings) == (True, 200, "ok")


def test_load_settings_defaults_to_loopback_host():
    settings = load_settings({})
    assert settings.host == "127.0.0.1"


def test_load_settings_rejects_invalid_ranges_with_context():
    bad_env = {
        "MIN_PASS_SCORE": "101",
        "MAX_REVISE_ROUNDS": "-1",
        "MAX_CONCURRENCY": "0",
    }
    try:
        load_settings(bad_env)
    except RuntimeError as exc:
        msg = str(exc)
        assert "MIN_PASS_SCORE" in msg
        assert "MAX_REVISE_ROUNDS" in msg
        assert "MAX_CONCURRENCY" in msg
        return
    raise AssertionError("应抛 RuntimeError")


def test_reviewer_pass_string_false_is_false_even_with_good_score():
    assert reviewer_passes("false", 95, 80) is False


def test_reviewer_pass_true_requires_minimum_score():
    assert reviewer_passes(True, 79, 80) is False
    assert reviewer_passes("true", 80, 80) is True


def test_reviewer_pass_missing_uses_score_threshold():
    assert reviewer_passes(None, 79, 80) is False
    assert reviewer_passes(None, 80, 80) is True


def test_generate_request_rejects_oversized_keywords():
    if GenerateRequest is None:
        print("SKIP  pydantic 未安装，跳过 schema 边界测试")
        return
    try:
        GenerateRequest(
            store={"store_name": "测试店"},
            keywords=["标签"] * 21,
        )
    except Exception:
        return
    raise AssertionError("应拒绝过多 keywords")


def test_generate_request_rejects_long_store_name():
    if GenerateRequest is None:
        print("SKIP  pydantic 未安装，跳过 schema 边界测试")
        return
    try:
        GenerateRequest(
            store={"store_name": "店" * 121},
            keywords=[],
        )
    except Exception:
        return
    raise AssertionError("应拒绝过长 store_name")


def test_review_item_rejects_invalid_score_and_grade():
    if ReviewItem is None:
        print("SKIP  pydantic 未安装，跳过 schema 边界测试")
        return
    for kwargs in (
        {"content": "内容", "score": 101, "grade": "A"},
        {"content": "内容", "score": 90, "grade": "Z"},
    ):
        try:
            ReviewItem(**kwargs)
        except Exception:
            continue
        raise AssertionError(f"应拒绝非法输出边界：{kwargs!r}")


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
