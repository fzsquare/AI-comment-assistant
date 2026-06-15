package service

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"ppk/backend/internal/model"
)

type ReviewGenerator interface {
	Generate(store model.Store, keywords []model.StoreKeyword, targetCount int) ([]model.ReviewItem, error)
}

type MockReviewGenerator struct{}

func (g *MockReviewGenerator) Generate(store model.Store, keywords []model.StoreKeyword, targetCount int) ([]model.ReviewItem, error) {
	rand.Seed(time.Now().UnixNano())
	prefixes := []string{"这次体验很不错", "整体感受挺好", "来店里体验以后感觉很好", "这家店值得推荐"}
	suffixes := []string{"下次还会再来。", "整体很满意。", "适合朋友一起来。", "体验自然又舒服。"}
	batch := fmt.Sprintf("batch_%d", time.Now().UnixNano())

	keywordTexts := make([]string, 0, len(keywords))
	for _, item := range keywords {
		keywordTexts = append(keywordTexts, item.Keyword)
	}
	joined := strings.Join(keywordTexts, "、")
	if joined == "" {
		joined = "服务、环境、体验"
	}

	result := make([]model.ReviewItem, 0, targetCount)
	for i := 0; i < targetCount; i++ {
		content := fmt.Sprintf("%s，%s的%s给人印象挺深，%s", prefixes[rand.Intn(len(prefixes))], store.StoreName, joined, suffixes[rand.Intn(len(suffixes))])
		result = append(result, model.ReviewItem{
			StoreID:           store.ID,
			PlatformStyle:     store.PrimaryPlatformStyle,
			Content:           content,
			SourceType:        "ai",
			GenerationBatchNo: batch,
			IsDispatched:      false,
			Status:            model.ReviewStatusAvailable,
		})
	}

	return result, nil
}
