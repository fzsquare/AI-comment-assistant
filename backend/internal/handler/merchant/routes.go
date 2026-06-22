package merchant

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"ppk/backend/internal/middleware"
	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Register(api *gin.RouterGroup) {
	api.POST("/merchant/auth/login", h.login)

	merchant := api.Group("/merchant")
	merchant.Use(middleware.AuthRequired(h.Config.JWTSecret, middleware.DBStatusChecker(h.DB), "merchant"))
	{
		merchant.GET("/store/detail", h.getStoreDetail)
		merchant.PUT("/store/detail", h.updateStoreDetail)

		merchant.GET("/store/keywords", h.listKeywords)
		merchant.POST("/store/keywords", h.createKeyword)
		merchant.PUT("/store/keywords/:id", h.updateKeyword)
		merchant.DELETE("/store/keywords/:id", h.deleteKeyword)

		merchant.GET("/store/images", h.listImages)
		merchant.POST("/store/images/upload", h.createImage)
		merchant.DELETE("/store/images/:id", h.deleteImage)

		merchant.GET("/store/platform-links", h.listPlatformLinks)
		merchant.POST("/store/platform-links", h.createPlatformLink)
		merchant.PUT("/store/platform-links/:id", h.updatePlatformLink)
		merchant.DELETE("/store/platform-links/:id", h.deletePlatformLink)
		merchant.PUT("/store/platform-links/:id/status", h.updatePlatformLinkStatus)

		merchant.GET("/reviews", h.listReviews)
		merchant.POST("/reviews", h.createReview)
		merchant.PUT("/reviews/:id", h.updateReview)
		merchant.DELETE("/reviews/:id", h.deleteReview)
		merchant.POST("/reviews/generate", h.generateReviews)
		merchant.GET("/review-generation-tasks", h.listGenerationTasks)
	}
}

func (h *Handler) merchantID(c *gin.Context) uint {
	value, _ := c.Get("userID")
	id, _ := value.(uint)
	return id
}

func (h *Handler) currentStore(c *gin.Context) (*model.Store, error) {
	var store model.Store
	err := h.DB.Where("merchant_user_id = ?", h.merchantID(c)).First(&store).Error
	return &store, err
}

func parseUintParam(c *gin.Context, key string) uint {
	id, _ := strconv.ParseUint(c.Param(key), 10, 64)
	return uint(id)
}

func validateHTTPURL(raw string, required bool) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if required {
			return errors.New("url is required")
		}
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("url scheme must be http or https")
	}
	if u.Host == "" {
		return errors.New("url host is required")
	}
	return nil
}

func normalizeTargetCount(value, defaultValue, maxValue int) (int, error) {
	if value == 0 {
		value = defaultValue
	}
	if value < 0 {
		return 0, errors.New("targetCount must be greater than 0")
	}
	if value > maxValue {
		return 0, errors.New("targetCount exceeds maximum")
	}
	return value, nil
}

type loginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	result, err := h.Auth.MerchantLogin(req.Account, req.Password)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *Handler) getStoreDetail(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	response.Success(c, store)
}

type updateStoreRequest struct {
	StoreName            string `json:"storeName"`
	IndustryType         string `json:"industryType"`
	StoreIntro           string `json:"storeIntro"`
	Address              string `json:"address"`
	PrimaryPlatformStyle string `json:"primaryPlatformStyle"`
	BrandTone            string `json:"brandTone"`
}

func (h *Handler) updateStoreDetail(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var req updateStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	store.StoreName = req.StoreName
	store.IndustryType = req.IndustryType
	store.StoreIntro = req.StoreIntro
	store.Address = req.Address
	store.PrimaryPlatformStyle = req.PrimaryPlatformStyle
	store.BrandTone = req.BrandTone
	if err := h.DB.Save(store).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "保存失败")
		return
	}
	response.Success(c, store)
}

type keywordRequest struct {
	Keyword string `json:"keyword"`
	SortNo  int    `json:"sortNo"`
}

func (h *Handler) listKeywords(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var items []model.StoreKeyword
	h.DB.Where("store_id = ?", store.ID).Order("sort_no asc, id asc").Find(&items)
	response.Success(c, items)
}

func (h *Handler) createKeyword(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var req keywordRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Keyword == "" {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	item := model.StoreKeyword{StoreID: store.ID, Keyword: req.Keyword, SortNo: req.SortNo}
	if err := h.DB.Create(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败")
		return
	}
	response.Success(c, item)
}

func (h *Handler) updateKeyword(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var item model.StoreKeyword
	if err := h.DB.Where("id = ? AND store_id = ?", parseUintParam(c, "id"), store.ID).First(&item).Error; err != nil {
		response.Error(c, http.StatusNotFound, "关键词不存在")
		return
	}
	var req keywordRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Keyword == "" {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	item.Keyword = req.Keyword
	item.SortNo = req.SortNo
	if err := h.DB.Save(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "更新失败")
		return
	}
	response.Success(c, item)
}

func (h *Handler) deleteKeyword(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	if err := h.DB.Where("id = ? AND store_id = ?", parseUintParam(c, "id"), store.ID).Delete(&model.StoreKeyword{}).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败")
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

type imageRequest struct {
	ImageURL     string `json:"imageUrl"`
	ThumbnailURL string `json:"thumbnailUrl"`
	SortNo       int    `json:"sortNo"`
}

func (h *Handler) listImages(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var items []model.StoreImage
	h.DB.Where("store_id = ?", store.ID).Order("sort_no asc, id asc").Find(&items)
	response.Success(c, items)
}

func (h *Handler) createImage(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var req imageRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ImageURL == "" {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := validateHTTPURL(req.ImageURL, true); err != nil || validateHTTPURL(req.ThumbnailURL, false) != nil {
		response.Error(c, http.StatusBadRequest, "URL 协议不支持")
		return
	}
	item := model.StoreImage{StoreID: store.ID, ImageURL: req.ImageURL, ThumbnailURL: req.ThumbnailURL, SortNo: req.SortNo, Status: model.StatusEnabled}
	if err := h.DB.Create(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败")
		return
	}
	response.Success(c, item)
}

func (h *Handler) deleteImage(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	if err := h.DB.Where("id = ? AND store_id = ?", parseUintParam(c, "id"), store.ID).Delete(&model.StoreImage{}).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败")
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

type platformLinkRequest struct {
	PlatformCode string `json:"platformCode"`
	PlatformName string `json:"platformName"`
	ButtonText   string `json:"buttonText"`
	TargetURL    string `json:"targetUrl"`
	BackupURL    string `json:"backupUrl"`
	SortNo       int    `json:"sortNo"`
	Status       int    `json:"status"`
}

func (h *Handler) listPlatformLinks(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var items []model.StorePlatformLink
	h.DB.Where("store_id = ?", store.ID).Order("sort_no asc, id asc").Find(&items)
	response.Success(c, items)
}

func (h *Handler) createPlatformLink(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var req platformLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.TargetURL == "" || req.PlatformCode == "" {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := validateHTTPURL(req.TargetURL, true); err != nil || validateHTTPURL(req.BackupURL, false) != nil {
		response.Error(c, http.StatusBadRequest, "URL 协议不支持")
		return
	}
	item := model.StorePlatformLink{StoreID: store.ID, PlatformCode: req.PlatformCode, PlatformName: req.PlatformName, ButtonText: req.ButtonText, TargetURL: req.TargetURL, BackupURL: req.BackupURL, SortNo: req.SortNo, Status: req.Status}
	if item.Status == 0 {
		item.Status = model.StatusEnabled
	}
	if err := h.DB.Create(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败")
		return
	}
	response.Success(c, item)
}

func (h *Handler) updatePlatformLink(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var item model.StorePlatformLink
	if err := h.DB.Where("id = ? AND store_id = ?", parseUintParam(c, "id"), store.ID).First(&item).Error; err != nil {
		response.Error(c, http.StatusNotFound, "平台链接不存在")
		return
	}
	var req platformLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.TargetURL == "" || req.PlatformCode == "" {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := validateHTTPURL(req.TargetURL, true); err != nil || validateHTTPURL(req.BackupURL, false) != nil {
		response.Error(c, http.StatusBadRequest, "URL 协议不支持")
		return
	}
	item.PlatformCode = req.PlatformCode
	item.PlatformName = req.PlatformName
	item.ButtonText = req.ButtonText
	item.TargetURL = req.TargetURL
	item.BackupURL = req.BackupURL
	item.SortNo = req.SortNo
	if req.Status != 0 {
		item.Status = req.Status
	}
	if err := h.DB.Save(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "更新失败")
		return
	}
	response.Success(c, item)
}

func (h *Handler) deletePlatformLink(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	if err := h.DB.Where("id = ? AND store_id = ?", parseUintParam(c, "id"), store.ID).Delete(&model.StorePlatformLink{}).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败")
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *Handler) updatePlatformLinkStatus(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var item model.StorePlatformLink
	if err := h.DB.Where("id = ? AND store_id = ?", parseUintParam(c, "id"), store.ID).First(&item).Error; err != nil {
		response.Error(c, http.StatusNotFound, "平台链接不存在")
		return
	}
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	item.Status = req.Status
	if err := h.DB.Save(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "更新失败")
		return
	}
	response.Success(c, item)
}

type reviewRequest struct {
	Content string `json:"content"`
	Status  string `json:"status"`
}

func (h *Handler) listReviews(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var items []model.ReviewItem
	h.DB.Where("store_id = ? AND status <> ?", store.ID, model.ReviewStatusDeleted).Order("id desc").Find(&items)
	response.Success(c, items)
}

func (h *Handler) createReview(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var req reviewRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	status := req.Status
	if status == "" {
		status = model.ReviewStatusAvailable
	}
	item := model.ReviewItem{StoreID: store.ID, PlatformStyle: store.PrimaryPlatformStyle, Content: req.Content, SourceType: "manual", GenerationBatchNo: "manual", Status: status}
	if err := h.DB.Create(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败")
		return
	}
	response.Success(c, item)
}

func (h *Handler) updateReview(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var item model.ReviewItem
	if err := h.DB.Where("id = ? AND store_id = ?", parseUintParam(c, "id"), store.ID).First(&item).Error; err != nil {
		response.Error(c, http.StatusNotFound, "评价不存在")
		return
	}
	var req reviewRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	item.Content = req.Content
	if req.Status != "" {
		item.Status = req.Status
	}
	if err := h.DB.Save(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "更新失败")
		return
	}
	response.Success(c, item)
}

func (h *Handler) deleteReview(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var item model.ReviewItem
	if err := h.DB.Where("id = ? AND store_id = ?", parseUintParam(c, "id"), store.ID).First(&item).Error; err != nil {
		response.Error(c, http.StatusNotFound, "评价不存在")
		return
	}
	item.Status = model.ReviewStatusDeleted
	if err := h.DB.Save(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败")
		return
	}
	_ = h.ReviewPool.EnsureDispatchableStock(store.ID)
	response.Success(c, item)
}

func (h *Handler) generateReviews(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var req struct {
		TargetCount int `json:"targetCount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	targetCount, err := normalizeTargetCount(req.TargetCount, h.Config.DefaultReviewTargetCount, h.Config.MaxReviewGenerateCount)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "targetCount 超出允许范围")
		return
	}
	if err := h.ReviewPool.GenerateForStore(store.ID, model.TriggerManual, targetCount); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"generated": targetCount})
}

func (h *Handler) listGenerationTasks(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var items []model.ReviewGenerationTask
	h.DB.Where("store_id = ?", store.ID).Order("id desc").Find(&items)
	response.Success(c, items)
}
