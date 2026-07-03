"""组装评审 agent 的 prompt —— 编码约束手册第六部分（质量评分表 + 15 项自检）。"""
from __future__ import annotations

from ..config import settings
from ..constraints.humanizer import reviewer_humanizer_note
from ..constraints.industries import RESTAURANT, IndustrySpec, reviewer_industry_note
from ..constraints.platforms.base import PlatformSpec

REVIEWER_SYSTEM = (
    "你是文案质量审核员，依据《多平台文案生成约束手册》第六部分给一条平台文案打分。"
    "严格、客观，宁可严一点。\n\n"
    "【内容质量评分表（100分制）】\n"
    "- 真实性 25：消费场景具体(10) + 细节丰富度(10) + 身份标签清晰(5)\n"
    "- 合规性 25：禁用词规避(10) + 平台规则匹配(10) + 商业标识/星级一致(5)\n"
    "- 拟人化 25：AI 痕迹少(10) + 口语化自然(10) + 情感自然且与满意度匹配(5)\n"
    "- 多样性 15：句式变化(10) + 表达不模板化(5)\n"
    "- 价值性 10：对他人参考价值(5) + 信息完整度(5)\n"
    "【等级】S=90-100 直接发布 / A=80-89 建议发布 / B=70-79 修改后发布 / "
    "C=60-69 建议重写 / D<60 禁止发布。\n\n"
    "【发布前自检（命中越多分越高）】具体消费时间 / 同行人员 / 至少2个具体项目或产品 / "
    "至少1个合理缺点 / 避开所有高风险禁用词 / 字数符合平台 / 段落结构清晰 / 语气自然口语化 / "
    "有互动引导（小红书/抖音）。不得因为正文未出现店名或人均花费扣分。\n\n"
    "【真实底线核查（命中即大幅扣分，grade 不得高于 C，pass=false）】\n"
    "- 文案出现店名、分店或品牌名（普通用户评论不要自报店铺名）；\n"
    "- 文案出现人均、总价、客单价或具体消费金额；\n"
    "- 文案编造了具体的路名/门牌/地标/分店位置；\n"
    "- 文案出现“允许范围”之外编造的具体项目/菜品/款式。\n"
    "注意：城市级的身份表述（如“新上海人/北漂”）属轻微问题，只在 issues 里提醒、"
    "小幅扣分即可，不计入上面的硬扣项。\n\n"
    + reviewer_humanizer_note()
    + "\n\n"
    "【输出格式（严格）】只返回一个 JSON 对象，不要 markdown、不要解释：\n"
    '{"score": 0-100, "grade": "S/A/B/C/D", "pass": true/false, "issues": ["可操作的修改建议", ...]}\n'
    "issues 针对扣分点给出具体、可执行的修改建议（如“删掉店名或具体花费，改成自然体验描述”）。"
)


def build_reviewer_user(
    spec: PlatformSpec,
    satisfaction: str,
    content: str,
    store_name: str,
    keywords: list[str],
    industry: IndustrySpec = RESTAURANT,
) -> str:
    item = industry.item_word
    kw = "、".join(keywords) if keywords else f"（未提供具体{item}，文案不应出现任何具体{item}）"
    return (
        "门店真实信息（用于核查是否编造）：\n"
        f"- 店名（仅用于核查，正文不应出现）：{store_name}\n"
        f"- 允许出现的{item}/关键词（不得超出）：{kw}\n"
        f"- {reviewer_industry_note(industry)}\n\n"
        f"平台：{spec.display_name}\n"
        f"满意度要求：{satisfaction}\n"
        f"字数要求：{spec.total_min_chars}-{spec.total_max_chars} 字\n"
        f"达标线：score >= {settings.min_pass_score}（达标则 pass=true）\n\n"
        "请审核下面这条文案并打分：\n"
        f"<<<\n{content}\n>>>"
    )
