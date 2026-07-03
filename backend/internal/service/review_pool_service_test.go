package service

import (
	"testing"

	"ppk/backend/internal/model"
)

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
