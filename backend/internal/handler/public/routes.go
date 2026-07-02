package public

import (
	"net/http"
	"strings"
	"time"

	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/response"
	"ppk/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	DB         *gorm.DB
	ReviewPool *service.ReviewPoolService
}

func NewHandler(db *gorm.DB, reviewPool *service.ReviewPoolService) *Handler {
	return &Handler{DB: db, ReviewPool: reviewPool}
}

func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/public/landing/:uuid/init", h.initLanding)
	api.POST("/public/landing/:uuid/switch-review", h.switchReview)
	api.POST("/public/landing/:uuid/events", h.createEvent)
}

func (h *Handler) initLanding(c *gin.Context) {
	payload, err := h.ReviewPool.InitLandingByUUID(c.Param("uuid"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, payload)
}

func (h *Handler) switchReview(c *gin.Context) {
	var req struct {
		PlatformCode string `json:"platformCode"`
		Tag          string `json:"tag"` // 顾客选的菜品/场景标签，空则随机换一条
	}
	_ = c.ShouldBindJSON(&req)
	item, remaining, err := h.ReviewPool.SwitchReview(c.Param("uuid"), req.PlatformCode, req.Tag)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, gin.H{"review": item, "remainingDispatchableCount": remaining})
}

type eventRequest struct {
	SessionID       string `json:"sessionId"`
	ReviewItemID    uint   `json:"reviewItemId"`
	ActionType      string `json:"actionType"`
	PlatformCode    string `json:"platformCode"`
	EditedContent   string `json:"editedContent"`
	ClientUserAgent string `json:"clientUserAgent"`
}

func (h *Handler) createEvent(c *gin.Context) {
	var req eventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	var store model.Store
	if err := h.DB.Where("uuid = ?", c.Param("uuid")).First(&store).Error; err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	// uuid 落地：URL 为门店级，不区分具体卡片，nfc_tag_id 记 0（无特定卡）。
	userAgent := eventUserAgent(c, req)
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		log := model.ReviewDisplayLog{StoreID: store.ID, ReviewItemID: req.ReviewItemID, SessionID: req.SessionID, ActionType: req.ActionType, PlatformCode: req.PlatformCode, ClientIP: c.ClientIP(), UserAgent: userAgent}
		if err := tx.Create(&log).Error; err != nil {
			return err
		}
		return h.recordFeedback(tx, store, req, c.ClientIP(), userAgent)
	}); err != nil {
		response.Error(c, http.StatusInternalServerError, "保存失败")
		return
	}
	response.Success(c, gin.H{"saved": true})
}

func eventUserAgent(c *gin.Context, req eventRequest) string {
	ua := strings.TrimSpace(c.Request.UserAgent())
	if ua == "" {
		ua = strings.TrimSpace(req.ClientUserAgent)
	}
	if len(ua) > 255 {
		return ua[:255]
	}
	return ua
}

func (h *Handler) recordFeedback(tx *gorm.DB, store model.Store, req eventRequest, clientIP string, userAgent string) error {
	feedbackType, tracked := feedbackTypeForAction(req.ActionType)
	if !tracked {
		return nil
	}
	if req.ReviewItemID == 0 {
		return nil
	}

	var review model.ReviewItem
	if err := tx.Where("id = ? AND store_id = ?", req.ReviewItemID, store.ID).First(&review).Error; err != nil {
		return err
	}

	platformCode := strings.TrimSpace(req.PlatformCode)
	if platformCode == "" {
		platformCode = review.PlatformStyle
	}
	if platformCode == "" {
		platformCode = store.PrimaryPlatformStyle
	}

	var acceptedCount int64
	if err := tx.Model(&model.ReviewFeedback{}).
		Where("store_id = ? AND review_item_id = ? AND session_id = ? AND feedback_type = ?", store.ID, req.ReviewItemID, req.SessionID, model.ReviewFeedbackAccepted).
		Count(&acceptedCount).Error; err != nil {
		return err
	}
	if feedbackType == model.ReviewFeedbackRejected && acceptedCount > 0 {
		return nil
	}

	var existingCount int64
	if err := tx.Model(&model.ReviewFeedback{}).
		Where("store_id = ? AND review_item_id = ? AND session_id = ? AND feedback_type = ?", store.ID, req.ReviewItemID, req.SessionID, feedbackType).
		Count(&existingCount).Error; err != nil {
		return err
	}
	if existingCount == 0 {
		feedback := model.ReviewFeedback{
			StoreID:         store.ID,
			ReviewItemID:    req.ReviewItemID,
			SessionID:       req.SessionID,
			PlatformCode:    platformCode,
			FeedbackType:    feedbackType,
			SourceAction:    req.ActionType,
			ContentSnapshot: review.Content,
			EditedContent:   strings.TrimSpace(req.EditedContent),
			ClientIP:        clientIP,
			UserAgent:       userAgent,
		}
		if err := tx.Create(&feedback).Error; err != nil {
			return err
		}
	}

	if status, ok := reviewStatusForFeedback(feedbackType); ok && review.Status != status {
		updates := map[string]any{"status": status}
		if feedbackType == model.ReviewFeedbackAccepted && review.UsedAt == nil {
			updates["used_at"] = time.Now()
		}
		return tx.Model(&model.ReviewItem{}).
			Where("id = ? AND store_id = ?", review.ID, store.ID).
			Updates(updates).Error
	}
	return nil
}

func feedbackTypeForAction(action string) (string, bool) {
	switch strings.TrimSpace(action) {
	case "review_copy", "platform_link_click":
		return model.ReviewFeedbackAccepted, true
	case "review_reject":
		return model.ReviewFeedbackRejected, true
	default:
		return "", false
	}
}

func reviewStatusForFeedback(feedbackType string) (string, bool) {
	switch feedbackType {
	case model.ReviewFeedbackAccepted:
		return model.ReviewStatusUsed, true
	case model.ReviewFeedbackRejected:
		return model.ReviewStatusDisabled, true
	default:
		return "", false
	}
}
