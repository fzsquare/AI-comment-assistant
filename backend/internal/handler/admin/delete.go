package admin

import (
	"net/http"

	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func storeDeletionPlan(includeMerchant bool) []string {
	steps := []string{
		"unbind_nfc_tags",
		"delete_review_feedbacks",
		"delete_review_display_logs",
		"delete_review_generation_audit_logs",
		"delete_review_generation_tasks",
		"delete_platform_review_few_shots",
		"delete_external_store_reviews",
		"delete_review_crawl_batches",
		"delete_review_crawl_configs",
		"delete_store_generation_preferences",
		"delete_review_items",
		"delete_store_platform_links",
		"delete_store_images",
		"delete_store_keywords",
		"delete_store",
	}
	if includeMerchant {
		steps = append(steps, "delete_merchant")
	}
	return steps
}

func executeStoreDeletionPlan(tx *gorm.DB, store model.Store, includeMerchant bool) error {
	for _, step := range storeDeletionPlan(includeMerchant) {
		var err error
		switch step {
		case "unbind_nfc_tags":
			err = tx.Model(&model.NFCTag{}).
				Where("store_id = ?", store.ID).
				Updates(map[string]interface{}{
					"store_id":      nil,
					"landing_token": "",
					"status":        model.TagStatusUnbound,
				}).Error
		case "delete_review_display_logs":
			err = tx.Where("store_id = ?", store.ID).Delete(&model.ReviewDisplayLog{}).Error
		case "delete_review_feedbacks":
			err = tx.Where("store_id = ?", store.ID).Delete(&model.ReviewFeedback{}).Error
		case "delete_review_generation_tasks":
			err = tx.Where("store_id = ?", store.ID).Delete(&model.ReviewGenerationTask{}).Error
		case "delete_review_generation_audit_logs":
			err = tx.Where("store_id = ?", store.ID).Delete(&model.ReviewGenerationAuditLog{}).Error
		case "delete_platform_review_few_shots":
			err = tx.Where("store_id = ?", store.ID).Delete(&model.PlatformReviewFewShot{}).Error
		case "delete_external_store_reviews":
			err = tx.Where("store_id = ?", store.ID).Delete(&model.ExternalStoreReview{}).Error
		case "delete_review_crawl_batches":
			err = tx.Where("store_id = ?", store.ID).Delete(&model.StoreReviewCrawlBatch{}).Error
		case "delete_review_crawl_configs":
			err = tx.Where("store_id = ?", store.ID).Delete(&model.StoreReviewCrawlConfig{}).Error
		case "delete_store_generation_preferences":
			err = tx.Where("store_id = ?", store.ID).Delete(&model.StoreGenerationPreference{}).Error
		case "delete_review_items":
			err = tx.Where("store_id = ?", store.ID).Delete(&model.ReviewItem{}).Error
		case "delete_store_platform_links":
			err = tx.Where("store_id = ?", store.ID).Delete(&model.StorePlatformLink{}).Error
		case "delete_store_images":
			err = tx.Where("store_id = ?", store.ID).Delete(&model.StoreImage{}).Error
		case "delete_store_keywords":
			err = tx.Where("store_id = ?", store.ID).Delete(&model.StoreKeyword{}).Error
		case "delete_store":
			err = tx.Delete(&model.Store{}, store.ID).Error
		case "delete_merchant":
			err = tx.Delete(&model.MerchantUser{}, store.MerchantUserID).Error
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) deleteStore(c *gin.Context) {
	var store model.Store
	if err := h.DB.First(&store, uintParam(c)).Error; err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		return executeStoreDeletionPlan(tx, store, true)
	}); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败")
		return
	}

	response.Success(c, gin.H{"deleted": true})
}

func (h *Handler) deleteMerchant(c *gin.Context) {
	var merchant model.MerchantUser
	if err := h.DB.First(&merchant, uintParam(c)).Error; err != nil {
		response.Error(c, http.StatusNotFound, "商家不存在")
		return
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		var stores []model.Store
		if err := tx.Where("merchant_user_id = ?", merchant.ID).Find(&stores).Error; err != nil {
			return err
		}
		for _, store := range stores {
			if err := executeStoreDeletionPlan(tx, store, false); err != nil {
				return err
			}
		}
		return tx.Delete(&model.MerchantUser{}, merchant.ID).Error
	}); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败")
		return
	}

	response.Success(c, gin.H{"deleted": true})
}
