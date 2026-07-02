package service

import (
	"testing"
	"time"
)

func TestCrawlGuidedShareStatsWithoutDataAccumulates(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.Local)

	stats := CrawlGuidedShareStats(nil, 1, now.AddDate(0, 0, -7), now, now.AddDate(0, -1, 0), now)

	if stats.WeeklyReady || stats.MonthlyReady {
		t.Fatalf("expected shares to be hidden while data accumulates, got %+v", stats)
	}
	if stats.Message != "数据积累中" {
		t.Fatalf("message got %q, want 数据积累中", stats.Message)
	}
}
