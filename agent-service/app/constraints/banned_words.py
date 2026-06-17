"""第五部分：禁用词清单。

设计取舍：
- HARD_BANNED：无歧义的多字短语/词，命中即判为高风险 → 触发重写。
- SOFT_ABSOLUTE：单字或易误伤的绝对化词（最/送/绝对…），只写进 writer 指令
  + 评审“合规性”扣分，不做硬过滤，避免误伤“最近/赠送”等正常表达。
- MID_RISK_REPLACEMENTS：中风险词 → 安全替换，写进 writer 指令引导模型规避。
"""
from __future__ import annotations

from typing import Dict, List

# --- 高风险：无歧义短语，硬过滤（命中即重写）---
HARD_BANNED: List[str] = [
    # 绝对化（多字、无歧义）
    # 注意：不要把“第一”放进硬名单——会误伤系统自己注入的人设“第一次来”。
    # “第一/最”这类作最高级时的绝对词交给 SOFT_ABSOLUTE + 评审合规分处理。
    "顶级", "国家级", "世界级", "独家", "万能", "空前", "绝后",
    "巅峰", "极致", "完美", "无敌", "天花板",
    # 功效承诺
    "根治", "痊愈", "100%有效", "永不反弹", "一天见效", "立刻见效",
    "包治", "包好", "无效退款", "guaranteed",
    # 价格误导
    "历史最低", "全网最低", "0元", "白给", "亏本", "跳楼价", "清仓价", "赔本卖",
    # 导流
    "加微信", "私信", "联系方式", "购买链接", "淘口令", "二维码",
    "手机号", "扫码", "vx", "VX",
    # 平台名称 / 电商动作
    "淘宝", "拼多多", "京东", "天猫", "秒杀", "抢购", "领券", "优惠券",
    # 刷评敏感
    "霸王餐", "刷单", "好评返现", "免费体验", "商家赠送", "探店邀请",
    # 网络热词（限流）
    "绝绝子", "yyds", "YYDS", "闭眼入", "手慢无", "无限回购",
]

# --- 软约束：单字/易误伤绝对词，只进指令 + 评审扣分，不硬过滤 ---
SOFT_ABSOLUTE: List[str] = ["最", "第一", "永久", "送", "免费", "绝对", "必须", "必买", "必吃", "首选"]

# --- 中风险 → 安全替换（写进 writer 指令）---
MID_RISK_REPLACEMENTS: Dict[str, str] = {
    "超": "挺", "巨": "比较", "特别": "挺", "非常": "比较",
    "绝对": "建议", "肯定": "可以", "必须": "值得",
    "推荐": "分享", "安利": "分享", "种草": "体验",
    "必买": "值得试试", "必吃": "个人喜欢", "首选": "个人喜欢",
    "亲测": "用了一段时间", "无限回购": "会考虑再买",
    "闭眼入": "值得试试", "手慢无": "可以试试",
}

# --- 安全可用词（供 writer 参考）---
SAFE_WORDS: Dict[str, List[str]] = {
    "评价词": ["不错", "可以", "还行", "值得", "适合", "喜欢", "满意", "挺好的", "蛮好的"],
    "行为词": ["分享", "自用", "朋友推荐", "用了一段时间", "个人体验", "会再来", "试试看"],
    "称呼词": ["姐妹", "宝子", "家人们", "友友们", "大家", "朋友们"],
    "语气词": ["嘛", "哈", "啦", "呗", "咯", "呀", "哦", "呢", "吧"],
}

# --- 平台特定禁用（餐饮场景下医疗词基本不会出现，仍保留以防万一）---
PLATFORM_BANNED: Dict[str, List[str]] = {
    "xiaohongshu": [
        "治疗", "药用", "消炎", "止痛", "减肥", "瘦身", "祛痘", "祛斑",
        "生发", "防脱", "抗过敏", "医用", "处方",
    ],
    "dianping": [],
    "douyin": [],
}


def find_hard_violations(text: str, platform: str) -> List[str]:
    """返回文本中命中的高风险词（含平台特定）。空列表 = 通过硬过滤。"""
    low = text.lower()
    hits: List[str] = []
    for w in HARD_BANNED:
        if w.lower() in low:
            hits.append(w)
    for w in PLATFORM_BANNED.get(platform, []):
        if w in text:
            hits.append(w)
    return hits


def banned_words_block(platform: str) -> str:
    """渲染进 writer system prompt 的禁用词约束段。"""
    hard = "、".join(HARD_BANNED[:24]) + " 等"
    soft = "、".join(SOFT_ABSOLUTE)
    repl = "；".join(f"{k}→{v}" for k, v in list(MID_RISK_REPLACEMENTS.items())[:10])
    extra = PLATFORM_BANNED.get(platform, [])
    extra_line = f"\n- 本平台额外禁用：{('、'.join(extra))}" if extra else ""
    return (
        "【禁用词约束（强制）】\n"
        f"- 绝对禁止出现：{hard}\n"
        f"- 避免绝对化单字：{soft}（不要用作最高级修饰，如“最好吃”请改为“挺好吃”）\n"
        f"- 中风险词请替换：{repl} 等\n"
        "- 禁止任何导流/电商/刷评信息（微信、电话、二维码、各类电商平台名等）"
        f"{extra_line}"
    )
