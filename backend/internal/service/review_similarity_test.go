package service

import "testing"

func TestReviewTextsHighSimilarityMatchesEditedPlatformReview(t *testing.T) {
	left := "七欣天皇岗店环境干净舒适，海鲜鲜活现点，招牌波格力蟹酱汁浓郁，服务响应及时。"
	right := "七欣天皇岗店环境干净舒适，海鲜鲜活现点。招牌波格力蟹酱汁浓郁醇厚，服务响应很及时。"

	result := CompareReviewText(left, right)
	if !result.Matched {
		t.Fatalf("expected high similarity match, got %+v", result)
	}
	if result.Score < 0.86 {
		t.Fatalf("score got %.2f, want >= 0.86", result.Score)
	}
	if result.Reason == "" {
		t.Fatal("expected reason to explain match")
	}
}

func TestReviewTextsHighSimilarityRejectsShortLooseMatch(t *testing.T) {
	result := CompareReviewText("不错", "不错不错")
	if result.Matched {
		t.Fatalf("short loose text should not match, got %+v", result)
	}
}

func TestReviewTextsHighSimilarityAllowsExactShortMatch(t *testing.T) {
	result := CompareReviewText("不错", "不错")
	if !result.Matched {
		t.Fatalf("exact short text should match, got %+v", result)
	}
	if result.Reason != "exact" {
		t.Fatalf("reason got %q, want exact", result.Reason)
	}
}
