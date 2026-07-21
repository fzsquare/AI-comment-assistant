"""约束层单元测试：禁用词硬过滤 + 容错 JSON 解析。

可直接跑：  python tests/test_constraints.py
或用 pytest：python -m pytest tests/
（纯 stdlib，不依赖 openai-agents / openai，能在无 key 环境跑。）
"""
from __future__ import annotations

import asyncio
import os
import sys
import types

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from app.constraints.banned_words import find_hard_violations  # noqa: E402
from app.constraints.humanizer import find_ai_tells  # noqa: E402
from app.constraints.industries import (  # noqa: E402
    find_cross_industry_leak,
    match_industry,
)
from app.constraints.registry import get_spec  # noqa: E402
from app.content_normalizer import normalize_generated_content  # noqa: E402
from app.config import Settings, load_settings  # noqa: E402
from app.internal_auth import check_internal_token  # noqa: E402
from app.jsonutil import extract_json  # noqa: E402
from app.prompts.reviewer import REVIEWER_SYSTEM, build_reviewer_user  # noqa: E402
from app.prompts.writer import build_writer_system, build_writer_user  # noqa: E402
from app.reviewer_logic import reviewer_passes  # noqa: E402

try:
    from app.schemas import FeedbackExamples, GenerateRequest, GenerationPreferences, ReviewItem, StoreContext  # noqa: E402
except ModuleNotFoundError as exc:
    if exc.name != "pydantic":
        raise
    FeedbackExamples = None
    GenerateRequest = None
    GenerationPreferences = None
    ReviewItem = None
    StoreContext = None


def test_hard_violations_catches_real_banned():
    assert "全网最低" in find_hard_violations("这家全网最低价", "dianping")
    assert "加微信" in find_hard_violations("加微信领券", "dianping")
    assert "领券" in find_hard_violations("加微信领券", "dianping")


def test_hard_violations_no_false_positive_on_diyi_ci():
    # 回归：系统人设“第一次来”不能被“第一”误伤（评审子代理发现的 P1）
    assert find_hard_violations("第一次来这家店，体验不错", "dianping") == []


def test_ai_tells_catches_cliche_and_dash():
    assert "总而言之" in find_ai_tells("菜不错，总而言之很满意")
    assert "回味无穷" in find_ai_tells("那道鱼回味无穷")
    assert "破折号——" in find_ai_tells("环境很好——尤其是灯光")


def test_ai_tells_no_false_positive_on_plain_review():
    # 真人随手写、无套话无破折号 → 不该误报
    assert find_ai_tells("上周三和朋友来的，点了酸菜鱼，鱼挺嫩，就是上菜有点慢。") == []


def test_match_industry_routes_store_types():
    assert match_industry("足疗按摩").code == "footmassage"
    assert match_industry("美发沙龙").code == "hairsalon"
    assert match_industry("美甲店").code == "nailsalon"
    assert match_industry("川菜/餐饮").code == "restaurant"
    assert match_industry("").code == "restaurant"  # 未填默认餐饮
    # 各行业的 item_word 已正确区分
    assert match_industry("美甲").item_word != match_industry("川菜").item_word


def test_match_industry_extended_types():
    assert match_industry("健身房").code == "fitness"
    assert match_industry("KTV").code == "entertainment"
    assert match_industry("剧本杀").code == "entertainment"
    assert match_industry("宠物美容").code == "pet"
    assert match_industry("洗车养护").code == "auto"
    assert match_industry("皮肤管理").code == "beauty"


def test_cross_industry_leak_detection():
    nail = match_industry("美甲店")
    restaurant = match_industry("餐饮")
    foot = match_industry("足疗")
    # 美甲文案里混进餐饮/足疗标志词 → 判定串味
    assert "上菜" in find_cross_industry_leak("做完美甲顺便上菜了", nail)
    assert "推拿" in find_cross_industry_leak("美甲师还帮我推拿了肩颈", nail)
    # 干净的美甲文案 → 不串味
    assert find_cross_industry_leak("卸甲很轻，光疗做得扎实，款式好看", nail) == []
    # 餐饮文案里混进美甲标志词 → 判定串味
    assert "美甲师" in find_cross_industry_leak("菜不错，美甲师手法也好", restaurant)
    # 足疗自身词不算串味
    assert find_cross_industry_leak("技师推拿到位，采耳也舒服", foot) == []


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


def test_dianping_prompt_allows_brief_natural_reviews_without_forced_three_part_structure():
    spec = get_spec("meituan")
    writer_prompt = build_writer_system(spec, "比较满意")
    reviewer_prompt = REVIEWER_SYSTEM

    assert spec.total_min_chars <= 60
    assert "三段式结构（强制）" not in writer_prompt
    assert "不强制三段式" in writer_prompt
    assert "简约短评" in writer_prompt
    assert "流水账" in writer_prompt
    assert "不得因为不是三段式" in reviewer_prompt


def test_non_xiaohongshu_generated_content_strips_explicit_title():
    content = "标题：适合朋友聚餐的小店\n\n上周和朋友过去吃饭，环境挺舒服，服务也比较自然。"
    assert normalize_generated_content("dianping", content) == "上周和朋友过去吃饭，环境挺舒服，服务也比较自然。"


def test_non_xiaohongshu_generated_content_keeps_normal_sentence_with_title_word():
    content = "标题取得普通一点反而真实，上周和朋友过去吃饭，环境挺舒服，服务也比较自然。"
    assert normalize_generated_content("dianping", content) == content


def test_xiaohongshu_generated_content_keeps_title_without_label():
    content = "标题：周末挖到宝藏小店\n\n上周和朋友过去吃饭，环境挺舒服，服务也比较自然。"
    assert normalize_generated_content("xiaohongshu", content) == "周末挖到宝藏小店\n\n上周和朋友过去吃饭，环境挺舒服，服务也比较自然。"


def test_xiaohongshu_generated_content_can_prepend_json_title():
    assert (
        normalize_generated_content(
            "xiaohongshu",
            "上周和朋友过去吃饭，环境挺舒服，服务也比较自然。",
            title="周末挖到宝藏小店",
        )
        == "周末挖到宝藏小店\n\n上周和朋友过去吃饭，环境挺舒服，服务也比较自然。"
    )


def test_natural_review_violations_flag_store_name_and_per_capita_spend():
    from app import content_normalizer as normalizer

    violations = normalizer.find_natural_review_violations(
        "七欣天香辣蟹这家店味道不错，人均80左右，朋友聚餐挺方便。",
        "七欣天香辣蟹",
    )

    joined = "；".join(violations)
    assert "店名" in joined
    assert "人均" in joined
    assert normalizer.find_natural_review_violations(
        "上周和朋友过去吃饭，蟹肉挺入味，服务员换盘也主动。",
        "七欣天香辣蟹",
    ) == []


def test_writer_and_reviewer_prompts_do_not_request_store_name_or_spend():
    if StoreContext is None:
        print("SKIP  pydantic 未安装，跳过自然评论 prompt 测试")
        return

    spec = get_spec("meituan")
    writer_prompt = build_writer_system(spec, "比较满意") + "\n" + build_writer_user(
        spec,
        StoreContext(store_name="七欣天香辣蟹", industry_type="餐饮"),
        ["香辣蟹", "服务热情"],
        "比较满意",
        0,
    )
    reviewer_prompt = REVIEWER_SYSTEM + "\n" + build_reviewer_user(
        spec,
        "比较满意",
        "上周和朋友过去吃饭，蟹肉挺入味，服务员换盘也主动。",
        "七欣天香辣蟹",
        ["香辣蟹", "服务热情"],
    )

    assert "正文不要出现店名" in writer_prompt
    assert "不要写人均" in writer_prompt
    assert "必须原样使用给定店名" not in writer_prompt
    assert "价格参考" not in writer_prompt
    assert "缺少具体价格" not in reviewer_prompt
    assert "不得因为正文未出现店名或人均花费扣分" in reviewer_prompt


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
        "AGENT_GENERATION_TIMEOUT_SECONDS": "0",
    }
    try:
        load_settings(bad_env)
    except RuntimeError as exc:
        msg = str(exc)
        assert "MIN_PASS_SCORE" in msg
        assert "MAX_REVISE_ROUNDS" in msg
        assert "MAX_CONCURRENCY" in msg
        assert "AGENT_GENERATION_TIMEOUT_SECONDS" in msg
        return
    raise AssertionError("应抛 RuntimeError")


def test_load_settings_accepts_generation_timeout():
    settings = load_settings({"AGENT_GENERATION_TIMEOUT_SECONDS": "240"})
    assert settings.generation_timeout_seconds == 240


def test_load_settings_defaults_to_conservative_concurrency():
    settings = load_settings({})
    assert settings.max_concurrency == 2


def test_pipeline_returns_completed_items_before_soft_timeout():
    if ReviewItem is None:
        print("SKIP  pydantic 未安装，跳过 pipeline partial timeout 测试")
        return
    try:
        import app.pipeline as pipeline
    except ModuleNotFoundError as exc:
        if exc.name in {"agents", "openai"}:
            print("SKIP  openai-agents 未安装，跳过 pipeline partial timeout 测试")
            return
        raise

    req = GenerateRequest(
        store={"store_name": "测试店"},
        keywords=[],
        platform="dianping",
        count=2,
    )
    previous_settings = pipeline.settings
    previous_make_writer = pipeline.make_writer_agent
    previous_make_reviewer = pipeline.make_reviewer_agent
    previous_generate_one = pipeline._generate_one

    class FakeSettings:
        max_concurrency = 2
        generation_timeout_seconds = 1

        def require_key(self):
            return None

    async def fake_generate_one(_writer, _reviewer, _spec, _req, index, _industry):
        if index == 0:
            return ReviewItem(content="服务自然，菜也稳定。", tags=[], score=80, grade="A")
        await asyncio.sleep(5)

    pipeline.settings = FakeSettings()
    pipeline.make_writer_agent = lambda *_args, **_kwargs: object()
    pipeline.make_reviewer_agent = lambda *_args, **_kwargs: object()
    pipeline._generate_one = fake_generate_one
    try:
        result = asyncio.run(pipeline.generate(req))
        assert result.produced == 1
        assert result.items[0].content == "服务自然，菜也稳定。"
    finally:
        pipeline.settings = previous_settings
        pipeline.make_writer_agent = previous_make_writer
        pipeline.make_reviewer_agent = previous_make_reviewer
        pipeline._generate_one = previous_generate_one


def test_generate_reviews_returns_504_when_generation_times_out():
    if GenerateRequest is None:
        print("SKIP  pydantic 未安装，跳过 agent timeout 路由测试")
        return
    try:
        from fastapi.testclient import TestClient
        from app.main import app, settings as app_settings
    except ModuleNotFoundError as exc:
        if exc.name in {"fastapi", "httpx", "starlette"}:
            print("SKIP  FastAPI 测试依赖未安装，跳过 agent timeout 路由测试")
            return
        raise

    async def slow_generate(_req):
        await asyncio.sleep(0.05)

    fake_pipeline = types.ModuleType("app.pipeline")
    fake_pipeline.generate = slow_generate
    previous_pipeline = sys.modules.get("app.pipeline")
    previous_token = app_settings.internal_token
    previous_timeout = app_settings.generation_timeout_seconds
    sys.modules["app.pipeline"] = fake_pipeline
    app_settings.internal_token = "expected-token"
    app_settings.generation_timeout_seconds = 0.001
    try:
        client = TestClient(app)
        resp = client.post(
            "/generate-reviews",
            headers={"X-Agent-Internal-Token": "expected-token"},
            json={
                "store": {"store_name": "测试店"},
                "keywords": [],
                "platform": "dianping",
                "count": 1,
            },
        )
        assert resp.status_code == 504
        assert "生成超时" in resp.text
    finally:
        app_settings.internal_token = previous_token
        app_settings.generation_timeout_seconds = previous_timeout
        if previous_pipeline is None:
            sys.modules.pop("app.pipeline", None)
        else:
            sys.modules["app.pipeline"] = previous_pipeline


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


def test_writer_prompt_uses_feedback_examples():
    if FeedbackExamples is None:
        print("SKIP  pydantic 未安装，跳过 feedback prompt 测试")
        return
    spec = get_spec("meituan")
    user = build_writer_user(
        spec,
        StoreContext(store_name="七欣天香辣蟹"),
        ["香辣蟹"],
        "比较满意",
        0,
        feedback=FeedbackExamples(
            accepted=["蟹肉饱满，服务员会主动换盘。"],
            rejected=["太像广告，夸得太满。"],
        ),
    )
    assert "用户喜欢的评论样本" in user
    assert "蟹肉饱满，服务员会主动换盘。" in user
    assert "用户不喜欢的评论样本" in user
    assert "太像广告，夸得太满。" in user


def test_generate_request_accepts_generation_preferences():
    if GenerateRequest is None:
        print("SKIP  pydantic 未安装，跳过 generation_preferences schema 测试")
        return
    req = GenerateRequest(
        store={"store_name": "七欣天香辣蟹"},
        keywords=["香辣蟹"],
        platform="meituan",
        generation_preferences={
            "focus_keywords": ["香辣蟹", "服务热情"],
            "style_codes": ["natural", "detail_rich"],
            "diversity_dimensions": ["customer_identity", "content_angle"],
            "reference_reviews": ["蟹很入味，服务员会主动帮忙换盘。"],
            "length_variance": "wide",
        },
    )
    assert req.generation_preferences.focus_keywords == ["香辣蟹", "服务热情"]
    assert req.generation_preferences.style_codes == ["natural", "detail_rich"]
    assert req.generation_preferences.diversity_dimensions == ["customer_identity", "content_angle"]
    assert req.generation_preferences.length_variance == "wide"


def test_writer_prompt_uses_generation_preferences():
    if GenerationPreferences is None:
        print("SKIP  pydantic 未安装，跳过 generation_preferences prompt 测试")
        return
    spec = get_spec("meituan")
    user = build_writer_user(
        spec,
        StoreContext(store_name="七欣天香辣蟹"),
        ["香辣蟹", "聚餐"],
        "比较满意",
        2,
        generation_preferences=GenerationPreferences(
            focus_keywords=["香辣蟹", "服务热情"],
            style_codes=["natural", "detail_rich"],
            diversity_dimensions=["customer_identity"],
            reference_reviews=["蟹很入味，服务员会主动帮忙换盘。"],
            length_variance="wide",
        ),
    )
    assert "商家生成方向" in user
    assert "香辣蟹、服务热情" in user
    assert "自然随手写、细节丰富" in user
    assert "本条多样化视角" in user
    assert "顾客身份：附近上班族" in user
    assert "蟹很入味，服务员会主动帮忙换盘。" in user
    assert "本条字数目标" in user


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
