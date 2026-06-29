-- 迁移：门店类型标签 + 店铺 uuid + nfc_tags 卡片库存化
-- 幂等：每个 DDL 用 information_schema 守卫，可在「全新库（schema.sql 已建最终形态）」
-- 与「旧库（缺列）」上反复运行而不报错。由 deploy.sh 在 schema 之后、seed 之前运行。

SET @db := DATABASE();

-- 1) 类型标签表 + 9 预置（CREATE IF NOT EXISTS / INSERT ON DUPLICATE 本身幂等）
CREATE TABLE IF NOT EXISTS store_types (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  code VARCHAR(64) NOT NULL UNIQUE,
  name VARCHAR(64) NOT NULL,
  industry_code VARCHAR(64) NOT NULL DEFAULT 'restaurant',
  is_preset TINYINT NOT NULL DEFAULT 0,
  status TINYINT NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

INSERT INTO store_types (code, name, industry_code, is_preset, status) VALUES
  ('restaurant', '餐饮', 'restaurant', 1, 1),
  ('footmassage', '足疗按摩', 'footmassage', 1, 1),
  ('hairsalon', '理发美发', 'hairsalon', 1, 1),
  ('nailsalon', '美甲美睫', 'nailsalon', 1, 1),
  ('beauty', '美容护肤', 'beauty', 1, 1),
  ('fitness', '健身运动', 'fitness', 1, 1),
  ('entertainment', '休闲娱乐', 'entertainment', 1, 1),
  ('pet', '宠物服务', 'pet', 1, 1),
  ('auto', '汽车服务', 'auto', 1, 1)
ON DUPLICATE KEY UPDATE name = VALUES(name), industry_code = VALUES(industry_code), is_preset = VALUES(is_preset);

-- 2) stores.uuid 列（守卫：不存在才加）
SET @has := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=@db AND TABLE_NAME='stores' AND COLUMN_NAME='uuid');
SET @s := IF(@has=0, 'ALTER TABLE stores ADD COLUMN uuid CHAR(36) NULL AFTER merchant_user_id', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- 回填：演示门店固定 uuid（与 seed 一致），其余逐行随机（均幂等：仅填空值）
UPDATE stores s JOIN merchant_users m ON s.merchant_user_id = m.id
  SET s.uuid = '11111111-1111-4111-8111-111111111111'
  WHERE m.account = 'merchant' AND (s.uuid IS NULL OR s.uuid = '');
UPDATE stores SET uuid = (UUID()) WHERE uuid IS NULL OR uuid = '';

-- uuid 唯一约束（守卫：索引不存在才加）
SET @has := (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=@db AND TABLE_NAME='stores' AND INDEX_NAME='uk_stores_uuid');
SET @s := IF(@has=0, 'ALTER TABLE stores ADD UNIQUE KEY uk_stores_uuid (uuid)', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- 3) stores.type_id 列 + 外键（各自守卫）
SET @has := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=@db AND TABLE_NAME='stores' AND COLUMN_NAME='type_id');
SET @s := IF(@has=0, 'ALTER TABLE stores ADD COLUMN type_id BIGINT UNSIGNED NULL AFTER uuid', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @has := (SELECT COUNT(*) FROM information_schema.TABLE_CONSTRAINTS WHERE TABLE_SCHEMA=@db AND TABLE_NAME='stores' AND CONSTRAINT_NAME='fk_store_type');
SET @s := IF(@has=0, 'ALTER TABLE stores ADD CONSTRAINT fk_store_type FOREIGN KEY (type_id) REFERENCES store_types(id)', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- 回填 type_id：按 industry_type 中文名匹配，未命中兜底 restaurant（幂等：仅填 NULL）
UPDATE stores s JOIN store_types t ON s.industry_type = t.name
  SET s.type_id = t.id WHERE s.type_id IS NULL;
UPDATE stores s JOIN store_types t ON t.code = 'restaurant'
  SET s.type_id = t.id WHERE s.type_id IS NULL;

-- 4) nfc_tags.landing_token：去唯一约束（守卫）+ 改可空（MODIFY 幂等）
SET @has := (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=@db AND TABLE_NAME='nfc_tags' AND INDEX_NAME='landing_token');
SET @s := IF(@has>0, 'ALTER TABLE nfc_tags DROP INDEX landing_token', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

ALTER TABLE nfc_tags MODIFY landing_token VARCHAR(128) NULL;
