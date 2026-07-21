package merchant

import (
	"testing"
	"time"

	"ppk/backend/internal/service"
)

func TestNormalizeGenerationPreferencesDefaultsAndDedupes(t *testing.T) {
	prefs, err := normalizeGenerationPreferenceRequest(generationPreferenceRequest{
		FocusKeywords:       []string{" 香辣蟹 ", "服务热情", "香辣蟹", ""},
		StyleCodes:          []string{},
		DiversityDimensions: []string{" customer_identity ", "content_angle", "customer_identity"},
		ReferenceReviews:    []string{"  蟹很入味，服务员会主动帮忙换盘。  "},
		LengthVariance:      "",
	})
	if err != nil {
		t.Fatalf("normalizeGenerationPreferenceRequest returned error: %v", err)
	}
	if got := prefs.FocusKeywords; len(got) != 2 || got[0] != "香辣蟹" || got[1] != "服务热情" {
		t.Fatalf("focus keywords got %#v", got)
	}
	if got := prefs.StyleCodes; len(got) != 1 || got[0] != "natural" {
		t.Fatalf("style codes got %#v, want default natural", got)
	}
	if got := prefs.DiversityDimensions; len(got) != 2 || got[0] != "customer_identity" || got[1] != "content_angle" {
		t.Fatalf("diversity dimensions got %#v", got)
	}
	if got := prefs.ReferenceReviews; len(got) != 1 || got[0] != "蟹很入味，服务员会主动帮忙换盘。" {
		t.Fatalf("reference reviews got %#v", got)
	}
	if prefs.LengthVariance != "wide" {
		t.Fatalf("length variance got %q, want wide", prefs.LengthVariance)
	}
}

func TestNormalizeGenerationPreferencesRejectsTooManyAndTooLong(t *testing.T) {
	tooManyKeywords := generationPreferenceRequest{
		FocusKeywords: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"},
		StyleCodes:    []string{"natural"},
	}
	if _, err := normalizeGenerationPreferenceRequest(tooManyKeywords); err == nil {
		t.Fatal("expected too many focus keywords to be rejected")
	}

	longReview := generationPreferenceRequest{
		StyleCodes:       []string{"natural"},
		ReferenceReviews: []string{repeatRune('好', 301)},
	}
	if _, err := normalizeGenerationPreferenceRequest(longReview); err == nil {
		t.Fatal("expected long reference review to be rejected")
	}

	unknownStyle := generationPreferenceRequest{
		StyleCodes: []string{"salesy"},
	}
	if _, err := normalizeGenerationPreferenceRequest(unknownStyle); err == nil {
		t.Fatal("expected unknown style code to be rejected")
	}

	unknownDimension := generationPreferenceRequest{
		StyleCodes:          []string{"natural"},
		DiversityDimensions: []string{"fake_identity"},
	}
	if _, err := normalizeGenerationPreferenceRequest(unknownDimension); err == nil {
		t.Fatal("expected unknown diversity dimension to be rejected")
	}
}

func repeatRune(r rune, n int) string {
	out := make([]rune, n)
	for i := range out {
		out[i] = r
	}
	return string(out)
}

func TestSummarizePlatformContentHealthRequiresEveryEnabledPlatform(t *testing.T) {
	hasInventory, generationFailed := summarizePlatformContentHealth(
		[]string{"meituan", "dianping"},
		map[string]int64{"meituan": 12},
		map[string]string{"meituan": "success", "dianping": "failed"},
	)

	if hasInventory {
		t.Fatal("one stocked platform must not hide another enabled platform with no inventory")
	}
	if !generationFailed {
		t.Fatal("a failed latest task on any enabled platform should be surfaced")
	}
}

type legacyStatsCounterFunc func(actionType string, platformCode string, start time.Time, end time.Time) (int64, error)

func (fn legacyStatsCounterFunc) Count(actionType string, platformCode string, start time.Time, end time.Time) (int64, error) {
	return fn(actionType, platformCode, start, end)
}

func TestLoadLegacyPublishStatsKeepsAllTimeTotalsAndTwelvePeriodSeries(t *testing.T) {
	counter := legacyStatsCounterFunc(func(actionType string, platformCode string, start time.Time, end time.Time) (int64, error) {
		if start.IsZero() && end.IsZero() {
			switch actionType {
			case service.ReviewActionPlatformLinkClick:
				return 900, nil
			case service.ReviewActionPlatformSelect:
				return 400, nil
			}
		}
		return 7, nil
	})
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, dashboardLocation())

	legacy, err := loadLegacyPublishStats(counter, "meituan", now)
	if err != nil {
		t.Fatalf("load legacy stats: %v", err)
	}
	if legacy.totalClicks != 900 || legacy.totalVisits != 400 {
		t.Fatalf("legacy all-time totals changed semantics: %#v", legacy)
	}
	if len(legacy.weeklySeries) != 12 || len(legacy.monthlySeries) != 12 {
		t.Fatalf("legacy series must retain 12 periods: weeks=%d months=%d", len(legacy.weeklySeries), len(legacy.monthlySeries))
	}
}
