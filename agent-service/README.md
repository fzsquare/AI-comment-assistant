# 多平台文案生成 Agent 服务

独立的 Python 服务，用 **OpenAI Agents SDK + 任意 OpenAI 兼容端点**（OpenAI、GPT 代理、
DeepSeek 等），按《多平台文案生成约束手册》
为大众点评 / 美团 / 小红书 / 抖音评论生成**优质、合规、拟人化**的真实评价文案。Go 后端通过
HTTP 调用本服务来填充评价池；当前项目的主 AI 生成路径已经切到本服务。

本服务是内部服务，默认监听 `127.0.0.1:8090`。生产环境只允许 Go backend 通过
`X-Agent-Internal-Token` 调用，不直接暴露给浏览器、前端服务商或公网网关。

## 设计

```
Go backend ──POST /generate-reviews + X-Agent-Internal-Token──▶ 本服务
           ◀──  {items:[{content,tags,score,grade}]} ───────────

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
- `meituan` 当前复用大众点评约束，平台编码仍会原样返回给 Go backend 写入评价池
- `constraints/personas.py` —— 拟人化:身份/情感↔满意度/缺点↔满意度/口语化(第四部分)
- `constraints/banned_words.py` —— 禁用词硬过滤 + 软约束(第五部分)
- `prompts/reviewer.py` —— 100 分评分表 + 15 项自检(第六部分)

## 运行

```bash
cd agent-service
python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
cp .env.example .env          # 填入 LLM_API_KEY / LLM_BASE_URL / LLM_MODEL / AGENT_INTERNAL_TOKEN
python -m app.main            # 默认 http://127.0.0.1:8090
```

健康检查：
```bash
curl http://127.0.0.1:8090/health
```

生成示例：
```bash
curl -X POST http://127.0.0.1:8090/generate-reviews \
  -H 'Content-Type: application/json' \
  -H 'X-Agent-Internal-Token: replace-with-shared-internal-token' \
  -d '{
  "store": {"store_name": "示例川菜馆", "industry_type": "餐饮", "brand_tone": "轻松自然"},
  "keywords": ["招牌椒麻鸡", "环境舒服", "适合聚餐"],
  "platform": "xiaohongshu",
  "count": 3,
  "satisfaction": "比较满意"
}'
```

## 接入 Go

- Go 侧只认这个 HTTP 契约,不含任何 provider 字眼;换模型/换 OpenAI 只改本服务内部。
- Go 侧会把商家/消费者选择的 `platformCode` 传入本服务，并用同一个平台编码写入 `review_items.platform_style`。
- Go backend 的 `AGENT_SERVICE_URL` 指向本机或私有网络地址,并通过
  `X-Agent-Internal-Token` 传入与本服务一致的 `AGENT_INTERNAL_TOKEN`。
- 前端只请求 Go backend 的 `/api`,不要配置或调用本服务地址。
- **入池阈值:** 本服务返回每条的 `score`/`grade`。Go backend 默认只保留
  `grade ∈ {S, A, B}`(即 score ≥ 70),C/D 丢弃；阈值可通过 `AGENT_MIN_GRADE` 调整。
- 内置 mock 生成器只在评价池为空且 agent-service 不可用时兜底补 1 条,避免消费者落地页白屏。

## 关键说明

- **用支持 chat/completions + JSON 输出的对话模型**(如 `gpt-5.4`、`deepseek-chat`),
  不要用纯推理模型(如 `deepseek-reasoner`/R1)——不支持 JSON 模式,评审循环用不了。
- 多数第三方端点只支持 `json_object`(非严格 schema),本服务用容错解析(`jsonutil.py`),
  不依赖严格 schema。
- tracing 已在 `client.py` 关闭(第三方 key 用不了 OpenAI tracing)。
- `openai-agents` 的精确版本/符号请以 `pip show openai-agents` 为准;本服务只用其
  稳定公开 API(`Agent` / `Runner` / `OpenAIChatCompletionsModel` / `set_tracing_disabled`)。
