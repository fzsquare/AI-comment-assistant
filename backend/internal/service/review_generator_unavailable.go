package service

import (
	"fmt"

	"ppk/backend/internal/model"
)

type UnavailableReviewGenerator struct {
	reason string
}

func NewUnavailableReviewGenerator(reason string) *UnavailableReviewGenerator {
	if reason == "" {
		reason = "agent-service is not configured"
	}
	return &UnavailableReviewGenerator{reason: reason}
}

func (g *UnavailableReviewGenerator) Generate(_ model.Store, _ []model.StoreKeyword, _ int) ([]model.ReviewItem, error) {
	return nil, fmt.Errorf("文案 agent 不可用：%s", g.reason)
}
