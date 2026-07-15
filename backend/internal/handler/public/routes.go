package public

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/response"
	"ppk/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
		SessionID    string `json:"sessionId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	req.PlatformCode = strings.TrimSpace(req.PlatformCode)
	req.SessionID = strings.TrimSpace(req.SessionID)
	var store model.Store
	if err := h.DB.Where("uuid = ?", c.Param("uuid")).First(&store).Error; err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	if err := h.ReviewPool.VerifyLandingSession(store.UUID, req.SessionID, time.Now()); err != nil {
		response.Error(c, http.StatusBadRequest, "会话已失效，请刷新页面后重试")
		return
	}
	item, remaining, err := h.ReviewPool.SwitchReview(c.Param("uuid"), req.PlatformCode, strings.TrimSpace(req.Tag), req.SessionID)
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
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActionType = strings.TrimSpace(req.ActionType)
	req.PlatformCode = strings.TrimSpace(req.PlatformCode)
	var store model.Store
	if err := h.DB.Where("uuid = ?", c.Param("uuid")).First(&store).Error; err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	if err := h.ReviewPool.VerifyLandingSession(store.UUID, req.SessionID, time.Now()); err != nil {
		response.Error(c, http.StatusBadRequest, "会话已失效，请刷新页面后重试")
		return
	}
	if err := validateLandingEventShape(req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.validateLandingEventReferences(store, req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
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

func validateLandingEventShape(req eventRequest) error {
	action := strings.TrimSpace(req.ActionType)
	needsReview := false
	needsPlatform := false
	switch action {
	case "page_view":
	case "platform_select":
		needsPlatform = true
	case "review_switch", "review_pick_by_tag", "review_reject", "review_copy", "platform_link_click":
		needsReview = true
		needsPlatform = true
	default:
		return errors.New("不支持的行为类型")
	}
	if strings.TrimSpace(req.SessionID) == "" {
		return errors.New("会话不能为空")
	}
	if needsReview && req.ReviewItemID == 0 {
		return errors.New("评价内容不能为空")
	}
	if needsPlatform && strings.TrimSpace(req.PlatformCode) == "" {
		return errors.New("评价平台不能为空")
	}
	return nil
}

func (h *Handler) validateLandingEventReferences(store model.Store, req eventRequest) error {
	platformCode := strings.TrimSpace(req.PlatformCode)
	if platformCode != "" {
		var platformCount int64
		if err := h.DB.Model(&model.StorePlatformLink{}).
			Where("store_id = ? AND platform_code = ? AND status = ?", store.ID, platformCode, model.StatusEnabled).
			Count(&platformCount).Error; err != nil {
			return err
		}
		if platformCount == 0 {
			return errors.New("评价平台不可用")
		}
	}
	if req.ReviewItemID != 0 {
		var reviewCount int64
		if err := dispatchedReviewReferenceQuery(h.DB, store, req, platformCode).Count(&reviewCount).Error; err != nil {
			return err
		}
		if reviewCount == 0 {
			return errors.New("评价内容无效")
		}
	}
	return nil
}

func dispatchedReviewReferenceQuery(db *gorm.DB, store model.Store, req eventRequest, platformCode string) *gorm.DB {
	return db.Model(&model.ReviewItem{}).
		Where("id = ? AND store_id = ? AND platform_style = ? AND dispatched_session_id = ?", req.ReviewItemID, store.ID, platformCode, req.SessionID)
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
	// review_copy 与 platform_link_click 会并发上报且都代表 accepted。
	// 唯一键冲突必须幂等忽略，不能回滚已经写入的漏斗事件日志。
	if err := insertFeedbackIfAbsent(tx, &feedback).Error; err != nil {
		return err
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

func insertFeedbackIfAbsent(tx *gorm.DB, feedback *model.ReviewFeedback) *gorm.DB {
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(feedback)
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
