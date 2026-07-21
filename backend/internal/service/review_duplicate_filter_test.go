package service

import (
	"testing"

	"ppk/backend/internal/model"
)

func TestFilterDuplicateGeneratedReviewsRejectsExistingSimilarText(t *testing.T) {
	generated := []model.ReviewItem{
		{Content: "七欣天皇岗店环境干净舒适，海鲜鲜活现点，服务响应及时。"},
		{Content: "第一次来吃波格力蟹，味道挺惊喜，适合朋友小聚。"},
	}
	existing := []string{
		"七欣天皇岗店环境干净舒适，海鲜鲜活现点。服务响应很及时。",
	}

	filtered, duplicateCount := filterDuplicateGeneratedReviews(generated, existing)

	if duplicateCount != 1 {
		t.Fatalf("duplicate count got %d, want 1", duplicateCount)
	}
	if len(filtered) != 1 {
		t.Fatalf("filtered length got %d, want 1", len(filtered))
	}
	if filtered[0].Content != generated[1].Content {
		t.Fatalf("kept content got %q, want second generated review", filtered[0].Content)
	}
}

func TestFilterDuplicateGeneratedReviewsChecksWithinNewBatch(t *testing.T) {
	generated := []model.ReviewItem{
		{Content: "朋友推荐来的，菜品和环境都比较稳定，适合周末聚餐。"},
		{Content: "朋友推荐来的，菜品和环境都比较稳定，适合周末聚餐。整体挺满意。"},
	}

	filtered, duplicateCount := filterDuplicateGeneratedReviews(generated, nil)

	if duplicateCount != 1 {
		t.Fatalf("duplicate count got %d, want 1", duplicateCount)
	}
	if len(filtered) != 1 {
		t.Fatalf("filtered length got %d, want 1", len(filtered))
	}
}
