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
