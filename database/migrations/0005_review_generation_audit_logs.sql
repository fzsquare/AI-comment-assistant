-- 评价生成审计日志：记录 Go 后端调用本地 agent-service 的阶段、耗时和失败原因。

CREATE TABLE IF NOT EXISTS review_generation_audit_logs (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  task_id BIGINT UNSIGNED NOT NULL,
  store_id BIGINT UNSIGNED NOT NULL,
  platform_style VARCHAR(64) NOT NULL,
  trigger_type VARCHAR(32) NOT NULL,
  stage VARCHAR(64) NOT NULL,
  level VARCHAR(16) NOT NULL,
  status VARCHAR(32) NOT NULL,
  message VARCHAR(512) NOT NULL,
  detail TEXT,
  agent_endpoint VARCHAR(255) DEFAULT '',
  http_status INT NOT NULL DEFAULT 0,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  target_count INT NOT NULL DEFAULT 0,
  generated_raw_count INT NOT NULL DEFAULT 0,
  inserted_row_count INT NOT NULL DEFAULT 0,
  duplicate_filtered_count INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_generation_audit_task_id (task_id),
  INDEX idx_generation_audit_store_id (store_id),
  INDEX idx_generation_audit_stage (stage),
  CONSTRAINT fk_generation_audit_task FOREIGN KEY (task_id) REFERENCES review_generation_tasks(id),
  CONSTRAINT fk_generation_audit_store FOREIGN KEY (store_id) REFERENCES stores(id)
);
