package service

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/utils"

	"gorm.io/gorm"
)

type ReviewPoolService struct {
	DB        *gorm.DB
	Generator ReviewGenerator

	refillMu       sync.Mutex
	refillInFlight map[string]struct{}
}

type LandingPayload struct {
	SessionID                  string                    `json:"sessionId"`
	StoreName                  string                    `json:"storeName"`
	PrimaryPlatformStyle       string                    `json:"primaryPlatformStyle"`
	Review                     *model.ReviewItem         `json:"review,omitempty"`
	Keywords                   []model.StoreKeyword      `json:"keywords"` // 顾客可选的菜品/场景 chips
	Images                     []model.StoreImage        `json:"images"`
	PlatformLinks              []model.StorePlatformLink `json:"platformLinks"`
	RemainingDispatchableCount int64                     `json:"remainingDispatchableCount"`
}

func (s *ReviewPoolService) GenerateForStore(storeID uint, triggerType string, targetCount int) error {
	return s.GenerateForStorePlatform(storeID, "", triggerType, targetCount)
}

func (s *ReviewPoolService) GenerateForStorePlatform(storeID uint, platformCode string, triggerType string, targetCount int) error {
	var store model.Store
	if err := s.DB.First(&store, storeID).Error; err != nil {
		return err
	}
	platformCode = normalizePlatformCode(platformCode, store.PrimaryPlatformStyle)
	store.PrimaryPlatformStyle = platformCode

	var keywords []model.StoreKeyword
	s.DB.Where("store_id = ?", storeID).Order("sort_no asc, id asc").Find(&keywords)

	task := model.ReviewGenerationTask{
		StoreID:       storeID,
		PlatformStyle: platformCode,
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
		reviews[i].PlatformStyle = platformCode
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

func (s *ReviewPoolService) EnsureDispatchableStock(storeID uint, platformCode string) error {
	if platformCode == "" {
		return errors.New("platformCode is required")
	}
	var count int64
	if err := s.DB.Model(&model.ReviewItem{}).
		Where("store_id = ? AND platform_style = ? AND status = ? AND is_dispatched = ?", storeID, platformCode, model.ReviewStatusAvailable, false).
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
	return s.GenerateForStorePlatform(storeID, platformCode, model.TriggerAutoRefill, target)
}

func (s *ReviewPoolService) EnsureDispatchableStockAsync(storeID uint, platformCode string) {
	if storeID == 0 || platformCode == "" {
		return
	}
	key := refillKey(storeID, platformCode)

	s.refillMu.Lock()
	if s.refillInFlight == nil {
		s.refillInFlight = make(map[string]struct{})
	}
	if _, ok := s.refillInFlight[key]; ok {
		s.refillMu.Unlock()
		return
	}
	s.refillInFlight[key] = struct{}{}
	s.refillMu.Unlock()

	go func() {
		defer func() {
			s.refillMu.Lock()
			delete(s.refillInFlight, key)
			s.refillMu.Unlock()
		}()
		_ = s.EnsureDispatchableStock(storeID, platformCode)
	}()
}

// dispatchOne 从池中取一条可发放评价并标记已发放。
// tag 非空时优先取带该标签的评价，无匹配则回退取任意一条。
//
// 并发安全：用「条件更新 + 重试」而非 select-then-save，避免两个并发请求
// 选中同一行后双双发放（双发）。UPDATE ... WHERE is_dispatched=false 是原子的，
// RowsAffected==0 说明被别的请求抢先发放了，重新选一条。
func (s *ReviewPoolService) dispatchOne(storeID uint, platformCode string, tag string) (*model.ReviewItem, error) {
	const maxAttempts = 5
	// base 每次重建：GORM 会就地改写 *gorm.DB，链式条件不能复用
	base := func() *gorm.DB {
		return s.DB.Where("store_id = ? AND platform_style = ? AND status = ? AND is_dispatched = ?",
			storeID, platformCode, model.ReviewStatusAvailable, false)
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

func (s *ReviewPoolService) dispatchAvailable(store model.Store, platformCode string, tag string) (*model.ReviewItem, error) {
	review, err := s.dispatchOne(store.ID, platformCode, tag)
	if err == nil {
		return review, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("暂无可用评论，请先在商家后台使用 AI agent 生成评价")
	}
	return nil, err
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

// activeStoreByUUID 按门店 uuid 取启用中的门店（且商家启用）。落地入口的唯一映射。
func (s *ReviewPoolService) activeStoreByUUID(uuid string) (*model.Store, error) {
	var store model.Store
	err := s.DB.Table("stores").
		Select("stores.*").
		Joins("JOIN merchant_users ON merchant_users.id = stores.merchant_user_id").
		Where("stores.uuid = ? AND stores.status = ? AND merchant_users.status = ?", uuid, model.StatusEnabled, model.StatusEnabled).
		First(&store).Error
	return &store, err
}

func (s *ReviewPoolService) InitLandingByUUID(uuid string) (*LandingPayload, error) {
	store, err := s.activeStoreByUUID(uuid)
	if err != nil {
		return nil, errors.New("页面暂不可用")
	}

	var keywords []model.StoreKeyword
	s.DB.Where("store_id = ?", store.ID).Order("sort_no asc, id asc").Find(&keywords)

	var images []model.StoreImage
	s.DB.Where("store_id = ? AND status = ?", store.ID, model.StatusEnabled).Order("sort_no asc, id asc").Limit(3).Find(&images)

	var links []model.StorePlatformLink
	s.DB.Where("store_id = ? AND status = ?", store.ID, model.StatusEnabled).Order("sort_no asc, id asc").Find(&links)

	return &LandingPayload{
		SessionID:                  utils.RandomString(16),
		StoreName:                  store.StoreName,
		PrimaryPlatformStyle:       store.PrimaryPlatformStyle,
		Keywords:                   keywords,
		Images:                     images,
		PlatformLinks:              links,
		RemainingDispatchableCount: 0,
	}, nil
}

// SwitchReview 换一换；tag 非空时按顾客选择的菜品/场景标签取。
func (s *ReviewPoolService) SwitchReview(uuid string, platformCode string, tag string) (*model.ReviewItem, int64, error) {
	store, err := s.activeStoreByUUID(uuid)
	if err != nil {
		return nil, 0, errors.New("页面暂不可用")
	}
	platformCode = normalizePlatformCode(platformCode, "")
	if err := s.ensureActivePlatformLink(store.ID, platformCode); err != nil {
		return nil, 0, err
	}

	review, err := s.dispatchAvailable(*store, platformCode, tag)
	if err != nil {
		return nil, 0, err
	}

	var remaining int64
	s.DB.Model(&model.ReviewItem{}).
		Where("store_id = ? AND platform_style = ? AND status = ? AND is_dispatched = ?", store.ID, platformCode, model.ReviewStatusAvailable, false).
		Count(&remaining)
	s.EnsureDispatchableStockAsync(store.ID, platformCode)
	return review, remaining, nil
}

func (s *ReviewPoolService) ensureActivePlatformLink(storeID uint, platformCode string) error {
	if platformCode == "" {
		return errors.New("请选择评价平台")
	}
	var link model.StorePlatformLink
	if err := s.DB.Where("store_id = ? AND platform_code = ? AND status = ?", storeID, platformCode, model.StatusEnabled).First(&link).Error; err != nil {
		return errors.New("评价平台不可用")
	}
	return nil
}

func normalizePlatformCode(platformCode string, fallback string) string {
	platformCode = strings.TrimSpace(platformCode)
	if platformCode == "" {
		return strings.TrimSpace(fallback)
	}
	return platformCode
}

func refillKey(storeID uint, platformCode string) string {
	return fmt.Sprintf("%d:%s", storeID, platformCode)
}
