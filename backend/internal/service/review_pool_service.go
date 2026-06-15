package service

import (
	"errors"
	"fmt"
	"time"

	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/utils"

	"gorm.io/gorm"
)

type ReviewPoolService struct {
	DB        *gorm.DB
	Generator ReviewGenerator
}

type LandingPayload struct {
	SessionID                 string                    `json:"sessionId"`
	StoreName                 string                    `json:"storeName"`
	PrimaryPlatformStyle      string                    `json:"primaryPlatformStyle"`
	Review                    model.ReviewItem          `json:"review"`
	Images                    []model.StoreImage        `json:"images"`
	PlatformLinks             []model.StorePlatformLink `json:"platformLinks"`
	RemainingDispatchableCount int64                    `json:"remainingDispatchableCount"`
}

func (s *ReviewPoolService) GenerateForStore(storeID uint, triggerType string, targetCount int) error {
	var store model.Store
	if err := s.DB.First(&store, storeID).Error; err != nil {
		return err
	}

	var keywords []model.StoreKeyword
	s.DB.Where("store_id = ?", storeID).Order("sort_no asc, id asc").Find(&keywords)

	task := model.ReviewGenerationTask{
		StoreID:       storeID,
		PlatformStyle: store.PrimaryPlatformStyle,
		TriggerType:   triggerType,
		TargetCount:   targetCount,
		Status:        model.TaskStatusRunning,
	}
	if err := s.DB.Create(&task).Error; err != nil {
		return err
	}

	reviews, err := s.Generator.Generate(store, keywords, targetCount)
	if err != nil {
		task.Status = model.TaskStatusFailed
		task.ErrorMessage = err.Error()
		s.DB.Save(&task)
		return err
	}

	if len(reviews) == 0 {
		task.Status = model.TaskStatusFailed
		task.ErrorMessage = "no reviews generated"
		s.DB.Save(&task)
		return errors.New("no reviews generated")
	}

	for i := range reviews {
		reviews[i].StoreID = storeID
		reviews[i].PlatformStyle = store.PrimaryPlatformStyle
		if reviews[i].GenerationBatchNo == "" {
			reviews[i].GenerationBatchNo = fmt.Sprintf("batch_%s", utils.RandomString(12))
		}
		if reviews[i].Status == "" {
			reviews[i].Status = model.ReviewStatusAvailable
		}
	}

	if err := s.DB.Create(&reviews).Error; err != nil {
		task.Status = model.TaskStatusFailed
		task.ErrorMessage = err.Error()
		s.DB.Save(&task)
		return err
	}

	task.SuccessCount = len(reviews)
	task.Status = model.TaskStatusSuccess
	return s.DB.Save(&task).Error
}

func (s *ReviewPoolService) EnsureDispatchableStock(storeID uint) error {
	var count int64
	if err := s.DB.Model(&model.ReviewItem{}).
		Where("store_id = ? AND status = ? AND is_dispatched = ?", storeID, model.ReviewStatusAvailable, false).
		Count(&count).Error; err != nil {
		return err
	}
	if count >= 20 {
		return nil
	}
	target := int(50 - count)
	if target <= 0 {
		return nil
	}
	return s.GenerateForStore(storeID, model.TriggerAutoRefill, target)
}

func (s *ReviewPoolService) DispatchByLandingToken(token string) (*LandingPayload, error) {
	var tag model.NFCTag
	if err := s.DB.Where("landing_token = ? AND status = ?", token, model.TagStatusBound).First(&tag).Error; err != nil {
		return nil, errors.New("页面暂不可用")
	}

	var store model.Store
	if err := s.DB.Where("id = ? AND status = ?", tag.StoreID, model.StatusEnabled).First(&store).Error; err != nil {
		return nil, errors.New("商家服务已暂停")
	}

	var review model.ReviewItem
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("store_id = ? AND status = ? AND is_dispatched = ?", store.ID, model.ReviewStatusAvailable, false).
			Order("RAND()").First(&review).Error; err != nil {
			return err
		}
		now := time.Now()
		review.IsDispatched = true
		review.DispatchedAt = &now
		return tx.Save(&review).Error
	})
	if err != nil {
		return nil, errors.New("暂无推荐文案，请稍后再试")
	}

	var images []model.StoreImage
	s.DB.Where("store_id = ? AND status = ?", store.ID, model.StatusEnabled).Order("sort_no asc, id asc").Limit(3).Find(&images)

	var links []model.StorePlatformLink
	s.DB.Where("store_id = ? AND status = ?", store.ID, model.StatusEnabled).Order("sort_no asc, id asc").Find(&links)

	var remaining int64
	s.DB.Model(&model.ReviewItem{}).
		Where("store_id = ? AND status = ? AND is_dispatched = ?", store.ID, model.ReviewStatusAvailable, false).
		Count(&remaining)

	_ = s.DB.Create(&model.ReviewDisplayLog{StoreID: store.ID, ReviewItemID: review.ID, NFCTagID: tag.ID, SessionID: utils.RandomString(16), ActionType: "review_show"}).Error
	_ = s.EnsureDispatchableStock(store.ID)

	return &LandingPayload{
		SessionID:                 utils.RandomString(16),
		StoreName:                 store.StoreName,
		PrimaryPlatformStyle:      store.PrimaryPlatformStyle,
		Review:                    review,
		Images:                    images,
		PlatformLinks:             links,
		RemainingDispatchableCount: remaining,
	}, nil
}

func (s *ReviewPoolService) SwitchReview(token string) (*model.ReviewItem, int64, error) {
	var tag model.NFCTag
	if err := s.DB.Where("landing_token = ? AND status = ?", token, model.TagStatusBound).First(&tag).Error; err != nil {
		return nil, 0, errors.New("页面暂不可用")
	}

	var review model.ReviewItem
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("store_id = ? AND status = ? AND is_dispatched = ?", tag.StoreID, model.ReviewStatusAvailable, false).
			Order("RAND()").First(&review).Error; err != nil {
			return err
		}
		now := time.Now()
		review.IsDispatched = true
		review.DispatchedAt = &now
		return tx.Save(&review).Error
	})
	if err != nil {
		return nil, 0, errors.New("暂无推荐文案，请稍后再试")
	}

	var remaining int64
	s.DB.Model(&model.ReviewItem{}).
		Where("store_id = ? AND status = ? AND is_dispatched = ?", tag.StoreID, model.ReviewStatusAvailable, false).
		Count(&remaining)
	_ = s.EnsureDispatchableStock(tag.StoreID)
	return &review, remaining, nil
}
