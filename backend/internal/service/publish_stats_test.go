package service

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func mustDashboardTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation("2006-01-02 15:04", value, dashboardStatsLocation())
	if err != nil {
		t.Fatalf("parse dashboard time: %v", err)
	}
	return parsed
}

func TestResolvePublishStatsRangeDefaultsToSevenDaysAndSupportsThirtyDays(t *testing.T) {
	now := mustDashboardTime(t, "2027-01-02 09:30")

	seven, err := ResolvePublishStatsRange("", now)
	if err != nil {
		t.Fatalf("default range: %v", err)
	}
	if seven.Code != "7d" || seven.Start.Format("2006-01-02") != "2026-12-27" || seven.End.Format("2006-01-02") != "2027-01-03" {
		t.Fatalf("unexpected 7d range: %#v", seven)
	}

	thirty, err := ResolvePublishStatsRange("30d", now)
	if err != nil {
		t.Fatalf("30d range: %v", err)
	}
	if thirty.Start.Format("2006-01-02") != "2026-12-04" || thirty.End.Format("2006-01-02") != "2027-01-03" {
		t.Fatalf("unexpected 30d range: %#v", thirty)
	}

	if _, err := ResolvePublishStatsRange("90d", now); err == nil {
		t.Fatal("unsupported range should fail")
	}
}

func TestBuildPublishStatsSnapshotFillsEmptyDaysAcrossYearBoundary(t *testing.T) {
	rangeSpec, _ := ResolvePublishStatsRange("7d", mustDashboardTime(t, "2027-01-02 09:30"))
	rows := []PublishStatsAggregateRow{
		{BucketType: publishStatsBucketDay, Day: "2026-12-27", ActionType: ReviewActionPageView, SessionCount: 4},
		{BucketType: publishStatsBucketDay, Day: "2027-01-02", ActionType: ReviewActionPlatformLinkClick, SessionCount: 2},
		{BucketType: publishStatsBucketRange, ActionType: ReviewActionPageView, SessionCount: 4},
		{BucketType: publishStatsBucketRange, ActionType: ReviewActionPlatformSelect, SessionCount: 3},
		{BucketType: publishStatsBucketRange, ActionType: ReviewActionReviewCopy, SessionCount: 2},
		{BucketType: publishStatsBucketRange, ActionType: ReviewActionPlatformLinkClick, SessionCount: 2},
	}

	snapshot := BuildPublishStatsSnapshot(rangeSpec, rows, PublishStatsHealth{})
	if len(snapshot.DailySeries) != 7 {
		t.Fatalf("daily series length = %d, want 7", len(snapshot.DailySeries))
	}
	if snapshot.DailySeries[0].Date != "2026-12-27" || snapshot.DailySeries[6].Date != "2027-01-02" {
		t.Fatalf("unexpected boundary dates: %#v", snapshot.DailySeries)
	}
	if snapshot.DailySeries[1].PageViews != 0 || snapshot.DailySeries[5].ReviewCopies != 0 {
		t.Fatalf("empty dates should be zero-filled: %#v", snapshot.DailySeries)
	}
	if snapshot.Funnel[0].Count != 4 || snapshot.Funnel[3].Count != 2 {
		t.Fatalf("range totals should use dedicated distinct-session rows: %#v", snapshot.Funnel)
	}
}

func TestBuildPublishStatsSnapshotUsesSafeZeroDenominator(t *testing.T) {
	rangeSpec, _ := ResolvePublishStatsRange("7d", mustDashboardTime(t, "2026-07-15 12:00"))
	rows := []PublishStatsAggregateRow{
		{BucketType: publishStatsBucketRange, ActionType: ReviewActionReviewCopy, SessionCount: 3},
	}

	snapshot := BuildPublishStatsSnapshot(rangeSpec, rows, PublishStatsHealth{})
	if snapshot.Funnel[1].ConversionAvailable || snapshot.Funnel[1].ConversionRate != 0 {
		t.Fatalf("zero denominator must be unavailable and zero: %#v", snapshot.Funnel[1])
	}
	if snapshot.Funnel[2].ConversionAvailable || snapshot.Funnel[2].ConversionRate != 0 {
		t.Fatalf("conversion must not divide by zero: %#v", snapshot.Funnel[2])
	}
}

func TestPublishStatsDataStateBoundaryAtTwentyUniqueSessions(t *testing.T) {
	rangeSpec, _ := ResolvePublishStatsRange("7d", mustDashboardTime(t, "2026-07-15 12:00"))
	rows := []PublishStatsAggregateRow{{BucketType: publishStatsBucketRange, ActionType: ReviewActionPageView, SessionCount: 19}}

	nineteen := BuildPublishStatsSnapshot(rangeSpec, rows, PublishStatsHealth{StoreActive: true, HasLandingRoute: true, HasActivePlatformLink: true, HasReviewInventory: true})
	if nineteen.UniqueSessions != 19 || nineteen.DataState != PublishStatsDataAccumulating {
		t.Fatalf("19 sessions should accumulate: %#v", nineteen)
	}

	rows[0].SessionCount = 20
	twenty := BuildPublishStatsSnapshot(rangeSpec, rows, PublishStatsHealth{StoreActive: true, HasLandingRoute: true, HasActivePlatformLink: true, HasReviewInventory: true})
	if twenty.UniqueSessions != 20 || twenty.DataState != PublishStatsDataReady {
		t.Fatalf("20 sessions should be ready: %#v", twenty)
	}
}

func TestPlatformPublishStatsUsesSelectedSessionsForSampleThreshold(t *testing.T) {
	rangeSpec, _ := ResolvePublishStatsRange("7d", mustDashboardTime(t, "2026-07-15 12:00"))
	rows := []PublishStatsAggregateRow{
		{BucketType: publishStatsBucketRange, ActionType: ReviewActionPageView, SessionCount: 100},
		{BucketType: publishStatsBucketRange, ActionType: ReviewActionPlatformSelect, SessionCount: 3},
		{BucketType: publishStatsBucketRange, ActionType: ReviewActionReviewCopy, SessionCount: 2},
		{BucketType: publishStatsBucketRange, ActionType: ReviewActionPlatformLinkClick, SessionCount: 1},
	}
	health := PublishStatsHealth{StoreActive: true, HasLandingRoute: true, HasActivePlatformLink: true, HasReviewInventory: true}

	snapshot := BuildPublishStatsSnapshot(rangeSpec, rows, health, "meituan")

	if snapshot.UniqueSessions != 3 || snapshot.DataState != PublishStatsDataAccumulating {
		t.Fatalf("platform sample should use selected sessions, got %#v", snapshot)
	}
	if snapshot.Funnel[1].ConversionLabel != "全部访问中选择该平台" {
		t.Fatalf("platform selection should be labelled as share, got %#v", snapshot.Funnel[1])
	}
	if snapshot.Recommendation.Code != PublishStatsRecommendationAccumulating {
		t.Fatalf("small platform sample must not emit a drop conclusion: %#v", snapshot.Recommendation)
	}
}

func TestPlatformPublishStatsDoesNotCallSharedVisitsEmptyWhenNobodySelectedPlatform(t *testing.T) {
	rangeSpec, _ := ResolvePublishStatsRange("7d", mustDashboardTime(t, "2026-07-15 12:00"))
	rows := []PublishStatsAggregateRow{
		{BucketType: publishStatsBucketRange, ActionType: ReviewActionPageView, SessionCount: 100},
	}
	health := PublishStatsHealth{StoreActive: true, HasLandingRoute: true, HasActivePlatformLink: true, HasReviewInventory: true}

	snapshot := BuildPublishStatsSnapshot(rangeSpec, rows, health, "meituan")

	if snapshot.DataState != PublishStatsDataEmpty {
		t.Fatalf("zero platform selections should be empty, got %q", snapshot.DataState)
	}
	if !strings.Contains(snapshot.Recommendation.Title, "选择") || strings.Contains(snapshot.Recommendation.Message, "没有贴卡访问") {
		t.Fatalf("platform empty recommendation must describe no platform selections: %#v", snapshot.Recommendation)
	}
}

func TestPublishStatsRecommendationUsesFixedPriority(t *testing.T) {
	rangeSpec, _ := ResolvePublishStatsRange("7d", mustDashboardTime(t, "2026-07-15 12:00"))
	rows := []PublishStatsAggregateRow{
		{BucketType: publishStatsBucketRange, ActionType: ReviewActionPageView, SessionCount: 30},
		{BucketType: publishStatsBucketRange, ActionType: ReviewActionPlatformSelect, SessionCount: 20},
		{BucketType: publishStatsBucketRange, ActionType: ReviewActionReviewCopy, SessionCount: 10},
		{BucketType: publishStatsBucketRange, ActionType: ReviewActionPlatformLinkClick, SessionCount: 5},
	}

	tests := []struct {
		name   string
		health PublishStatsHealth
		rows   []PublishStatsAggregateRow
		want   string
	}{
		{name: "closed loop outranks every other issue", health: PublishStatsHealth{}, rows: rows, want: PublishStatsRecommendationClosedLoop},
		{name: "content blocker outranks funnel drop", health: PublishStatsHealth{StoreActive: true, HasLandingRoute: true, HasActivePlatformLink: true}, rows: rows, want: PublishStatsRecommendationContent},
		{name: "no visits outranks low sample", health: PublishStatsHealth{StoreActive: true, HasLandingRoute: true, HasActivePlatformLink: true, HasReviewInventory: true}, want: PublishStatsRecommendationNoVisits},
		{name: "funnel drop is selected for enough traffic", health: PublishStatsHealth{StoreActive: true, HasLandingRoute: true, HasActivePlatformLink: true, HasReviewInventory: true}, rows: rows, want: PublishStatsRecommendationFunnelDrop},
		{name: "low sample is the final recommendation", health: PublishStatsHealth{StoreActive: true, HasLandingRoute: true, HasActivePlatformLink: true, HasReviewInventory: true}, rows: []PublishStatsAggregateRow{{BucketType: publishStatsBucketRange, ActionType: ReviewActionPageView, SessionCount: 19}}, want: PublishStatsRecommendationAccumulating},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := BuildPublishStatsSnapshot(rangeSpec, tt.rows, tt.health)
			if snapshot.Recommendation.Code != tt.want {
				t.Fatalf("recommendation = %q, want %q (%#v)", snapshot.Recommendation.Code, tt.want, snapshot.Recommendation)
			}
		})
	}
}

type publishStatsQueryRecorder struct {
	query string
	args  []any
	rows  []PublishStatsAggregateRow
	err   error
	calls int
}

func (r *publishStatsQueryRecorder) Query(query string, args ...any) ([]PublishStatsAggregateRow, error) {
	r.calls++
	r.query = query
	r.args = args
	return r.rows, r.err
}

func TestLoadPublishStatsRowsUsesOneGroupedDistinctSessionQuery(t *testing.T) {
	rangeSpec, _ := ResolvePublishStatsRange("30d", mustDashboardTime(t, "2026-07-15 12:00"))
	runner := &publishStatsQueryRecorder{}

	_, err := LoadPublishStatsRows(runner, 42, "meituan", rangeSpec)
	if err != nil {
		t.Fatalf("load rows: %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("query calls = %d, want 1", runner.calls)
	}
	for _, fragment := range []string{"COUNT(DISTINCT session_id)", "store_id = ?", "platform_code = ?", "created_at >= ?", "created_at < ?", "GROUP BY DATE(created_at), action_type", "UNION ALL"} {
		if !strings.Contains(runner.query, fragment) {
			t.Fatalf("query missing %q:\n%s", fragment, runner.query)
		}
	}
	if strings.Contains(runner.query, " OR ") {
		t.Fatalf("platform query should split shared page views from platform events so each index can be used:\n%s", runner.query)
	}
	if len(runner.args) == 0 {
		t.Fatal("query should bind store, platform, actions, and time boundaries")
	}
}

func TestLoadPublishStatsRowsReturnsDatabaseFailure(t *testing.T) {
	rangeSpec, _ := ResolvePublishStatsRange("7d", mustDashboardTime(t, "2026-07-15 12:00"))
	runner := &publishStatsQueryRecorder{err: errors.New("database unavailable")}

	if _, err := LoadPublishStatsRows(runner, 42, "", rangeSpec); err == nil || !strings.Contains(err.Error(), "database unavailable") {
		t.Fatalf("expected database error, got %v", err)
	}
}
