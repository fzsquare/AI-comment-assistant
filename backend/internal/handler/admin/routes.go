package admin

import (
	"net/http"
	"strconv"
	"time"

	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/response"
	"ppk/backend/internal/pkg/utils"
	"ppk/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"ppk/backend/internal/middleware"
)

type adminStatsResponse struct {
	MerchantCount              int64               `json:"merchantCount"`
	EnabledMerchantCount       int64               `json:"enabledMerchantCount"`
	DisabledMerchantCount      int64               `json:"disabledMerchantCount"`
	CurrentWeekNewMerchants    int64               `json:"currentWeekNewMerchants"`
	CurrentMonthNewMerchants   int64               `json:"currentMonthNewMerchants"`
	StoreCount                 int64               `json:"storeCount"`
	EnabledStoreCount          int64               `json:"enabledStoreCount"`
	DisabledStoreCount         int64               `json:"disabledStoreCount"`
	TagCount                   int64               `json:"tagCount"`
	TaskCount                  int64               `json:"taskCount"`
	CrawlEnabledStoreCount     int64               `json:"crawlEnabledStoreCount"`
	CrawlFailedStoreCount      int64               `json:"crawlFailedStoreCount"`
	CrawlDataAccumulatingCount int64               `json:"crawlDataAccumulatingCount"`
	TotalCustomerVisits        int64               `json:"totalCustomerVisits"`
	CurrentWeekCustomerVisits  int64               `json:"currentWeekCustomerVisits"`
	CurrentMonthCustomerVisits int64               `json:"currentMonthCustomerVisits"`
	TotalPublishClicks         int64               `json:"totalPublishClicks"`
	CurrentWeekPublishClicks   int64               `json:"currentWeekPublishClicks"`
	CurrentMonthPublishClicks  int64               `json:"currentMonthPublishClicks"`
	DeviceStats                service.DeviceStats `json:"deviceStats"`
	DataSource                 string              `json:"dataSource"`
	DataSourceLabel            string              `json:"dataSourceLabel"`
	UpdatedAt                  string              `json:"updatedAt"`
}

type adminStoreAnalytics struct {
	TotalCustomerVisits        int64               `json:"totalCustomerVisits"`
	CurrentWeekCustomerVisits  int64               `json:"currentWeekCustomerVisits"`
	CurrentMonthCustomerVisits int64               `json:"currentMonthCustomerVisits"`
	TotalPublishClicks         int64               `json:"totalPublishClicks"`
	CurrentWeekPublishClicks   int64               `json:"currentWeekPublishClicks"`
	CurrentMonthPublishClicks  int64               `json:"currentMonthPublishClicks"`
	ActivePlatformLinkCount    int64               `json:"activePlatformLinkCount"`
	DeviceStats                service.DeviceStats `json:"deviceStats"`
	DataSource                 string              `json:"dataSource"`
	DataSourceLabel            string              `json:"dataSourceLabel"`
}

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
		admin.POST("/stores/:id/review-crawl/run", h.runStoreReviewCrawl)
		admin.GET("/stores/:id/review-crawl/batches", h.listStoreReviewCrawlBatches)
		admin.GET("/stores/:id/review-crawl/matches", h.listStoreReviewCrawlMatches)
		admin.GET("/nfc-tags", h.listTags)
		admin.POST("/nfc-tags", h.createTag)
		admin.PUT("/nfc-tags/:id/bind", h.bindTag)
		admin.PUT("/nfc-tags/:id/status", h.updateTagStatus)
		admin.GET("/review-generation-tasks", h.listTasks)
		admin.GET("/stats", h.stats)
	}
}

func (h *Handler) runStoreReviewCrawl(c *gin.Context) {
	if h.ReviewCrawl == nil {
		response.Error(c, http.StatusServiceUnavailable, "评论采集服务未配置")
		return
	}
	result, err := h.ReviewCrawl.RunForStore(c.Request.Context(), uintParam(c), model.CrawlTriggerManual)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *Handler) listStoreReviewCrawlBatches(c *gin.Context) {
	var items []model.StoreReviewCrawlBatch
	if err := h.DB.Where("store_id = ?", uintParam(c)).Order("id desc").Limit(20).Find(&items).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "采集批次加载失败")
		return
	}
	response.Success(c, items)
}

func (h *Handler) listStoreReviewCrawlMatches(c *gin.Context) {
	var items []model.ExternalStoreReview
	if err := h.DB.
		Where("store_id = ? AND matched_feedback_id IS NOT NULL", uintParam(c)).
		Order("review_time desc, id desc").
		Limit(50).
		Find(&items).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "验证明细加载失败")
		return
	}
	response.Success(c, items)
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
		view := h.storeView(h.DB, item, merchant)
		analytics, err := h.storeAnalytics(item.ID)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "门店数据加载失败")
			return
		}
		view.Analytics = analytics
		views = append(views, view)
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
	h.DB.Preload("AuditLogs", func(db *gorm.DB) *gorm.DB {
		return db.Order("id desc")
	}).Order("id desc").Find(&items)
	response.Success(c, items)
}

func (h *Handler) stats(c *gin.Context) {
	loc := adminDashboardLocation()
	now := time.Now().In(loc)
	weekStart := adminStartOfWeek(now)
	monthStart := adminStartOfMonth(now)

	var merchantCount, enabledMerchantCount, disabledMerchantCount, currentWeekNewMerchants, currentMonthNewMerchants int64
	h.DB.Model(&model.MerchantUser{}).Count(&merchantCount)
	h.DB.Model(&model.MerchantUser{}).Where("status = ?", model.StatusEnabled).Count(&enabledMerchantCount)
	h.DB.Model(&model.MerchantUser{}).Where("status <> ?", model.StatusEnabled).Count(&disabledMerchantCount)
	h.DB.Model(&model.MerchantUser{}).Where("created_at >= ?", weekStart).Count(&currentWeekNewMerchants)
	h.DB.Model(&model.MerchantUser{}).Where("created_at >= ?", monthStart).Count(&currentMonthNewMerchants)

	var storeCount, enabledStoreCount, disabledStoreCount int64
	h.DB.Model(&model.Store{}).Count(&storeCount)
	h.DB.Model(&model.Store{}).Where("status = ?", model.StatusEnabled).Count(&enabledStoreCount)
	h.DB.Model(&model.Store{}).Where("status <> ?", model.StatusEnabled).Count(&disabledStoreCount)

	var tagCount, taskCount int64
	h.DB.Model(&model.NFCTag{}).Count(&tagCount)
	h.DB.Model(&model.ReviewGenerationTask{}).Count(&taskCount)

	var crawlEnabledStoreCount, crawlFailedStoreCount, crawlDataAccumulatingCount int64
	h.DB.Model(&model.StoreReviewCrawlConfig{}).Where("enabled = ?", true).Count(&crawlEnabledStoreCount)
	h.DB.Model(&model.StoreReviewCrawlConfig{}).Where("enabled = ? AND last_status = ?", true, model.CrawlStatusFailed).Count(&crawlFailedStoreCount)
	h.DB.Model(&model.StoreReviewCrawlConfig{}).Where("enabled = ? AND baseline_completed_at IS NULL", true).Count(&crawlDataAccumulatingCount)

	totalVisits, err := service.CountReviewLogAction(h.DB, 0, service.ReviewActionPageView, time.Time{}, time.Time{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "访问次数加载失败")
		return
	}
	currentWeekVisits, err := service.CountReviewLogAction(h.DB, 0, service.ReviewActionPageView, weekStart, weekStart.AddDate(0, 0, 7))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "本周访问次数加载失败")
		return
	}
	currentMonthVisits, err := service.CountReviewLogAction(h.DB, 0, service.ReviewActionPageView, monthStart, monthStart.AddDate(0, 1, 0))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "本月访问次数加载失败")
		return
	}
	totalPublishClicks, err := service.CountReviewLogAction(h.DB, 0, service.ReviewActionPlatformLinkClick, time.Time{}, time.Time{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "发布次数加载失败")
		return
	}
	currentWeekPublishClicks, err := service.CountReviewLogAction(h.DB, 0, service.ReviewActionPlatformLinkClick, weekStart, weekStart.AddDate(0, 0, 7))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "本周发布次数加载失败")
		return
	}
	currentMonthPublishClicks, err := service.CountReviewLogAction(h.DB, 0, service.ReviewActionPlatformLinkClick, monthStart, monthStart.AddDate(0, 1, 0))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "本月发布次数加载失败")
		return
	}
	deviceStats, err := service.DeviceStatsForReviewLogs(h.DB, 0, service.ReviewActionPageView, time.Time{}, time.Time{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "设备占比加载失败")
		return
	}

	response.Success(c, adminStatsResponse{
		MerchantCount:              merchantCount,
		EnabledMerchantCount:       enabledMerchantCount,
		DisabledMerchantCount:      disabledMerchantCount,
		CurrentWeekNewMerchants:    currentWeekNewMerchants,
		CurrentMonthNewMerchants:   currentMonthNewMerchants,
		StoreCount:                 storeCount,
		EnabledStoreCount:          enabledStoreCount,
		DisabledStoreCount:         disabledStoreCount,
		TagCount:                   tagCount,
		TaskCount:                  taskCount,
		CrawlEnabledStoreCount:     crawlEnabledStoreCount,
		CrawlFailedStoreCount:      crawlFailedStoreCount,
		CrawlDataAccumulatingCount: crawlDataAccumulatingCount,
		TotalCustomerVisits:        totalVisits,
		CurrentWeekCustomerVisits:  currentWeekVisits,
		CurrentMonthCustomerVisits: currentMonthVisits,
		TotalPublishClicks:         totalPublishClicks,
		CurrentWeekPublishClicks:   currentWeekPublishClicks,
		CurrentMonthPublishClicks:  currentMonthPublishClicks,
		DeviceStats:                deviceStats,
		DataSource:                 service.ReviewAnalyticsDataSource,
		DataSourceLabel:            service.ReviewAnalyticsDataSourceLabel,
		UpdatedAt:                  now.Format(time.RFC3339),
	})
}

func (h *Handler) storeAnalytics(storeID uint) (adminStoreAnalytics, error) {
	loc := adminDashboardLocation()
	now := time.Now().In(loc)
	weekStart := adminStartOfWeek(now)
	monthStart := adminStartOfMonth(now)

	totalVisits, err := service.CountReviewLogAction(h.DB, storeID, service.ReviewActionPageView, time.Time{}, time.Time{})
	if err != nil {
		return adminStoreAnalytics{}, err
	}
	currentWeekVisits, err := service.CountReviewLogAction(h.DB, storeID, service.ReviewActionPageView, weekStart, weekStart.AddDate(0, 0, 7))
	if err != nil {
		return adminStoreAnalytics{}, err
	}
	currentMonthVisits, err := service.CountReviewLogAction(h.DB, storeID, service.ReviewActionPageView, monthStart, monthStart.AddDate(0, 1, 0))
	if err != nil {
		return adminStoreAnalytics{}, err
	}
	totalPublishClicks, err := service.CountReviewLogAction(h.DB, storeID, service.ReviewActionPlatformLinkClick, time.Time{}, time.Time{})
	if err != nil {
		return adminStoreAnalytics{}, err
	}
	currentWeekPublishClicks, err := service.CountReviewLogAction(h.DB, storeID, service.ReviewActionPlatformLinkClick, weekStart, weekStart.AddDate(0, 0, 7))
	if err != nil {
		return adminStoreAnalytics{}, err
	}
	currentMonthPublishClicks, err := service.CountReviewLogAction(h.DB, storeID, service.ReviewActionPlatformLinkClick, monthStart, monthStart.AddDate(0, 1, 0))
	if err != nil {
		return adminStoreAnalytics{}, err
	}
	deviceStats, err := service.DeviceStatsForReviewLogs(h.DB, storeID, service.ReviewActionPageView, time.Time{}, time.Time{})
	if err != nil {
		return adminStoreAnalytics{}, err
	}

	var activeLinks int64
	if err := h.DB.Model(&model.StorePlatformLink{}).
		Where("store_id = ? AND status = ?", storeID, model.StatusEnabled).
		Count(&activeLinks).Error; err != nil {
		return adminStoreAnalytics{}, err
	}

	return adminStoreAnalytics{
		TotalCustomerVisits:        totalVisits,
		CurrentWeekCustomerVisits:  currentWeekVisits,
		CurrentMonthCustomerVisits: currentMonthVisits,
		TotalPublishClicks:         totalPublishClicks,
		CurrentWeekPublishClicks:   currentWeekPublishClicks,
		CurrentMonthPublishClicks:  currentMonthPublishClicks,
		ActivePlatformLinkCount:    activeLinks,
		DeviceStats:                deviceStats,
		DataSource:                 service.ReviewAnalyticsDataSource,
		DataSourceLabel:            service.ReviewAnalyticsDataSourceLabel,
	}, nil
}

func adminDashboardLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Local
	}
	return loc
}

func adminStartOfWeek(t time.Time) time.Time {
	t = t.In(adminDashboardLocation())
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	day := t.AddDate(0, 0, -(weekday - 1))
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
}

func adminStartOfMonth(t time.Time) time.Time {
	t = t.In(adminDashboardLocation())
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}
