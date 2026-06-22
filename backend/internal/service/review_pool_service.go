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
	// Fallback 在评价池为空且主生成器（agent）不可用时，即时产出一条兜底文案，
	// 保证落地页不白屏。通常注入内置 MockReviewGenerator。
	Fallback ReviewGenerator
}

type LandingPayload struct {
	SessionID                  string                    `json:"sessionId"`
	StoreName                  string                    `json:"storeName"`
	PrimaryPlatformStyle       string                    `json:"primaryPlatformStyle"`
	Review                     model.ReviewItem          `json:"review"`
	Keywords                   []model.StoreKeyword      `json:"keywords"` // 顾客可选的菜品/场景 chips
	Images                     []model.StoreImage        `json:"images"`
	PlatformLinks              []model.StorePlatformLink `json:"platformLinks"`
	RemainingDispatchableCount int64                     `json:"remainingDispatchableCount"`
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

// dispatchOne 从池中取一条可发放评价并标记已发放。
// tag 非空时优先取带该标签的评价，无匹配则回退取任意一条。
//
// 并发安全：用「条件更新 + 重试」而非 select-then-save，避免两个并发请求
// 选中同一行后双双发放（双发）。UPDATE ... WHERE is_dispatched=false 是原子的，
// RowsAffected==0 说明被别的请求抢先发放了，重新选一条。
func (s *ReviewPoolService) dispatchOne(storeID uint, tag string) (*model.ReviewItem, error) {
	const maxAttempts = 5
	// base 每次重建：GORM 会就地改写 *gorm.DB，链式条件不能复用
	base := func() *gorm.DB {
		return s.DB.Where("store_id = ? AND status = ? AND is_dispatched = ?",
			storeID, model.ReviewStatusAvailable, false)
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		var review model.ReviewItem
		found := false
		if tag != "" {
			e := base().Where("tags LIKE ?", "%"+tag+"%").Order("RAND()").First(&review).Error
			if e == nil {
				found = true
			} else if !errors.Is(e, gorm.ErrRecordNotFound) {
				return nil, e // 真实 DB 错误，不当作“无匹配”吞掉
			}
		}
		if !found {
			if e := base().Order("RAND()").First(&review).Error; e != nil {
				return nil, e // 含 ErrRecordNotFound（池空），交由上层兜底
			}
		}

		now := time.Now()
		res := s.DB.Model(&model.ReviewItem{}).
			Where("id = ? AND is_dispatched = ?", review.ID, false).
			Updates(map[string]any{"is_dispatched": true, "dispatched_at": now})
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 1 {
			review.IsDispatched = true
			review.DispatchedAt = &now
			return &review, nil
		}
		// RowsAffected==0：被并发抢走，重试
	}
	// 高并发下连续抢空，等价于暂时取不到，交由上层兜底
	return nil, gorm.ErrRecordNotFound
}

// ensureViaFallback 在池为空时用 Mock 即时补 1 条可发放文案，保证不白屏。
func (s *ReviewPoolService) ensureViaFallback(store model.Store) error {
	if s.Fallback == nil {
		return errors.New("无兜底生成器")
	}
	var keywords []model.StoreKeyword
	s.DB.Where("store_id = ?", store.ID).Find(&keywords)
	// 只补 1 条：仅为不白屏兜底，避免把低质 Mock 文案大量混入池子
	reviews, err := s.Fallback.Generate(store, keywords, 1)
	if err != nil || len(reviews) == 0 {
		return errors.New("兜底生成失败")
	}
	for i := range reviews {
		reviews[i].StoreID = store.ID
		reviews[i].PlatformStyle = store.PrimaryPlatformStyle
		reviews[i].SourceType = "fallback"
		reviews[i].GenerationBatchNo = "fallback_" + utils.RandomString(8)
		reviews[i].Status = model.ReviewStatusAvailable
	}
	return s.DB.Create(&reviews).Error
}

// dispatchWithFallback 先正常取；池空则兜底补一条再取一次。
func (s *ReviewPoolService) dispatchWithFallback(store model.Store, tag string) (*model.ReviewItem, error) {
	review, err := s.dispatchOne(store.ID, tag)
	if err == nil {
		return review, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if fbErr := s.ensureViaFallback(store); fbErr == nil {
			return s.dispatchOne(store.ID, tag)
		}
	}
	return nil, errors.New("暂无推荐文案，请稍后再试")
}

func (s *ReviewPoolService) activeStoreByID(storeID uint) (*model.Store, error) {
	var store model.Store
	err := s.DB.Table("stores").
		Select("stores.*").
		Joins("JOIN merchant_users ON merchant_users.id = stores.merchant_user_id").
		Where("stores.id = ? AND stores.status = ? AND merchant_users.status = ?", storeID, model.StatusEnabled, model.StatusEnabled).
		First(&store).Error
	return &store, err
}

func (s *ReviewPoolService) DispatchByLandingToken(token string) (*LandingPayload, error) {
	var tag model.NFCTag
	if err := s.DB.Where("landing_token = ? AND status = ?", token, model.TagStatusBound).First(&tag).Error; err != nil {
		return nil, errors.New("页面暂不可用")
	}

	store, err := s.activeStoreByID(tag.StoreID)
	if err != nil {
		return nil, errors.New("商家服务已暂停")
	}

	review, err := s.dispatchWithFallback(*store, "")
	if err != nil {
		return nil, err
	}

	var keywords []model.StoreKeyword
	s.DB.Where("store_id = ?", store.ID).Order("sort_no asc, id asc").Find(&keywords)

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
		SessionID:                  utils.RandomString(16),
		StoreName:                  store.StoreName,
		PrimaryPlatformStyle:       store.PrimaryPlatformStyle,
		Review:                     *review,
		Keywords:                   keywords,
		Images:                     images,
		PlatformLinks:              links,
		RemainingDispatchableCount: remaining,
	}, nil
}

// SwitchReview 换一换；tag 非空时按顾客选择的菜品/场景标签取。
func (s *ReviewPoolService) SwitchReview(token string, tag string) (*model.ReviewItem, int64, error) {
	var tagRow model.NFCTag
	if err := s.DB.Where("landing_token = ? AND status = ?", token, model.TagStatusBound).First(&tagRow).Error; err != nil {
		return nil, 0, errors.New("页面暂不可用")
	}

	store, err := s.activeStoreByID(tagRow.StoreID)
	if err != nil {
		return nil, 0, errors.New("商家服务已暂停")
	}

	review, err := s.dispatchWithFallback(*store, tag)
	if err != nil {
		return nil, 0, err
	}

	var remaining int64
	s.DB.Model(&model.ReviewItem{}).
		Where("store_id = ? AND status = ? AND is_dispatched = ?", store.ID, model.ReviewStatusAvailable, false).
		Count(&remaining)
	_ = s.EnsureDispatchableStock(store.ID)
	return review, remaining, nil
}
