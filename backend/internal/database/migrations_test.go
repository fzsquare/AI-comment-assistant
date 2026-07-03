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
