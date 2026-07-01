-- 给已存在的库补按平台派发评价池的查询索引。
-- 用法：mysql ... ppk < database/migration-2026-platform-review-pool.sql

CREATE INDEX idx_review_items_dispatch_platform
  ON review_items (store_id, platform_style, status, is_dispatched);
