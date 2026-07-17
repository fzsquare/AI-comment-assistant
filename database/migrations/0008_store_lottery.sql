-- 到店即时赠品抽奖：不依赖支付、会员、券或后续核销。
CREATE TABLE IF NOT EXISTS store_lottery_configs (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_store_lottery_config_store_id (store_id),
  CONSTRAINT fk_lottery_config_store FOREIGN KEY (store_id) REFERENCES stores(id)
);

CREATE TABLE IF NOT EXISTS store_lottery_prizes (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(64) NOT NULL,
  image_url VARCHAR(500) NOT NULL DEFAULT '',
  stock INT NOT NULL,
  win_rate INT NOT NULL,
  sort_no INT NOT NULL DEFAULT 0,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_lottery_prizes_store (store_id),
  CONSTRAINT fk_lottery_prize_store FOREIGN KEY (store_id) REFERENCES stores(id)
);

CREATE TABLE IF NOT EXISTS store_lottery_draws (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT UNSIGNED NOT NULL,
  session_id VARCHAR(128) NOT NULL,
  prize_id BIGINT UNSIGNED NULL,
  outcome VARCHAR(16) NOT NULL,
  prize_name VARCHAR(64) NOT NULL DEFAULT '',
  prize_image_url VARCHAR(500) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_lottery_draw_store_session (store_id, session_id),
  KEY idx_lottery_draws_prize (prize_id),
  CONSTRAINT fk_lottery_draw_store FOREIGN KEY (store_id) REFERENCES stores(id)
);
