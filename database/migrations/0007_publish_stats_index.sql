-- Delivery 2 商家 7/30 天漏斗与趋势聚合索引。
-- information_schema 守卫保证旧库可升级，也允许部署脚本重复执行。

SET @db := DATABASE();
SET @index_name := 'idx_review_logs_store_action_created_platform_session';
SET @has := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE TABLE_SCHEMA = @db
    AND TABLE_NAME = 'review_display_logs'
    AND INDEX_NAME = @index_name
);
SET @s := IF(
  @has = 0,
  'ALTER TABLE review_display_logs ADD INDEX idx_review_logs_store_action_created_platform_session (store_id, action_type, created_at, platform_code, session_id)',
  'SELECT 1'
);
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @index_name := 'idx_review_logs_store_platform_action_created_session';
SET @has := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE TABLE_SCHEMA = @db
    AND TABLE_NAME = 'review_display_logs'
    AND INDEX_NAME = @index_name
);
SET @s := IF(
  @has = 0,
  'ALTER TABLE review_display_logs ADD INDEX idx_review_logs_store_platform_action_created_session (store_id, platform_code, action_type, created_at, session_id)',
  'SELECT 1'
);
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @column_name := 'dispatched_session_id';
SET @has_column := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE TABLE_SCHEMA = @db
    AND TABLE_NAME = 'review_items'
    AND COLUMN_NAME = @column_name
);
SET @s := IF(
  @has_column = 0,
  'ALTER TABLE review_items ADD COLUMN dispatched_session_id VARCHAR(128) NOT NULL DEFAULT '''' AFTER dispatched_at',
  'SELECT 1'
);
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;
