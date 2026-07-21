-- 管理员从已入库平台真实评论中勾选可作为生成 few-shot 的样本。
-- 选择关系独立于 external_store_reviews，保留采集原始证据的只读语义。

CREATE TABLE IF NOT EXISTS platform_review_few_shots (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  external_review_id BIGINT UNSIGNED NOT NULL,
  store_id BIGINT UNSIGNED NOT NULL,
  platform_code VARCHAR(64) NOT NULL,
  selected_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_platform_review_few_shots_external (external_review_id),
  INDEX idx_platform_review_few_shots_store_platform (store_id, platform_code, selected_at),
  CONSTRAINT fk_platform_review_few_shot_external_review FOREIGN KEY (external_review_id) REFERENCES external_store_reviews(id),
  CONSTRAINT fk_platform_review_few_shot_store FOREIGN KEY (store_id) REFERENCES stores(id)
);
