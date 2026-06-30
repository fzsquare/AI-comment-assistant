package service

import "ppk/backend/internal/model"

type ReviewGenerator interface {
	Generate(store model.Store, keywords []model.StoreKeyword, targetCount int) ([]model.ReviewItem, error)
}
