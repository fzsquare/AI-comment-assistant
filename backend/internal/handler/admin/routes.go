package admin

import (
	"net/http"
	"strconv"

	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/response"
	"ppk/backend/internal/pkg/utils"

	"github.com/gin-gonic/gin"
	"ppk/backend/internal/middleware"
)

func (h *Handler) Register(api *gin.RouterGroup) {
	api.POST("/admin/auth/login", h.login)

	admin := api.Group("/admin")
	admin.Use(middleware.AuthRequired(h.Config.JWTSecret, middleware.DBStatusChecker(h.DB), "admin"))
	{
		admin.GET("/merchants", h.listMerchants)
		admin.PUT("/merchants/:id/status", h.updateMerchantStatus)
		admin.DELETE("/merchants/:id", h.deleteMerchant)
		admin.GET("/store-types", h.listStoreTypes)
		admin.POST("/store-types", h.createStoreType)
		admin.GET("/stores", h.listStores)
		admin.POST("/stores", h.createStore)
		admin.PUT("/stores/:id", h.updateStore)
		admin.PUT("/stores/:id/status", h.updateStoreStatus)
		admin.DELETE("/stores/:id", h.deleteStore)
		admin.GET("/nfc-tags", h.listTags)
		admin.POST("/nfc-tags", h.createTag)
		admin.PUT("/nfc-tags/:id/bind", h.bindTag)
		admin.PUT("/nfc-tags/:id/status", h.updateTagStatus)
		admin.GET("/review-generation-tasks", h.listTasks)
		admin.GET("/stats", h.stats)
	}
}

func uintParam(c *gin.Context) uint {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	return uint(id)
}

func (h *Handler) login(c *gin.Context) {
	var req struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	result, err := h.Auth.AdminLogin(req.Account, req.Password)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *Handler) listMerchants(c *gin.Context) {
	var items []model.MerchantUser
	h.DB.Order("id desc").Find(&items)
	response.Success(c, items)
}

func (h *Handler) updateMerchantStatus(c *gin.Context) {
	var item model.MerchantUser
	if err := h.DB.First(&item, uintParam(c)).Error; err != nil {
		response.Error(c, http.StatusNotFound, "商家不存在")
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

func (h *Handler) listStores(c *gin.Context) {
	var items []model.Store
	if err := h.DB.Order("id desc").Find(&items).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "门店列表加载失败")
		return
	}

	views := make([]adminStoreView, 0, len(items))
	for _, item := range items {
		var merchant model.MerchantUser
		_ = h.DB.First(&merchant, item.MerchantUserID).Error
		views = append(views, h.storeView(h.DB, item, merchant))
	}
	response.Success(c, views)
}

func (h *Handler) updateStoreStatus(c *gin.Context) {
	var item model.Store
	if err := h.DB.First(&item, uintParam(c)).Error; err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
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

func (h *Handler) listTags(c *gin.Context) {
	var items []model.NFCTag
	q := h.DB.Order("id desc")
	if sid, _ := strconv.ParseUint(c.Query("storeId"), 10, 64); sid > 0 {
		q = q.Where("store_id = ?", sid)
	}
	q.Find(&items)
	response.Success(c, items)
}

// createTag 新建一张卡片库存。可直接带 storeId 归属某门店（卡片即库存，落地走 store.uuid）。
func (h *Handler) createTag(c *gin.Context) {
	var req struct {
		TagCode string `json:"tagCode"`
		Remark  string `json:"remark"`
		StoreID uint   `json:"storeId"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.TagCode == "" {
		req.TagCode = "TAG-" + utils.RandomString(8)
	}
	item := model.NFCTag{TagCode: req.TagCode, Status: model.TagStatusUnbound, Remark: req.Remark}
	if req.StoreID > 0 {
		var store model.Store
		if err := h.DB.First(&store, req.StoreID).Error; err != nil {
			response.Error(c, http.StatusBadRequest, "门店不存在")
			return
		}
		item.StoreID = &req.StoreID
		item.Status = model.TagStatusBound
	}
	if err := h.DB.Create(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败")
		return
	}
	response.Success(c, item)
}

func (h *Handler) bindTag(c *gin.Context) {
	var item model.NFCTag
	if err := h.DB.First(&item, uintParam(c)).Error; err != nil {
		response.Error(c, http.StatusNotFound, "标签不存在")
		return
	}
	var req struct {
		StoreID uint `json:"storeId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.StoreID == 0 {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	var store model.Store
	if err := h.DB.Where("id = ? AND status = ?", req.StoreID, model.StatusEnabled).First(&store).Error; err != nil {
		response.Error(c, http.StatusBadRequest, "只能绑定启用中的门店")
		return
	}

	item.StoreID = &req.StoreID
	item.Status = model.TagStatusBound
	if err := h.DB.Save(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "绑定失败")
		return
	}
	response.Success(c, item)
}

func (h *Handler) updateTagStatus(c *gin.Context) {
	var item model.NFCTag
	if err := h.DB.First(&item, uintParam(c)).Error; err != nil {
		response.Error(c, http.StatusNotFound, "标签不存在")
		return
	}
	var req struct {
		Status string `json:"status"`
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

func (h *Handler) listTasks(c *gin.Context) {
	var items []model.ReviewGenerationTask
	h.DB.Order("id desc").Find(&items)
	response.Success(c, items)
}

func (h *Handler) stats(c *gin.Context) {
	var merchantCount, storeCount, tagCount, taskCount int64
	h.DB.Model(&model.MerchantUser{}).Count(&merchantCount)
	h.DB.Model(&model.Store{}).Count(&storeCount)
	h.DB.Model(&model.NFCTag{}).Count(&tagCount)
	h.DB.Model(&model.ReviewGenerationTask{}).Count(&taskCount)
	response.Success(c, gin.H{
		"merchantCount": merchantCount,
		"storeCount":    storeCount,
		"tagCount":      tagCount,
		"taskCount":     taskCount,
	})
}
