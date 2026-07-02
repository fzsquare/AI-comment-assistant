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
