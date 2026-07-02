CREATE TABLE IF NOT EXISTS admin_users (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  account VARCHAR(64) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  name VARCHAR(128) NOT NULL,
  status TINYINT NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS merchant_users (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  account VARCHAR(64) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  merchant_name VARCHAR(128) NOT NULL,
  contact_name VARCHAR(128) NOT NULL,
  status TINYINT NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 门店类型标签：预置 9 行业 + 管理员可自定义。industry_code 为生成/隔离基准
-- （对应 agent-service 的 9 行业 code）；store.industry_type 写类型中文名（含别名），
-- 保证 Python 串味隔离与 Go 推荐标签的中文子串匹配仍命中。
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

-- 9 个预置类型（参考数据，非演示数据：即使不导入 seed 也需要它们才能建店）。
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

CREATE TABLE IF NOT EXISTS stores (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  merchant_user_id BIGINT UNSIGNED NOT NULL UNIQUE,
  uuid CHAR(36) NOT NULL UNIQUE,
  type_id BIGINT UNSIGNED NULL,
  store_name VARCHAR(128) NOT NULL,
  industry_type VARCHAR(64) DEFAULT '',
  store_intro TEXT,
  address VARCHAR(255) DEFAULT '',
  primary_platform_style VARCHAR(64) NOT NULL,
  brand_tone VARCHAR(255) DEFAULT '',
  status TINYINT NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_store_merchant FOREIGN KEY (merchant_user_id) REFERENCES merchant_users(id),
  CONSTRAINT fk_store_type FOREIGN KEY (type_id) REFERENCES store_types(id)
);

CREATE TABLE IF NOT EXISTS store_keywords (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  keyword VARCHAR(128) NOT NULL,
  sort_no INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_store_keywords_store_id (store_id),
  CONSTRAINT fk_keyword_store FOREIGN KEY (store_id) REFERENCES stores(id)
);

CREATE TABLE IF NOT EXISTS store_images (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  image_url VARCHAR(500) NOT NULL,
  thumbnail_url VARCHAR(500) DEFAULT '',
  status TINYINT NOT NULL DEFAULT 1,
  sort_no INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_store_images_store_id (store_id),
  CONSTRAINT fk_image_store FOREIGN KEY (store_id) REFERENCES stores(id)
);

CREATE TABLE IF NOT EXISTS store_platform_links (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  platform_code VARCHAR(64) NOT NULL,
  platform_name VARCHAR(128) NOT NULL,
  button_text VARCHAR(128) NOT NULL,
  target_url VARCHAR(500) NOT NULL,
  backup_url VARCHAR(500) DEFAULT '',
  sort_no INT NOT NULL DEFAULT 0,
  status TINYINT NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_store_platform_code (store_id, platform_code),
  CONSTRAINT fk_platform_link_store FOREIGN KEY (store_id) REFERENCES stores(id)
);

CREATE TABLE IF NOT EXISTS review_items (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  platform_style VARCHAR(64) NOT NULL,
  content TEXT NOT NULL,
  tags VARCHAR(255) NOT NULL DEFAULT '',
  source_type VARCHAR(32) NOT NULL,
  generation_batch_no VARCHAR(64) NOT NULL,
  is_dispatched TINYINT(1) NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'available',
  dispatched_at DATETIME NULL,
  used_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_review_items_store_id (store_id),
  INDEX idx_review_items_dispatch (store_id, status, is_dispatched),
  INDEX idx_review_items_dispatch_platform (store_id, platform_style, status, is_dispatched),
  CONSTRAINT fk_review_store FOREIGN KEY (store_id) REFERENCES stores(id)
);

CREATE TABLE IF NOT EXISTS review_display_logs (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  review_item_id BIGINT UNSIGNED NULL,
  nfc_tag_id BIGINT UNSIGNED NULL,
  session_id VARCHAR(128) NOT NULL,
  action_type VARCHAR(64) NOT NULL,
  platform_code VARCHAR(64) DEFAULT '',
  client_ip VARCHAR(64) DEFAULT '',
  user_agent VARCHAR(255) DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_review_logs_store_id (store_id),
  INDEX idx_review_logs_session_id (session_id),
  INDEX idx_review_logs_store_action_created (store_id, action_type, created_at),
  CONSTRAINT fk_log_store FOREIGN KEY (store_id) REFERENCES stores(id)
);

CREATE TABLE IF NOT EXISTS review_feedbacks (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  review_item_id BIGINT UNSIGNED NOT NULL,
  session_id VARCHAR(128) NOT NULL,
  platform_code VARCHAR(64) NOT NULL,
  feedback_type VARCHAR(32) NOT NULL,
  source_action VARCHAR(64) NOT NULL,
  content_snapshot TEXT NOT NULL,
  edited_content TEXT,
  client_ip VARCHAR(64) DEFAULT '',
  user_agent VARCHAR(255) DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_review_feedbacks_store_platform_type (store_id, platform_code, feedback_type),
  INDEX idx_review_feedbacks_review_item_id (review_item_id),
  UNIQUE KEY uk_review_feedback_once (store_id, review_item_id, session_id, feedback_type),
  CONSTRAINT fk_feedback_store FOREIGN KEY (store_id) REFERENCES stores(id),
  CONSTRAINT fk_feedback_review_item FOREIGN KEY (review_item_id) REFERENCES review_items(id)
);

CREATE TABLE IF NOT EXISTS review_generation_tasks (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  platform_style VARCHAR(64) NOT NULL,
  trigger_type VARCHAR(32) NOT NULL,
  target_count INT NOT NULL,
  generated_raw_count INT NOT NULL DEFAULT 0,
  inserted_row_count INT NOT NULL DEFAULT 0,
  duplicate_filtered_count INT NOT NULL DEFAULT 0,
  duplicate_check_version VARCHAR(64) NOT NULL DEFAULT '',
  success_count INT NOT NULL DEFAULT 0,
  failed_count INT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  error_message TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_generation_tasks_store_id (store_id),
  CONSTRAINT fk_task_store FOREIGN KEY (store_id) REFERENCES stores(id)
);

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

CREATE TABLE IF NOT EXISTS store_generation_preferences (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  focus_keywords JSON NOT NULL,
  style_codes JSON NOT NULL,
  diversity_dimensions JSON NOT NULL,
  reference_reviews JSON NOT NULL,
  length_variance VARCHAR(32) NOT NULL DEFAULT 'wide',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_store_generation_preferences_store_id (store_id),
  CONSTRAINT fk_generation_preferences_store FOREIGN KEY (store_id) REFERENCES stores(id)
);

-- 卡片库存：一店可多卡。落地访问改用 store.uuid，landing_token 仅历史保留（可空、不唯一）。
CREATE TABLE IF NOT EXISTS nfc_tags (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tag_code VARCHAR(128) NOT NULL UNIQUE,
  store_id BIGINT UNSIGNED NULL,
  landing_token VARCHAR(128) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'unbound',
  remark VARCHAR(255) DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_nfc_tags_store_id (store_id),
  CONSTRAINT fk_nfc_store FOREIGN KEY (store_id) REFERENCES stores(id)
);
