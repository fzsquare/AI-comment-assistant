-- 给已存在的库补 review_items.tags 列（新库由 schema.sql 直接带）。
-- 用法：mysql ... ppk < database/migration-2026-add-review-tags.sql
USE ppk;

ALTER TABLE review_items
  ADD COLUMN IF NOT EXISTS tags VARCHAR(255) NOT NULL DEFAULT '' AFTER content;
