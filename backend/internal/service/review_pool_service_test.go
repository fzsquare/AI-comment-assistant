package service

import (
	"strings"
	"testing"

	"ppk/backend/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestPendingInitTaskScopeOnlyRecoversQueuedInitialGeneration(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8&parseTime=True&loc=Local",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open dry-run database: %v", err)
	}

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var taskIDs []uint
		return pendingInitTaskScope(tx.Model(&model.ReviewGenerationTask{})).Pluck("id", &taskIDs)
	})

	for _, required := range []string{"trigger_type", model.TriggerInit, "status", model.TaskStatusPending} {
		if !strings.Contains(sql, required) {
			t.Fatalf("recovery SQL %q missing %q", sql, required)
		}
	}
}

func TestClaimPendingGenerationTaskRequiresPendingStatus(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8&parseTime=True&loc=Local",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open dry-run database: %v", err)
	}

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return claimPendingGenerationTask(tx, 42)
	})

	for _, required := range []string{"42", "status", model.TaskStatusPending, model.TaskStatusRunning} {
		if !strings.Contains(sql, required) {
			t.Fatalf("claim SQL %q missing %q", sql, required)
		}
	}
}

func TestGenerationPreferencesFromRowIncludesDiversityDimensions(t *testing.T) {
	prefs := generationPreferencesFromRow(model.StoreGenerationPreference{
		FocusKeywords:       `["香辣蟹","服务热情"]`,
		StyleCodes:          `["natural","detail_rich"]`,
		DiversityDimensions: `["customer_identity","content_angle"]`,
		ReferenceReviews:    `["蟹很入味，服务员会主动帮忙换盘。"]`,
		LengthVariance:      "tight",
	})

	if got := prefs.FocusKeywords; len(got) != 2 || got[0] != "香辣蟹" || got[1] != "服务热情" {
		t.Fatalf("focus keywords got %#v", got)
	}
	if got := prefs.StyleCodes; len(got) != 2 || got[0] != "natural" || got[1] != "detail_rich" {
		t.Fatalf("style codes got %#v", got)
	}
	if got := prefs.DiversityDimensions; len(got) != 2 || got[0] != "customer_identity" || got[1] != "content_angle" {
		t.Fatalf("diversity dimensions got %#v", got)
	}
	if got := prefs.ReferenceReviews; len(got) != 1 || got[0] != "蟹很入味，服务员会主动帮忙换盘。" {
		t.Fatalf("reference reviews got %#v", got)
	}
	if prefs.LengthVariance != "tight" {
		t.Fatalf("length variance got %q, want tight", prefs.LengthVariance)
	}
}

func TestMergeReferenceReviewsPrioritizesPlatformFewShotsAndCapsFive(t *testing.T) {
	got := mergeReferenceReviews(
		[]string{
			" 平台真实评论 A ",
			"平台真实评论 B",
			"平台真实评论 A",
			"",
			"平台真实评论 C",
			"平台真实评论 D",
			"平台真实评论 E",
			"平台真实评论 F",
		},
		[]string{
			"商家手填参考 1",
			"平台真实评论 B",
			"商家手填参考 2",
		},
		5,
	)

	want := []string{
		"平台真实评论 A",
		"平台真实评论 B",
		"平台真实评论 C",
		"平台真实评论 D",
		"平台真实评论 E",
	}
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("item %d got %q, want %q; full=%#v", i, got[i], want[i], got)
		}
	}
}
