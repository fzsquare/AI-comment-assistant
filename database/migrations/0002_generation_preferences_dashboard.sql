-- 商家价值看板与评论优化入口。
-- 幂等：deploy.sh 通过 schema_migrations 去重执行，本文件内部仍守卫索引创建。

CREATE TABLE IF NOT EXISTS store_generation_preferences (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  focus_keywords JSON NOT NULL,
  style_codes JSON NOT NULL,
  reference_reviews JSON NOT NULL,
  length_variance VARCHAR(32) NOT NULL DEFAULT 'wide',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_store_generation_preferences_store_id (store_id),
  CONSTRAINT fk_generation_preferences_store FOREIGN KEY (store_id) REFERENCES stores(id)
);

SET @idx_exists := (
  SELECT COUNT(1)
  FROM INFORMATION_SCHEMA.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'review_display_logs'
    AND INDEX_NAME = 'idx_review_logs_store_action_created'
);

SET @sql := IF(
  @idx_exists = 0,
  'CREATE INDEX idx_review_logs_store_action_created ON review_display_logs (store_id, action_type, created_at)',
  'SELECT 1'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
