package service

import (
	"strings"

	"ppk/backend/internal/model"
)

func filterDuplicateGeneratedReviews(generated []model.ReviewItem, existingContents []string) ([]model.ReviewItem, int) {
	seen := make([]string, 0, len(existingContents)+len(generated))
	for _, content := range existingContents {
		content = strings.TrimSpace(content)
		if content != "" {
			seen = append(seen, content)
		}
	}

	filtered := make([]model.ReviewItem, 0, len(generated))
	duplicates := 0
	for _, item := range generated {
		content := strings.TrimSpace(item.Content)
		if content == "" || hasSimilarReviewText(content, seen) {
			duplicates++
			continue
		}
		filtered = append(filtered, item)
		seen = append(seen, content)
	}
	return filtered, duplicates
}

func hasSimilarReviewText(content string, candidates []string) bool {
	for _, candidate := range candidates {
		if CompareReviewText(content, candidate).Matched {
			return true
		}
	}
	return false
}
