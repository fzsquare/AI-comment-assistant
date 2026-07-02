-- 评价抓取分析：使用 information_schema 守卫新增列，兼容全新库先导入 schema.sql
-- 后再执行 migrations，以及旧库重复执行迁移的情况。

SET @db := DATABASE();

SET @has := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=@db AND TABLE_NAME='review_items' AND COLUMN_NAME='used_at');
SET @s := IF(@has=0, 'ALTER TABLE review_items ADD COLUMN used_at DATETIME NULL AFTER dispatched_at', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @has := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=@db AND TABLE_NAME='review_generation_tasks' AND COLUMN_NAME='generated_raw_count');
SET @s := IF(@has=0, 'ALTER TABLE review_generation_tasks ADD COLUMN generated_raw_count INT NOT NULL DEFAULT 0 AFTER target_count', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @has := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=@db AND TABLE_NAME='review_generation_tasks' AND COLUMN_NAME='inserted_row_count');
SET @s := IF(@has=0, 'ALTER TABLE review_generation_tasks ADD COLUMN inserted_row_count INT NOT NULL DEFAULT 0 AFTER generated_raw_count', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @has := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=@db AND TABLE_NAME='review_generation_tasks' AND COLUMN_NAME='duplicate_filtered_count');
SET @s := IF(@has=0, 'ALTER TABLE review_generation_tasks ADD COLUMN duplicate_filtered_count INT NOT NULL DEFAULT 0 AFTER inserted_row_count', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @has := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=@db AND TABLE_NAME='review_generation_tasks' AND COLUMN_NAME='duplicate_check_version');
SET @s := IF(@has=0, 'ALTER TABLE review_generation_tasks ADD COLUMN duplicate_check_version VARCHAR(64) NOT NULL DEFAULT '''' AFTER duplicate_filtered_count', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

CREATE TABLE IF NOT EXISTS store_review_crawl_configs (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  platform_code VARCHAR(64) NOT NULL,
  external_shop_id VARCHAR(128) NOT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 0,
  baseline_completed_at DATETIME NULL,
  last_crawled_at DATETIME NULL,
  next_crawl_at DATETIME NULL,
  last_status VARCHAR(32) NOT NULL DEFAULT 'never_run',
  last_error_message TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_review_crawl_configs_store_id (store_id),
  INDEX idx_review_crawl_configs_enabled_next (enabled, next_crawl_at),
  CONSTRAINT fk_review_crawl_config_store FOREIGN KEY (store_id) REFERENCES stores(id)
);

CREATE TABLE IF NOT EXISTS store_review_crawl_batches (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  config_id BIGINT UNSIGNED NOT NULL,
  store_id BIGINT UNSIGNED NOT NULL,
  platform_code VARCHAR(64) NOT NULL,
  external_shop_id_snapshot VARCHAR(128) NOT NULL,
  trigger_type VARCHAR(32) NOT NULL,
  attempt_no INT NOT NULL DEFAULT 1,
  is_baseline TINYINT(1) NOT NULL DEFAULT 0,
  window_days INT NOT NULL DEFAULT 7,
  window_start_at DATETIME NULL,
  window_end_at DATETIME NULL,
  started_at DATETIME NULL,
  finished_at DATETIME NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'running',
  raw_row_count INT NOT NULL DEFAULT 0,
  inserted_row_count INT NOT NULL DEFAULT 0,
  matched_review_count INT NOT NULL DEFAULT 0,
  failure_code VARCHAR(64) NOT NULL DEFAULT '',
  failure_stage VARCHAR(64) NOT NULL DEFAULT '',
  retryable TINYINT(1) NOT NULL DEFAULT 0,
  error_message TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_review_crawl_batches_config_status (config_id, status),
  INDEX idx_review_crawl_batches_store_window (store_id, platform_code, is_baseline, window_start_at, window_end_at),
  CONSTRAINT fk_review_crawl_batch_config FOREIGN KEY (config_id) REFERENCES store_review_crawl_configs(id),
  CONSTRAINT fk_review_crawl_batch_store FOREIGN KEY (store_id) REFERENCES stores(id)
);

CREATE TABLE IF NOT EXISTS external_store_reviews (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  batch_id BIGINT UNSIGNED NOT NULL,
  store_id BIGINT UNSIGNED NOT NULL,
  platform_code VARCHAR(64) NOT NULL,
  source_review_ref VARCHAR(128) DEFAULT '',
  user_name VARCHAR(255) DEFAULT '',
  rating_raw VARCHAR(32) DEFAULT '',
  rating_normalized DECIMAL(4,2) NULL,
  review_time DATETIME NULL,
  content TEXT,
  is_baseline TINYINT(1) NOT NULL DEFAULT 0,
  matched_feedback_id BIGINT UNSIGNED NULL,
  matched_review_item_id BIGINT UNSIGNED NULL,
  match_score DECIMAL(5,4) NOT NULL DEFAULT 0,
  match_reason VARCHAR(64) NOT NULL DEFAULT '',
  match_source VARCHAR(64) NOT NULL DEFAULT '',
  match_algorithm_version VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_external_reviews_store_time (store_id, platform_code, is_baseline, review_time),
  INDEX idx_external_reviews_batch (batch_id),
  INDEX idx_external_reviews_match_feedback (matched_feedback_id),
  CONSTRAINT fk_external_review_batch FOREIGN KEY (batch_id) REFERENCES store_review_crawl_batches(id),
  CONSTRAINT fk_external_review_store FOREIGN KEY (store_id) REFERENCES stores(id),
  CONSTRAINT fk_external_review_feedback FOREIGN KEY (matched_feedback_id) REFERENCES review_feedbacks(id),
  CONSTRAINT fk_external_review_item FOREIGN KEY (matched_review_item_id) REFERENCES review_items(id)
);
