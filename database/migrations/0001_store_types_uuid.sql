-- 迁移：门店类型标签 + 店铺 uuid + nfc_tags 卡片库存化
-- 由 scripts/deploy.sh MIGRATE_DB=true 以 root 运行一次（schema_migrations 去重）。
-- 与 database/schema.sql 的最终形态保持一致。

-- 1) 类型标签表 + 9 预置（FK 目标，必须先建）
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

-- 2) stores.uuid：可空 -> 回填 -> 加唯一约束
ALTER TABLE stores ADD COLUMN uuid CHAR(36) NULL AFTER merchant_user_id;
-- 演示门店固定 uuid（与 seed 一致，保证演示链接稳定）；非演示库无该账号则空操作
UPDATE stores s JOIN merchant_users m ON s.merchant_user_id = m.id
  SET s.uuid = '11111111-1111-4111-8111-111111111111'
  WHERE m.account = 'merchant' AND (s.uuid IS NULL OR s.uuid = '');
-- 其余门店逐行随机 UUID()
UPDATE stores SET uuid = (UUID()) WHERE uuid IS NULL OR uuid = '';
ALTER TABLE stores ADD UNIQUE KEY uk_stores_uuid (uuid);

-- 3) stores.type_id：可空 FK -> 按 industry_type 中文名回填 -> 未命中兜底 restaurant
ALTER TABLE stores ADD COLUMN type_id BIGINT UNSIGNED NULL AFTER uuid,
  ADD CONSTRAINT fk_store_type FOREIGN KEY (type_id) REFERENCES store_types(id);
UPDATE stores s JOIN store_types t ON s.industry_type = t.name
  SET s.type_id = t.id WHERE s.type_id IS NULL;
UPDATE stores s JOIN store_types t ON t.code = 'restaurant'
  SET s.type_id = t.id WHERE s.type_id IS NULL;

-- 4) nfc_tags.landing_token：落地改用 store.uuid，去唯一约束 + 改可空
--    （列级 UNIQUE 的索引名默认即列名 landing_token）
ALTER TABLE nfc_tags DROP INDEX landing_token;
ALTER TABLE nfc_tags MODIFY landing_token VARCHAR(128) NULL;
