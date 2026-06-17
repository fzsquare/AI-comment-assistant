# 多平台文案生成 Agent 服务

独立的 Python 服务，用 **OpenAI Agents SDK + 任意 OpenAI 兼容端点**（当前接 GPT 代理，
也可换 DeepSeek 等），按《多平台文案生成约束手册》
为大众点评 / 小红书 / 抖音评论生成**优质、合规、拟人化**的真实评价文案。Go 后端通过
HTTP 调用本服务来填充评价池。

## 设计

```
Go 核心(现有 MVP) ──POST /generate-reviews──▶ 本服务
                  ◀──  {items:[{content,tags,score,grade}]} ──

本服务内部：
  按 platform 选 writer 专家 agent(各自 instructions + few-shot)
    → 生成 JSON {content, tags}
    → 硬过滤(高风险禁用词命中即重写)
    → 评审 agent 按 100 分 rubric 打分(手册第六部分)
    → 未达标(< MIN_PASS_SCORE)则带着问题清单重写,上限 MAX_REVISE_ROUNDS 轮
  批量请求并发执行(MAX_CONCURRENCY)
```

约束手册的落点：
- `constraints/platforms/*.py` —— 各平台结构/风格/标签(手册第一、二、三部分)
- `constraints/personas.py` —— 拟人化:身份/情感↔满意度/缺点↔满意度/口语化(第四部分)
- `constraints/banned_words.py` —— 禁用词硬过滤 + 软约束(第五部分)
- `prompts/reviewer.py` —— 100 分评分表 + 15 项自检(第六部分)

## 运行

```bash
cd agent-service
python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
cp .env.example .env          # 填入 LLM_API_KEY / LLM_BASE_URL / LLM_MODEL
python -m app.main            # 默认 http://0.0.0.0:8090
```

健康检查：
```bash
curl localhost:8090/health
```

生成示例：
```bash
curl -X POST localhost:8090/generate-reviews -H 'Content-Type: application/json' -d '{
  "store": {"store_name": "示例川菜馆", "industry_type": "餐饮", "brand_tone": "轻松自然"},
  "keywords": ["招牌椒麻鸡", "环境舒服", "适合聚餐"],
  "platform": "xiaohongshu",
  "count": 3,
  "satisfaction": "比较满意"
}'
```

## 接入 Go

- Go 侧只认这个 HTTP 契约,不含任何 provider 字眼;换模型/换 OpenAI 只改本服务内部。
- **入池阈值建议:** 本服务返回每条的 `score`/`grade`。Go 填池时建议只保留
  `grade ∈ {S, A, B}`(即 score ≥ 70),C/D 丢弃。阈值可在 Go 侧配置。

## 关键说明

- **用支持 chat/completions + JSON 输出的对话模型**(如 `gpt-5.4`、`deepseek-chat`),
  不要用纯推理模型(如 `deepseek-reasoner`/R1)——不支持 JSON 模式,评审循环用不了。
- 多数第三方端点只支持 `json_object`(非严格 schema),本服务用容错解析(`jsonutil.py`),
  不依赖严格 schema。
- tracing 已在 `client.py` 关闭(第三方 key 用不了 OpenAI tracing)。
- `openai-agents` 的精确版本/符号请以 `pip show openai-agents` 为准;本服务只用其
  稳定公开 API(`Agent` / `Runner` / `OpenAIChatCompletionsModel` / `set_tracing_disabled`)。
