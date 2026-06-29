package public

import (
	"net/http"

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

func (h *Handler) createEvent(c *gin.Context) {
	var req struct {
		SessionID    string `json:"sessionId"`
		ReviewItemID uint   `json:"reviewItemId"`
		ActionType   string `json:"actionType"`
		PlatformCode string `json:"platformCode"`
	}
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
	log := model.ReviewDisplayLog{StoreID: store.ID, ReviewItemID: req.ReviewItemID, SessionID: req.SessionID, ActionType: req.ActionType, PlatformCode: req.PlatformCode, ClientIP: c.ClientIP(), UserAgent: c.Request.UserAgent()}
	if err := h.DB.Create(&log).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "保存失败")
		return
	}
	response.Success(c, gin.H{"saved": true})
}
