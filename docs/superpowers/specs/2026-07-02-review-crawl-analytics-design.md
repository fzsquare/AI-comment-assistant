# 美团真实评论采集与引导发布占比设计

## 背景

当前系统已经有两类评论数据：

- `review_items`：系统生成的评论库存，供顾客在落地页领取、编辑、复制或跳转发布。
- `review_display_logs` / `review_feedbacks`：顾客在落地页的行为日志，包括访问、换一换、复制、点击去发布，以及对应的评论快照。

现在需要接入外部真实评论采集。管理员为每个门店配置一个平台和一个外部商家 ID。第一版平台只有美团，外部采集接口每次返回最近 7 天评论。系统每 7 天采集一次，将真实评论落库，用来分析商家使用本产品后评论变化，以及我方引导发布在周/月总评论里的占比。

## 目标

1. 管理员可为门店配置美团评论采集：平台、外部商家 ID、是否启用。
2. 系统每 7 天滚动采集一次外部最近 7 天评论，首次成功采集只作为基线。
3. 外部真实评论进入独立数据表，不混入 `review_items` 评论库存。
4. 商家后台只显示业务结果：本周/本月引导发布数，以及本周/本月我方引导发布占比。
5. 管理员后台可见内部数据：采集批次、真实评论数、失败原因、严格 C 验证命中数和匹配明细。
6. 评论被顾客使用后保留原始数据，不再用删除语义处理，以便和外部真实评论做交叉验证。
7. 新生成评论入库前检查是否和我方历史生成/库存评论高度相似，避免库存同质化。

## 非目标

1. 第一版不接入大众点评、小红书、抖音等其他平台。
2. 第一版不把管理员 C 验证详情展示给商家。
3. 第一版不展示平台真实评论数给商家，商家自己可以在平台看到该数据。
4. 新生成评论不需要和外部真实评论查重，真实平台评论可以作为自然语料存在。
5. 外部 Excel 中的评论 ID 不作为系统唯一键，不用它去重。

## 已验证的外部接口事实

本地已用两个美团外部商家 ID 验证接口链路：

- `POST http://8.138.99.159:34553/crawl`，body 为 `{"id":"<external_shop_id>"}`，可启动采集任务。
- `GET /status/{external_shop_id}` 返回 `status=true` 时可下载。
- `GET /download/{external_shop_id}` 返回 Excel 文件。
- Excel 表头为 `ID / 用户名 / 评分 / 时间 / 内容`。
- 示例 ID `1226022464` 返回 8 条有效评论。
- 示例 ID `1953748828` 返回 36 条有效评论。

接口每次返回最近 7 天评论，所以系统设计按滚动 7 天采集，展示按自然周和自然月聚合。

## 核心口径

### A 口径：商家可见的默认归因

顾客点击落地页的“去发布”或平台跳转按钮，即记录为一次我方引导发布。该口径来自 `review_display_logs.action_type = platform_link_click`。

商家后台展示：

- 本周引导发布数。
- 本月引导发布数。
- 本周我方引导发布占比。
- 本月我方引导发布占比。

占比的分母来自外部采集到的真实评论数，但商家后台不直接展示该分母。

### C 口径：管理员可见的严格验证

外部真实评论入库后，系统将真实评论内容和我方已使用评论进行严格相似度匹配。命中结果仅管理员可见，用于验证“点击去发布”最终有多少真实出现在平台上。

匹配优先级：

1. `review_feedbacks.edited_content`，即顾客最终编辑后的内容。
2. `review_feedbacks.content_snapshot`，即顾客领取时的系统评论原文。
3. 必要时可回查 `review_items.content`，但验证依据优先使用反馈快照。

## 数据模型

### `store_review_crawl_configs`

门店级采集配置。每个门店第一版最多一条启用配置。

字段建议：

| 字段 | 含义 |
| --- | --- |
| `id` | 系统内部主键 |
| `store_id` | 门店 ID |
| `platform_code` | 平台编码，第一版只允许 `meituan` |
| `external_shop_id` | 管理员配置的美团外部商家 ID |
| `enabled` | 是否启用采集 |
| `baseline_completed_at` | 首次基线完成时间 |
| `last_crawled_at` | 最近一次成功采集时间 |
| `next_crawl_at` | 下次应采集时间 |
| `last_status` | 最近状态：`never_run` / `success` / `failed` / `running` |
| `last_error_message` | 最近失败原因 |
| `created_at` / `updated_at` | 创建和更新时间 |

唯一约束：

- `UNIQUE(store_id)`，因为每个门店当前只有一个唯一外部评论 ID。

### `store_review_crawl_batches`

每次采集产生一个批次。

字段建议：

| 字段 | 含义 |
| --- | --- |
| `id` | 系统内部主键 |
| `config_id` | 采集配置 ID |
| `store_id` | 门店 ID，便于查询 |
| `platform_code` | 平台编码 |
| `external_shop_id_snapshot` | 批次执行时使用的外部商家 ID |
| `trigger_type` | `scheduled` / `manual` |
| `attempt_no` | 同一配置的尝试序号 |
| `is_baseline` | 是否首次基线批次 |
| `window_days` | 固定为 7 |
| `window_start_at` | 本批次对应的评论窗口开始时间 |
| `window_end_at` | 本批次对应的评论窗口结束时间 |
| `started_at` | 开始时间 |
| `finished_at` | 结束时间 |
| `status` | `running` / `success` / `failed` |
| `raw_row_count` | Excel 有效数据行数 |
| `inserted_row_count` | 成功入库行数 |
| `matched_review_count` | C 口径验证命中数 |
| `failure_code` | 机器可读失败原因 |
| `failure_stage` | `crawl_start` / `status_poll` / `download` / `parse` / `import` / `match` |
| `retryable` | 是否可重试 |
| `error_message` | 失败原因 |
| `created_at` / `updated_at` | 创建和更新时间 |

失败批次不进入商家侧统计。

### `external_store_reviews`

真实平台评论明细。每条 Excel 数据行生成一条内部记录。

字段建议：

| 字段 | 含义 |
| --- | --- |
| `id` | 系统内部主键，不能使用外部 Excel ID |
| `batch_id` | 来源采集批次 |
| `store_id` | 门店 ID |
| `platform_code` | 平台编码 |
| `source_review_ref` | Excel 中的原始 `ID` 字段，仅保存原始值，不做唯一键 |
| `user_name` | 用户名 |
| `rating_raw` | Excel 原始评分，例如 `50`、`45`、`5` |
| `rating_normalized` | 归一化到 0-5 的评分 |
| `review_time` | 平台评论时间 |
| `content` | 评论内容 |
| `is_baseline` | 是否来自基线批次 |
| `matched_feedback_id` | 命中的我方反馈 ID，未命中为空 |
| `matched_review_item_id` | 命中的我方评论 ID，未命中为空 |
| `match_score` | 相似度分数 |
| `match_reason` | 命中原因 |
| `match_source` | `edited_content` / `content_snapshot` / `review_item` |
| `match_algorithm_version` | 匹配算法版本 |
| `created_at` | 入库时间 |

不按外部 ID 去重，不按用户名去重，也不跨批次去重。同一用户多次评论应全部入库。

### `review_items` 状态调整

现有 `review_items` 不应在顾客使用后表达为删除。建议增加或复用更准确状态：

- `available`：可发放库存。
- `disabled`：被换一换或运营禁用。
- `used`：顾客已复制或点击去发布，作为我方已使用评论留档。
- `pending_review`：保留现有审核中语义。

顾客拿到评论时继续设置 `is_dispatched=true`、`dispatched_at=...`。顾客点击复制或去发布后，设置为 `used`，并记录 `used_at` 或通过 `review_feedbacks.created_at` 回查使用时间。

如果不新增 `used_at` 字段，统计验证仍可通过 `review_feedbacks` 完成；但新增字段会让后台排查更直观。

兼容规则：

- 新增 `ReviewStatusUsed = "used"`。
- 新的 `accepted` 反馈不再写 `deleted`，改写 `used`。
- 历史 `deleted` 不做全量改写，因为其中可能包含商家手动删除的评论。
- C 验证和新生成查重中，只有能在 `review_feedbacks` 找到 accepted 记录的历史 `deleted` 评论，才按已使用评论处理。
- 商家评论库存列表默认隐藏 `used` 和 `deleted`，避免已使用评论重新进入运营视野。

## 采集流程

1. 管理员配置门店采集：
   - 平台：第一版只能选择美团。
   - 美团商家 ID：例如 `1953748828`。
   - 是否启用：启用后进入采集队列。

2. 调度器定期扫描：
   - `enabled = true`。
   - `next_crawl_at` 为空，或 `next_crawl_at <= now`。
   - 门店和商家账号仍启用。
   - 同一配置不存在 `running` 批次。

3. 对每个到期配置创建 `running` 批次。

4. 调用外部采集服务：
   - `POST /crawl` 启动任务。
   - 有限轮询 `GET /status/{external_shop_id}`。
   - 成功后下载 `GET /download/{external_shop_id}`。

5. 解析 Excel：
   - 必须包含表头 `ID / 用户名 / 评分 / 时间 / 内容`。
   - 每个有效数据行生成一条 `external_store_reviews`。
   - Excel 原始 `ID` 只保存到 `source_review_ref`。
   - 同一个用户名多条评论全部入库。

6. 事务提交：
   - 批次明细和批次成功状态在同一事务中提交。
   - 如果解析失败或入库失败，该批次标记失败，不写入明细，不进入统计。

7. 更新配置：
   - 首次成功批次标记 `is_baseline=true`，写入 `baseline_completed_at`。
   - 后续成功批次标记 `is_baseline=false`。
   - 成功后 `last_crawled_at = now`，`next_crawl_at = now + 7 days`。
   - 失败后记录 `last_error_message`，管理员可手动重试。

### 幂等和人工同步规则

- 同一配置同一时间只允许一个 `running` 批次。
- 调度器通过事务性 claim 把到期配置标记为运行中，避免多实例重复采集。
- 如果存在 `running` 批次超过超时时间，标记为 `failed`，`failure_code=stale_running_batch`，然后允许后续重试。
- 管理员手动同步第一版只用于首次基线或失败重试。
- 如果最近一次成功采集后还未到 `next_crawl_at`，手动同步不创建新的统计批次，返回最近成功状态，避免重复写入同一 7 天窗口导致分母膨胀。
- 手动重试失败批次会创建新的 `manual` 批次，旧失败批次保留作为审计记录。

## 统计流程

### 商家后台

商家后台只展示业务结果，不展示真实评论分母和采集细节。

字段建议：

| 字段 | 含义 |
| --- | --- |
| `currentWeekPublishClicks` | 本周引导发布数 |
| `currentMonthPublishClicks` | 本月引导发布数 |
| `weeklyGuidedSharePercent` | 本周我方引导发布占比 |
| `monthlyGuidedSharePercent` | 本月我方引导发布占比 |
| `crawlDataReady` | 是否已有可用于计算占比的采集数据 |
| `crawlDataMessage` | 不可用时统一为“数据积累中” |

占比计算：

```text
周占比 = 本周 platform_link_click 数 / 本周非基线真实评论数
月占比 = 本月 platform_link_click 数 / 本月非基线真实评论数
```

商家后台展示规则：

- 没有完成基线：显示“数据积累中”。
- 最近采集失败：显示“数据积累中”。
- 分母为 0 或不可用：显示“数据积累中”。
- 外部服务未配置：显示“数据积累中”。
- 如果当前自然周/月没有至少一个成功的非基线批次覆盖到该周期，显示“数据积累中”。
- `baseline_completed_at` 之前的点击不参与占比 numerator。
- 时间口径统一使用 Asia/Shanghai；真实评论按 `review_time` 聚合，点击按 `review_display_logs.created_at` 聚合。
- 不展示平台真实评论数。
- 不展示 C 验证数据。

### 管理员后台

管理员后台展示完整运营和排障信息：

- 门店是否启用采集。
- 平台和外部商家 ID。
- 最近采集时间和下次采集时间。
- 最近批次状态、失败原因、抓到多少行、入库多少行。
- 非基线真实评论周/月数量。
- A 口径引导发布周/月数量。
- A 口径引导占比。
- C 口径命中数、命中率、匹配明细。

管理员可以手动触发一次同步，手动同步和定时同步使用同一套采集、导入、验证逻辑。

## 严格 C 验证设计

验证只用于管理员内部真实性校验。

候选我方评论：

- 同门店、同平台。
- `review_feedbacks.feedback_type = accepted`。
- 来源动作包括 `platform_link_click` 和 `review_copy`。
- 优先使用 `edited_content`，为空时使用 `content_snapshot`。

匹配规则第一版采用可解释规则，不使用 AI 判定：

1. 文本归一化：
   - 去除空白符。
   - 统一全角半角。
   - 去除常见标点。
   - 转小写。

2. 强命中：
   - 归一化后完全相等。
   - 一方完整包含另一方，且较短文本长度达到最低阈值。

3. 相似命中：
   - 长公共片段达到阈值。
   - 或字符级相似度超过阈值。

4. 记录命中：
   - 写入 `matched_feedback_id`。
   - 写入 `matched_review_item_id`。
   - 写入 `match_score` 和 `match_reason`。

一条真实评论最多匹配一条我方评论，选择分数最高者。管理员可以看到匹配详情，商家不可见。

第一版阈值建议：

- 归一化后文本长度小于 12 个中文字符时，不做相似命中，只允许完全相等。
- 包含命中要求较短文本长度至少 18 个字符。
- 最长公共子串长度至少 24 个字符，或占较短文本 70% 以上。
- 字符级相似度阈值为 0.86。
- 候选时间窗默认从我方 accepted 反馈时间开始，到外部评论 `review_time` 后 14 天；超出窗口不自动命中。
- 分数相同时优先级为 `edited_content` 高于 `content_snapshot`，`platform_link_click` 高于 `review_copy`，时间更接近者优先。

## 新生成评论查重设计

新生成评论入库前只和我方评论比对，不和外部真实评论比对。

查重范围：

- 同门店、同平台的当前可用库存评论。
- 同门店、同平台的已发放或已使用评论。
- 同批次内已准备入库的新评论。

不查重范围：

- `external_store_reviews` 外部真实评论。

处理方式：

- 如果新评论和我方历史/库存评论高度相似，则拒绝该条入库。
- 生成任务记录被过滤数量。
- 如果过滤后入库数量小于目标数量但大于 0，生成任务标记 `partial_failed`，记录过滤数量。
- 如果过滤后入库数量为 0，生成任务标记 `failed`。
- 第一版不递归补生成，避免一个请求里无限重试；后续可由自动补池机制再次触发。

这样避免我方评论池同质化，同时允许外部真实评论自然相似。

## 后端接口建议

### 管理员接口

- `GET /api/admin/stores`：返回门店采集配置摘要和最近批次状态。
- `POST /api/admin/stores` / `PUT /api/admin/stores/:id`：支持保存采集平台、美团商家 ID、启用状态。
- `POST /api/admin/stores/:id/review-crawl/run`：手动触发一次同步。
- `GET /api/admin/stores/:id/review-crawl/batches`：查看采集批次。
- `GET /api/admin/stores/:id/review-crawl/matches`：查看 C 验证命中详情。

### 商家接口

现有 `GET /api/merchant/dashboard/publish-stats` 可扩展返回占比字段：

- `weeklyGuidedSharePercent`
- `monthlyGuidedSharePercent`
- `crawlDataReady`
- `crawlDataMessage`

不返回平台真实评论数，不返回 C 验证字段。

## 配置项

后端需要新增服务端配置：

| 环境变量 | 含义 |
| --- | --- |
| `REVIEW_CRAWL_SERVICE_URL` | 外部评论采集服务地址，例如 `http://8.138.99.159:34553` |
| `REVIEW_CRAWL_POLL_INTERVAL_SECONDS` | 轮询间隔，默认 3 |
| `REVIEW_CRAWL_POLL_MAX_ATTEMPTS` | 最大轮询次数，默认 40 |
| `REVIEW_CRAWL_HTTP_TIMEOUT_SECONDS` | 单次 HTTP 超时，默认 20 |
| `REVIEW_CRAWL_MAX_DOWNLOAD_BYTES` | Excel 下载大小上限，默认 5MB |

生产环境如果未配置 `REVIEW_CRAWL_SERVICE_URL`，管理员后台应显示采集服务未配置；商家后台统一显示“数据积累中”。

Excel 解析第一版优先使用 Go 标准库读取 `.xlsx` 的 zip/XML 结构，不新增第三方依赖。测试使用本地生成的最小 xlsx fixture，不依赖真实外部 IP 服务。

## 错误处理

- 外部服务未配置：管理员看到明确原因，商家只看到“数据积累中”。
- 外部服务请求失败：记录失败批次，不影响其他门店。
- 状态轮询超时：记录失败批次，可手动重试。
- 下载到的不是 Excel：记录失败批次。
- Excel 表头缺失或格式异常：记录失败批次，不写入明细。
- 单条评论内容为空：仍可入库，但后续 C 验证不会命中；是否计入真实总评论按 Excel 有效行判断。
- 评分异常：保留 `rating_raw`，`rating_normalized` 为空或 0，批次不因此失败。

## 删除和数据清理

删除门店时需要清理：

- `store_review_crawl_configs`
- `store_review_crawl_batches`
- `external_store_reviews`

同时保留现有删除规则：解绑 NFC 卡，删除评论池、反馈、展示日志、生成任务、平台链接、图片、关键词、生成偏好等关联数据。

## 迁移要求

需要同时更新：

- `database/schema.sql`
- 新增 `database/migrations/0004_review_crawl_analytics.sql`

只改 `schema.sql` 不够，因为已部署数据库需要增量迁移。

## 前端调整

### 管理员后台

门店表单新增采集配置区：

- 评论采集平台：第一版固定美团。
- 美团商家 ID。
- 启用评论采集。

门店列表或详情新增：

- 采集状态。
- 最近采集时间。
- 下次采集时间。
- 最近批次结果。
- 手动同步按钮。

管理员详情区可查看：

- 批次列表。
- 最近真实评论数量。
- C 验证命中明细。

### 商家后台

保留当前引导发布看板。新增或扩展展示：

- 本周引导发布占比。
- 本月引导发布占比。
- 数据不可用时显示“数据积累中”。

不展示平台真实评论数，不展示采集失败细节，不展示 C 验证数据。

## 测试计划

### 后端单测

- 采集配置保存和读取。
- 只允许第一版平台 `meituan`。
- 首次成功批次作为基线，不计入商家占比。
- 非基线批次按自然周/月统计。
- 商家接口不返回平台真实评论数和 C 验证详情。
- 管理员接口返回采集状态、批次和 C 验证详情。
- 外部服务未配置时商家显示“数据积累中”。
- 失败批次不进入统计。
- 删除门店清理采集相关表。
- 手动同步不会在未到周期时重复创建统计批次。
- 超时 running 批次会进入失败并允许重试。

### 导入和解析测试

- Excel 表头 `ID / 用户名 / 评分 / 时间 / 内容` 正常解析。
- 同一个用户名多条评论全部入库。
- Excel 原始 ID 保存到 `source_review_ref`，不作为唯一键。
- 表头缺失时批次失败，不写明细。
- 评分 `50`、`45`、`5` 可保留原始值并归一化。
- 本地 fake crawler / fixture 能跑通基线批次和第二个非基线批次。

### 相似度测试

- 外部真实评论能和已使用我方评论匹配。
- 优先匹配 `edited_content`。
- 新生成评论只和我方库存/历史评论查重。
- 新生成评论不和外部真实评论查重。
- 高度相似的新生成评论被过滤。

### 前端测试

- 管理员可配置美团商家 ID 和启用状态。
- 管理员可看到采集状态和手动同步入口。
- 商家后台只看到引导发布数和占比。
- 商家后台在无基线、采集失败、分母不可用时显示“数据积累中”。

### 验证命令

- `go test ./...`
- `npm run build`
- `git diff --check`

如改动到 agent-service，再补充：

- `python3 -m pytest agent-service/tests`

## 实施顺序建议

1. 数据库迁移和 Go model。
2. 评论状态语义调整：`deleted` 逐步替换为 `used`，并保持旧数据兼容。
3. 相似度工具函数和单测。
4. 外部采集 client、Excel 解析和批次导入服务。
5. 定时扫描和管理员手动同步入口。
6. 管理员配置和批次可见接口。
7. 商家 publish-stats 占比字段。
8. 前端管理员配置和商家看板展示。
9. 全量测试与一次真实美团 ID 手动采集验证。

## 风险和约束

- 外部采集服务当前是单一 HTTP 服务，必须有超时、有限轮询和失败批次记录。
- 外部接口返回最近 7 天，采集延迟会影响自然周/月归因，需要按 `review_time` 而不是采集时间聚合。
- 商家后台不展示分母，避免商家把采集异常误解为平台评论减少。
- C 验证不是商家承诺口径，只是管理员内部真实性校验。
- 旧数据里可能已经有 `review_items.status = deleted` 表达已使用，需要迁移或兼容查询。

## 空白 Agent DevEx 评审意见

评审方式：使用不继承当前对话上下文的空白 agent，按 `plan-devex-review` 的 DX POLISH 路径做非交互式评审。产品类型判断为内部 API/service 加运营工作流，主要开发者画像为需要实现、测试、排障并上线该能力的后端/运维开发者。

### 主要发现

1. P1：7 天采集幂等和调度锁原设计不足。已补充单配置单 running 批次、事务性 claim、stale running 失败处理、手动同步不重复创建成功周期批次、`window_start_at` / `window_end_at` / `trigger_type` / `attempt_no` 字段。
2. P1：基线和周期 readiness 规则不够精确。已补充 `baseline_completed_at` 前点击不参与占比、当前周/月必须有成功非基线批次、统一 Asia/Shanghai 时间口径。
3. P1：`deleted` 到 `used` 的兼容规则不明确。已补充新增 `used`，新 accepted 写 `used`，历史 `deleted` 仅在存在 accepted feedback 时按已使用参与验证和查重，避免误把商家手动删除当作已使用。
4. P1：C 验证缺少确定阈值和审计字段。已补充 `match_source`、`match_algorithm_version`、长度阈值、包含阈值、最长公共子串阈值、字符相似度阈值、时间窗和 tie-breaker。
5. P2：生成查重缺少任务记账。已补充 `partial_failed` 处理、过滤数量记录、不递归补生成的第一版策略。
6. P2：运营失败原因不能只有字符串。已补充 `failure_code`、`failure_stage`、`retryable` 和外部服务阶段枚举。
7. P2：本地开发路径缺少 fake crawler 和 fixture。已补充标准库 xlsx 解析、本地 fixture、fake crawler 测试要求和下载大小上限。

### DX Scorecard

| Dimension | Score | Reason |
| --- | ---: | --- |
| Getting Started | 7/10 | 实施顺序清晰，补充 fake crawler 和 fixture 后本地验证路径更短 |
| API/Service Design | 7/10 | 接口边界清晰，已补幂等和批次字段 |
| Error Messages | 7/10 | 已补 failure code、stage、retryable，商家侧仍保持简单 |
| Documentation | 7/10 | 设计文档覆盖主要行为，后续还应在部署文档补环境变量 |
| Upgrade Path | 7/10 | 已明确 `deleted` / `used` 兼容，不做危险全量改写 |
| Dev Environment | 7/10 | 不新增 Excel 依赖，要求 fixture 和 fake crawler |
| Operator Visibility | 8/10 | 管理员和商家可见范围拆分清楚 |
| DX Measurement | 7/10 | 批次、失败、匹配和占比都有可观测字段 |

Overall DX：7.1/10。实现者目标路径应控制在 10 分钟内完成本地基线批次和第二个非基线批次测试。

### 已解决的评审决策

- 最近采集失败时商家后台不使用陈旧成功数据，统一显示“数据积累中”。
- 手动同步不在成功周期内重复创建统计批次，只用于首次基线或失败重试。
- 历史 `deleted` 不全量迁移为 `used`，只在存在 accepted feedback 时按已使用处理。
- C 验证第一版采用可解释阈值，不使用 AI 判定。
- 重复过滤后不自动递归补生成，避免请求内无限重试。
