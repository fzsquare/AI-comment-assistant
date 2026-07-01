package service

import "ppk/backend/internal/model"

type ReviewGenerator interface {
	Generate(store model.Store, keywords []model.StoreKeyword, targetCount int) ([]model.ReviewItem, error)
}

type ReviewGenerationFeedback struct {
	Accepted []string
	Rejected []string
}

type GenerationPreferences struct {
	FocusKeywords       []string
	StyleCodes          []string
	DiversityDimensions []string
	ReferenceReviews    []string
	LengthVariance      string
}

func (p GenerationPreferences) WithDefaults() GenerationPreferences {
	if p.LengthVariance == "" || p.LengthVariance == "default" {
		p.LengthVariance = "wide"
	}
	return p
}

func (p GenerationPreferences) HasAnyInput() bool {
	return len(p.FocusKeywords) > 0 ||
		len(p.StyleCodes) > 0 ||
		len(p.DiversityDimensions) > 0 ||
		len(p.ReferenceReviews) > 0 ||
		p.LengthVariance != ""
}

type ReviewGenerationContext struct {
	Feedback    ReviewGenerationFeedback
	Preferences GenerationPreferences
}

type FeedbackAwareReviewGenerator interface {
	GenerateWithFeedback(store model.Store, keywords []model.StoreKeyword, feedback ReviewGenerationFeedback, targetCount int) ([]model.ReviewItem, error)
}

type PreferenceAwareReviewGenerator interface {
	GenerateWithContext(store model.Store, keywords []model.StoreKeyword, context ReviewGenerationContext, targetCount int) ([]model.ReviewItem, error)
}
