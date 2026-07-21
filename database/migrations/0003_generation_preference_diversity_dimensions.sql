-- 评论生成偏好：商家选择大方向，agent 轮换具体小方向，降低长期同质化。
-- 幂等：字段不存在才添加；旧数据回填默认「顾客身份」。

SET @db := DATABASE();

SET @has := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @db
    AND TABLE_NAME = 'store_generation_preferences'
    AND COLUMN_NAME = 'diversity_dimensions'
);
SET @s := IF(
  @has = 0,
  'ALTER TABLE store_generation_preferences ADD COLUMN diversity_dimensions JSON NULL AFTER style_codes',
  'SELECT 1'
);
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

UPDATE store_generation_preferences
SET diversity_dimensions = JSON_ARRAY('customer_identity')
WHERE diversity_dimensions IS NULL OR JSON_LENGTH(diversity_dimensions) = 0;

ALTER TABLE store_generation_preferences MODIFY diversity_dimensions JSON NOT NULL;
