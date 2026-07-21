package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type platformReviewLibraryFilter struct {
	StoreID      uint
	PlatformCode string
	Search       string
	SelectedOnly bool
	Limit        int
	Offset       int
}

type platformReviewLibraryResponse struct {
	Items         []platformReviewLibraryItem `json:"items"`
	Total         int64                       `json:"total"`
	SelectedCount int64                       `json:"selectedCount"`
	Limit         int                         `json:"limit"`
	Offset        int                         `json:"offset"`
}

type platformReviewLibraryItem struct {
	ID               uint       `json:"id"`
	BatchID          uint       `json:"batchId"`
	StoreID          uint       `json:"storeId"`
	StoreName        string     `json:"storeName"`
	PlatformCode     string     `json:"platformCode"`
	SourceReviewRef  string     `json:"sourceReviewRef"`
	UserName         string     `json:"userName"`
	RatingRaw        string     `json:"ratingRaw"`
	RatingNormalized *float64   `json:"ratingNormalized"`
	ReviewTime       *time.Time `json:"reviewTime"`
	Content          string     `json:"content"`
	IsBaseline       bool       `json:"isBaseline"`
	IsFewShot        bool       `json:"isFewShot"`
	SelectedAt       *time.Time `json:"selectedAt"`
	CreatedAt        time.Time  `json:"createdAt"`
}

func (h *Handler) listPlatformReviews(c *gin.Context) {
	filter := parsePlatformReviewLibraryFilter(c)

	var total int64
	if err := h.platformReviewLibraryQuery(filter).Count(&total).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "平台评论库加载失败")
		return
	}

	selectedFilter := filter
	selectedFilter.SelectedOnly = true
	var selectedCount int64
	if err := h.platformReviewLibraryQuery(selectedFilter).Count(&selectedCount).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "平台评论库选择状态加载失败")
		return
	}

	var items []platformReviewLibraryItem
	if err := h.platformReviewLibraryQuery(filter).
		Select(`external_store_reviews.id,
			external_store_reviews.batch_id,
			external_store_reviews.store_id,
			stores.store_name,
			external_store_reviews.platform_code,
			external_store_reviews.source_review_ref,
			external_store_reviews.user_name,
			external_store_reviews.rating_raw,
			external_store_reviews.rating_normalized,
			external_store_reviews.review_time,
			external_store_reviews.content,
			external_store_reviews.is_baseline,
			CASE WHEN platform_review_few_shots.id IS NULL THEN 0 ELSE 1 END AS is_few_shot,
			platform_review_few_shots.selected_at,
			external_store_reviews.created_at`).
		Order("is_few_shot desc, external_store_reviews.review_time desc, external_store_reviews.id desc").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&items).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "平台评论库加载失败")
		return
	}

	response.Success(c, platformReviewLibraryResponse{
		Items:         items,
		Total:         total,
		SelectedCount: selectedCount,
		Limit:         filter.Limit,
		Offset:        filter.Offset,
	})
}

func (h *Handler) updatePlatformReviewFewShot(c *gin.Context) {
	var req struct {
		Selected bool `json:"selected"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	var review model.ExternalStoreReview
	if err := h.DB.First(&review, uintParam(c)).Error; err != nil {
		response.Error(c, http.StatusNotFound, "平台评论不存在")
		return
	}

	if req.Selected {
		if strings.TrimSpace(review.Content) == "" {
			response.Error(c, http.StatusBadRequest, "空评论不能作为 few_shot")
			return
		}
		now := time.Now()
		row := model.PlatformReviewFewShot{ExternalReviewID: review.ID}
		if err := h.DB.
			Where("external_review_id = ?", review.ID).
			Assign(model.PlatformReviewFewShot{
				StoreID:      review.StoreID,
				PlatformCode: review.PlatformCode,
				SelectedAt:   now,
			}).
			FirstOrCreate(&row).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "few_shot 选择保存失败")
			return
		}
		response.Success(c, gin.H{"id": review.ID, "selected": true})
		return
	}

	if err := h.DB.Where("external_review_id = ?", review.ID).Delete(&model.PlatformReviewFewShot{}).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "few_shot 选择取消失败")
		return
	}
	response.Success(c, gin.H{"id": review.ID, "selected": false})
}

func parsePlatformReviewLibraryFilter(c *gin.Context) platformReviewLibraryFilter {
	filter := platformReviewLibraryFilter{
		PlatformCode: strings.TrimSpace(c.Query("platformCode")),
		Search:       strings.TrimSpace(c.Query("q")),
		SelectedOnly: c.Query("selectedOnly") == "true" || c.Query("selectedOnly") == "1",
		Limit:        clampQueryInt(c.Query("limit"), 80, 1, 200),
		Offset:       clampQueryInt(c.Query("offset"), 0, 0, 100000),
	}
	if storeID, err := strconv.ParseUint(c.Query("storeId"), 10, 64); err == nil {
		filter.StoreID = uint(storeID)
	}
	return filter
}

func clampQueryInt(raw string, fallback int, min int, max int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (h *Handler) platformReviewLibraryQuery(filter platformReviewLibraryFilter) *gorm.DB {
	query := h.DB.
		Table("external_store_reviews").
		Joins("JOIN stores ON stores.id = external_store_reviews.store_id").
		Joins("LEFT JOIN platform_review_few_shots ON platform_review_few_shots.external_review_id = external_store_reviews.id").
		Where("external_store_reviews.content IS NOT NULL AND TRIM(external_store_reviews.content) <> ''")

	if filter.StoreID > 0 {
		query = query.Where("external_store_reviews.store_id = ?", filter.StoreID)
	}
	if filter.PlatformCode != "" {
		query = query.Where("external_store_reviews.platform_code = ?", filter.PlatformCode)
	}
	if filter.SelectedOnly {
		query = query.Where("platform_review_few_shots.id IS NOT NULL")
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		query = query.Where(
			"external_store_reviews.content LIKE ? OR external_store_reviews.user_name LIKE ? OR stores.store_name LIKE ? OR external_store_reviews.source_review_ref LIKE ?",
			like,
			like,
			like,
			like,
		)
	}
	return query
}
