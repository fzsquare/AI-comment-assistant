package service

import "ppk/backend/internal/model"

type ReviewGenerator interface {
	Generate(store model.Store, keywords []model.StoreKeyword, targetCount int) ([]model.ReviewItem, error)
}

type ReviewGenerationFeedback struct {
	Accepted []string
	Rejected []string
}

type FeedbackAwareReviewGenerator interface {
	GenerateWithFeedback(store model.Store, keywords []model.StoreKeyword, feedback ReviewGenerationFeedback, targetCount int) ([]model.ReviewItem, error)
}
