package database

import (
	"os"
	"strings"
	"testing"
)

func TestReviewCrawlAnalyticsMigrationGuardsExistingColumns(t *testing.T) {
	body, err := os.ReadFile("../../../database/migrations/0004_review_crawl_analytics.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(body)

	for _, column := range []string{
		"used_at",
		"generated_raw_count",
		"inserted_row_count",
		"duplicate_filtered_count",
		"duplicate_check_version",
	} {
		if !strings.Contains(sql, "COLUMN_NAME = '"+column+"'") && !strings.Contains(sql, "COLUMN_NAME='"+column+"'") {
			t.Fatalf("migration should guard column %q with information_schema before ADD COLUMN", column)
		}
	}
}

func TestReviewGenerationAuditLogSchemaExists(t *testing.T) {
	schema, err := os.ReadFile("../../../database/schema.sql")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	migration, err := os.ReadFile("../../../database/migrations/0005_review_generation_audit_logs.sql")
	if err != nil {
		t.Fatalf("read audit migration: %v", err)
	}

	for _, sql := range []string{string(schema), string(migration)} {
		if !strings.Contains(sql, "review_generation_audit_logs") {
			t.Fatalf("expected review_generation_audit_logs in SQL:\n%s", sql)
		}
		if !strings.Contains(sql, "task_id") || !strings.Contains(sql, "stage") || !strings.Contains(sql, "duration_ms") {
			t.Fatalf("audit SQL missing required columns:\n%s", sql)
		}
	}
}

func TestPlatformReviewFewShotSchemaExists(t *testing.T) {
	schema, err := os.ReadFile("../../../database/schema.sql")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	migration, err := os.ReadFile("../../../database/migrations/0006_platform_review_few_shots.sql")
	if err != nil {
		t.Fatalf("read few-shot migration: %v", err)
	}

	for _, sql := range []string{string(schema), string(migration)} {
		if !strings.Contains(sql, "platform_review_few_shots") {
			t.Fatalf("expected platform_review_few_shots in SQL:\n%s", sql)
		}
		for _, required := range []string{"external_review_id", "store_id", "platform_code", "selected_at"} {
			if !strings.Contains(sql, required) {
				t.Fatalf("few-shot SQL missing %q:\n%s", required, sql)
			}
		}
	}
}

func TestPublishStatsCompositeIndexExistsInSchemaAndGuardedMigration(t *testing.T) {
	schema, err := os.ReadFile("../../../database/schema.sql")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	migration, err := os.ReadFile("../../../database/migrations/0007_publish_stats_index.sql")
	if err != nil {
		t.Fatalf("read publish stats migration: %v", err)
	}

	const indexName = "idx_review_logs_store_action_created_platform_session"
	const columns = "(store_id, action_type, created_at, platform_code, session_id)"
	const platformIndexName = "idx_review_logs_store_platform_action_created_session"
	const platformColumns = "(store_id, platform_code, action_type, created_at, session_id)"
	for name, sql := range map[string]string{"schema": string(schema), "migration": string(migration)} {
		if !strings.Contains(sql, indexName) || !strings.Contains(sql, columns) || !strings.Contains(sql, platformIndexName) || !strings.Contains(sql, platformColumns) {
			t.Fatalf("%s missing publish stats composite index: %s", name, sql)
		}
	}
	if !strings.Contains(string(migration), "information_schema.statistics") {
		t.Fatalf("migration must guard the index for existing databases: %s", migration)
	}
	for name, sql := range map[string]string{"schema": string(schema), "migration": string(migration)} {
		if !strings.Contains(sql, "dispatched_session_id") {
			t.Fatalf("%s must persist which signed session received a review", name)
		}
	}
	if !strings.Contains(string(migration), "information_schema.columns") {
		t.Fatalf("migration must guard dispatched_session_id for existing databases: %s", migration)
	}
}
