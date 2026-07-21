package admin

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/auth"
	"ppk/backend/internal/pkg/response"
	"ppk/backend/internal/pkg/utils"
	"ppk/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 9 行业 code -> 中文名（含别名）。写入 store.industry_type，保证 Python 串味隔离
// 与 Go 推荐标签的中文子串匹配命中；自定义类型按所选基准行业取此名。
var presetIndustryNames = map[string]string{
	"restaurant":    "餐饮",
	"footmassage":   "足疗按摩",
	"hairsalon":     "理发美发",
	"nailsalon":     "美甲美睫",
	"beauty":        "美容护肤",
	"fitness":       "健身运动",
	"entertainment": "休闲娱乐",
	"pet":           "宠物服务",
	"auto":          "汽车服务",
}

// 平台 code -> {显示名, 按钮文案}。顾客落地页据此渲染平台按钮并 deeplink 唤端。
var platformMeta = map[string][2]string{
	"dianping":    {"大众点评", "去大众点评发布"},
	"meituan":     {"美团", "去美团发布"},
	"xiaohongshu": {"小红书", "去小红书发布"},
	"douyin":      {"抖音", "去抖音发布"},
}

type storeMutationRequest struct {
	Account                   string `json:"account"`
	Password                  string `json:"password"`
	MerchantName              string `json:"merchantName"`
	ContactName               string `json:"contactName"`
	TypeID                    uint   `json:"typeId"`
	StoreName                 string `json:"storeName"`
	StoreIntro                string `json:"storeIntro"`
	Address                   string `json:"address"`
	PrimaryPlatformStyle      string `json:"primaryPlatformStyle"`
	BrandTone                 string `json:"brandTone"`
	PlatformURL               string `json:"platformUrl"`
	ReviewCrawlPlatformCode   string `json:"reviewCrawlPlatformCode"`
	ReviewCrawlExternalShopID string `json:"reviewCrawlExternalShopId"`
	ReviewCrawlEnabled        bool   `json:"reviewCrawlEnabled"`
}

type adminStoreView struct {
	model.Store
	MerchantAccount string                     `json:"merchantAccount"`
	MerchantName    string                     `json:"merchantName"`
	ContactName     string                     `json:"contactName"`
	PlatformURL     string                     `json:"platformUrl"`
	LandingURL      string                     `json:"landingUrl"`
	Analytics       adminStoreAnalytics        `json:"analytics"`
	NFCCardStatus   adminStoreNFCCardStatus    `json:"nfcCardStatus"`
	ReviewCrawl     *adminStoreReviewCrawlView `json:"reviewCrawl,omitempty"`
}

type adminStoreNFCCardStatus struct {
	TotalCount    int64  `json:"totalCount"`
	WrittenCount  int64  `json:"writtenCount"`
	DisabledCount int64  `json:"disabledCount"`
	PrimaryStatus string `json:"primaryStatus"`
	RouteStatus   string `json:"routeStatus"`
}

type adminStoreReviewCrawlView struct {
	PlatformCode        string `json:"platformCode"`
	ExternalShopID      string `json:"externalShopId"`
	Enabled             bool   `json:"enabled"`
	BaselineCompletedAt string `json:"baselineCompletedAt,omitempty"`
	LastCrawledAt       string `json:"lastCrawledAt,omitempty"`
	NextCrawlAt         string `json:"nextCrawlAt,omitempty"`
	LastStatus          string `json:"lastStatus"`
	LastErrorMessage    string `json:"lastErrorMessage,omitempty"`
}

func normalizeStoreMutationRequest(req *storeMutationRequest, requirePassword bool) error {
	req.Account = strings.TrimSpace(req.Account)
	req.StoreName = strings.TrimSpace(req.StoreName)
	req.MerchantName = strings.TrimSpace(req.MerchantName)
	req.ContactName = strings.TrimSpace(req.ContactName)
	req.PrimaryPlatformStyle = strings.TrimSpace(req.PrimaryPlatformStyle)
	req.PlatformURL = strings.TrimSpace(req.PlatformURL)
	req.Password = strings.TrimSpace(req.Password)
	req.ReviewCrawlPlatformCode = strings.TrimSpace(req.ReviewCrawlPlatformCode)
	req.ReviewCrawlExternalShopID = strings.TrimSpace(req.ReviewCrawlExternalShopID)

	if req.Account == "" || req.StoreName == "" || req.TypeID == 0 || (requirePassword && req.Password == "") {
		if requirePassword {
			return errors.New("登录账号、密码、门店名、类型均为必填")
		}
		return errors.New("登录账号、门店名、类型均为必填")
	}
	if req.PrimaryPlatformStyle == "" {
		req.PrimaryPlatformStyle = "dianping"
	}
	if err := validateClientJumpURL(req.PlatformURL); err != nil {
		return errors.New("商家跳转链接只支持 http/https")
	}
	if req.ReviewCrawlEnabled || req.ReviewCrawlExternalShopID != "" {
		if req.ReviewCrawlExternalShopID == "" {
			return errors.New("启用评论采集需要填写美团商家 ID")
		}
		if _, err := service.NormalizeReviewCrawlPlatform(req.ReviewCrawlPlatformCode); err != nil {
			return err
		}
	}
	return nil
}

func validateClientJumpURL(raw string) error {
	if raw == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("url scheme must be http or https")
	}
	if parsed.Host == "" {
		return errors.New("url host is required")
	}
	return nil
}

func industryNameForType(stype model.StoreType) string {
	industryName := presetIndustryNames[stype.IndustryCode]
	if industryName == "" {
		return "餐饮"
	}
	return industryName
}

func initialReviewGenerationTask(store model.Store, targetCount int) model.ReviewGenerationTask {
	return service.NewPendingReviewGenerationTask(
		store.ID,
		store.PrimaryPlatformStyle,
		model.TriggerInit,
		targetCount,
	)
}

func (h *Handler) listStoreTypes(c *gin.Context) {
	var items []model.StoreType
	h.DB.Order("is_preset desc, id asc").Find(&items)
	response.Success(c, items)
}

// createStoreType 新建自定义类型；必须选一个 9 行业之一作为生成/隔离基准。
func (h *Handler) createStoreType(c *gin.Context) {
	var req struct {
		Name         string `json:"name"`
		IndustryCode string `json:"industryCode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		response.Error(c, http.StatusBadRequest, "请填写类型名称")
		return
	}
	if _, ok := presetIndustryNames[req.IndustryCode]; !ok {
		response.Error(c, http.StatusBadRequest, "请选择有效的行业基准")
		return
	}
	item := model.StoreType{
		Code:         "custom-" + utils.RandomString(8),
		Name:         strings.TrimSpace(req.Name),
		IndustryCode: req.IndustryCode,
		IsPreset:     false,
		Status:       model.StatusEnabled,
	}
	if err := h.DB.Create(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败")
		return
	}
	response.Success(c, item)
}

// createStore 在某类型下新建门店：事务性创建「商家账号 + 门店」（仍一商家一店）。
// 自动生成 uuid，并把 industry_type 写成类型基准行业的中文名。
func (h *Handler) createStore(c *gin.Context) {
	var req storeMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := normalizeStoreMutationRequest(&req, true); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	var stype model.StoreType
	if err := h.DB.First(&stype, req.TypeID).Error; err != nil {
		response.Error(c, http.StatusBadRequest, "类型不存在")
		return
	}
	industryName := industryNameForType(stype)

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "密码处理失败")
		return
	}
	merchantName := req.MerchantName
	if merchantName == "" {
		merchantName = req.StoreName
	}
	contactName := req.ContactName
	if contactName == "" {
		contactName = merchantName
	}

	merchant := model.MerchantUser{
		Account:      req.Account,
		PasswordHash: hash,
		MerchantName: merchantName,
		ContactName:  contactName,
		Status:       model.StatusEnabled,
	}
	store := model.Store{
		UUID:                 utils.UUIDv4(),
		TypeID:               &req.TypeID,
		StoreName:            req.StoreName,
		IndustryType:         industryName,
		StoreIntro:           req.StoreIntro,
		Address:              req.Address,
		PrimaryPlatformStyle: req.PrimaryPlatformStyle,
		BrandTone:            req.BrandTone,
		Status:               model.StatusEnabled,
	}
	var initialTask model.ReviewGenerationTask

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&merchant).Error; err != nil {
			return err
		}
		store.MerchantUserID = merchant.ID
		if err := tx.Create(&store).Error; err != nil {
			return err
		}
		// 主推平台的商家链接：填了就落一条 store_platform_links，落地页据此唤端。
		if req.PlatformURL != "" {
			if err := savePrimaryPlatformLink(tx, store.ID, req.PrimaryPlatformStyle, req.PlatformURL); err != nil {
				return err
			}
		}
		if err := saveReviewCrawlConfig(tx, store.ID, req); err != nil {
			return err
		}
		initialTask = initialReviewGenerationTask(store, h.Config.DefaultReviewTargetCount)
		if err := tx.Create(&initialTask).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		response.Error(c, http.StatusBadRequest, "创建失败：登录账号可能已存在")
		return
	}
	if h.ReviewPool != nil {
		h.ReviewPool.RunPendingGenerationTaskAsync(initialTask.ID)
	}

	response.Success(c, gin.H{
		"store":      store,
		"merchant":   gin.H{"id": merchant.ID, "account": merchant.Account},
		"landingUrl": h.landingURL(store.UUID),
	})
}

func (h *Handler) updateStore(c *gin.Context) {
	var store model.Store
	if err := h.DB.First(&store, uintParam(c)).Error; err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}

	var req storeMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := normalizeStoreMutationRequest(&req, false); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	var stype model.StoreType
	if err := h.DB.First(&stype, req.TypeID).Error; err != nil {
		response.Error(c, http.StatusBadRequest, "类型不存在")
		return
	}
	industryName := industryNameForType(stype)

	var updated adminStoreView
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		var merchant model.MerchantUser
		if err := tx.First(&merchant, store.MerchantUserID).Error; err != nil {
			return err
		}

		merchant.Account = req.Account
		merchant.MerchantName = req.MerchantName
		if merchant.MerchantName == "" {
			merchant.MerchantName = req.StoreName
		}
		merchant.ContactName = req.ContactName
		if merchant.ContactName == "" {
			merchant.ContactName = merchant.MerchantName
		}
		if req.Password != "" {
			hash, err := auth.HashPassword(req.Password)
			if err != nil {
				return err
			}
			merchant.PasswordHash = hash
		}
		if err := tx.Save(&merchant).Error; err != nil {
			return err
		}

		store.TypeID = &req.TypeID
		store.StoreName = req.StoreName
		store.IndustryType = industryName
		store.StoreIntro = req.StoreIntro
		store.Address = req.Address
		store.PrimaryPlatformStyle = req.PrimaryPlatformStyle
		store.BrandTone = req.BrandTone
		if err := tx.Save(&store).Error; err != nil {
			return err
		}

		if err := savePrimaryPlatformLink(tx, store.ID, req.PrimaryPlatformStyle, req.PlatformURL); err != nil {
			return err
		}
		if err := saveReviewCrawlConfig(tx, store.ID, req); err != nil {
			return err
		}

		updated = h.storeView(tx, store, merchant)
		return nil
	}); err != nil {
		response.Error(c, http.StatusBadRequest, "保存失败：登录账号可能已存在")
		return
	}

	response.Success(c, updated)
}

func savePrimaryPlatformLink(db *gorm.DB, storeID uint, platformCode string, targetURL string) error {
	if targetURL == "" {
		return db.Where("store_id = ? AND platform_code = ?", storeID, platformCode).
			Delete(&model.StorePlatformLink{}).Error
	}

	meta := platformMeta[platformCode]
	name := meta[0]
	if name == "" {
		name = platformCode
	}
	btn := meta[1]
	if btn == "" {
		btn = "去发布"
	}

	var link model.StorePlatformLink
	err := db.Where("store_id = ? AND platform_code = ?", storeID, platformCode).First(&link).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		link = model.StorePlatformLink{
			StoreID:      storeID,
			PlatformCode: platformCode,
			SortNo:       1,
		}
	}

	link.PlatformName = name
	link.ButtonText = btn
	link.TargetURL = targetURL
	link.BackupURL = targetURL
	link.SortNo = 1
	link.Status = model.StatusEnabled

	if link.ID == 0 {
		return db.Create(&link).Error
	}
	return db.Save(&link).Error
}

func saveReviewCrawlConfig(db *gorm.DB, storeID uint, req storeMutationRequest) error {
	platformCode := strings.TrimSpace(req.ReviewCrawlPlatformCode)
	externalShopID := strings.TrimSpace(req.ReviewCrawlExternalShopID)

	var item model.StoreReviewCrawlConfig
	err := db.Where("store_id = ?", storeID).First(&item).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if !req.ReviewCrawlEnabled && externalShopID == "" {
			return nil
		}
		normalizedPlatform, err := service.NormalizeReviewCrawlPlatform(platformCode)
		if err != nil {
			return err
		}
		item = model.StoreReviewCrawlConfig{
			StoreID:      storeID,
			PlatformCode: normalizedPlatform,
			LastStatus:   model.CrawlStatusNeverRun,
		}
	} else if !req.ReviewCrawlEnabled && externalShopID == "" {
		if platformCode != "" {
			normalizedPlatform, err := service.NormalizeReviewCrawlPlatform(platformCode)
			if err != nil {
				return err
			}
			item.PlatformCode = normalizedPlatform
		}
		item.ExternalShopID = ""
		item.Enabled = false
		return db.Save(&item).Error
	} else {
		normalizedPlatform, err := service.NormalizeReviewCrawlPlatform(platformCode)
		if err != nil {
			return err
		}
		item.PlatformCode = normalizedPlatform
	}
	item.ExternalShopID = externalShopID
	item.Enabled = req.ReviewCrawlEnabled && externalShopID != ""
	if item.ID == 0 {
		return db.Create(&item).Error
	}
	return db.Save(&item).Error
}

func (h *Handler) storeView(db *gorm.DB, store model.Store, merchant model.MerchantUser) adminStoreView {
	view := adminStoreView{
		Store:           store,
		MerchantAccount: merchant.Account,
		MerchantName:    merchant.MerchantName,
		ContactName:     merchant.ContactName,
		LandingURL:      h.landingURL(store.UUID),
	}

	var link model.StorePlatformLink
	if err := db.Where("store_id = ? AND platform_code = ?", store.ID, store.PrimaryPlatformStyle).
		First(&link).Error; err == nil {
		view.PlatformURL = link.TargetURL
	}
	var crawlConfig model.StoreReviewCrawlConfig
	if err := db.Where("store_id = ?", store.ID).First(&crawlConfig).Error; err == nil {
		view.ReviewCrawl = reviewCrawlView(crawlConfig)
	}
	view.NFCCardStatus = h.nfcCardStatus(db, store)
	return view
}

func (h *Handler) nfcCardStatus(db *gorm.DB, store model.Store) adminStoreNFCCardStatus {
	var totalCount, writtenCount, disabledCount int64
	db.Model(&model.NFCTag{}).Where("store_id = ?", store.ID).Count(&totalCount)
	db.Model(&model.NFCTag{}).Where("store_id = ? AND status = ?", store.ID, model.TagStatusBound).Count(&writtenCount)
	db.Model(&model.NFCTag{}).Where("store_id = ? AND status = ?", store.ID, model.TagStatusDisabled).Count(&disabledCount)
	return deriveNFCCardStatus(store, totalCount, writtenCount, disabledCount)
}

func deriveNFCCardStatus(store model.Store, totalCount, writtenCount, disabledCount int64) adminStoreNFCCardStatus {
	status := adminStoreNFCCardStatus{
		TotalCount:    totalCount,
		WrittenCount:  writtenCount,
		DisabledCount: disabledCount,
		PrimaryStatus: "unusable",
		RouteStatus:   "ok",
	}
	if strings.TrimSpace(store.UUID) == "" {
		status.PrimaryStatus = "unwritten"
		status.RouteStatus = "missing_uuid"
		return status
	}
	if store.Status != model.StatusEnabled {
		status.PrimaryStatus = "unusable"
		status.RouteStatus = "store_inactive"
		return status
	}
	if writtenCount > 0 {
		status.PrimaryStatus = "usable"
		status.RouteStatus = "ok"
		return status
	}
	if totalCount == 0 {
		status.PrimaryStatus = "unwritten"
		status.RouteStatus = "no_bound_tag"
		return status
	}
	status.RouteStatus = "no_active_bound_tag"
	return status
}

func reviewCrawlView(config model.StoreReviewCrawlConfig) *adminStoreReviewCrawlView {
	view := &adminStoreReviewCrawlView{
		PlatformCode:     config.PlatformCode,
		ExternalShopID:   config.ExternalShopID,
		Enabled:          config.Enabled,
		LastStatus:       config.LastStatus,
		LastErrorMessage: config.LastErrorMessage,
	}
	if config.BaselineCompletedAt != nil {
		view.BaselineCompletedAt = config.BaselineCompletedAt.Format(time.RFC3339)
	}
	if config.LastCrawledAt != nil {
		view.LastCrawledAt = config.LastCrawledAt.Format(time.RFC3339)
	}
	if config.NextCrawlAt != nil {
		view.NextCrawlAt = config.NextCrawlAt.Format(time.RFC3339)
	}
	return view
}

// landingURL 拼接写入 NFC 卡片的落地地址。
// 配置 PublicBaseURL 时返回绝对地址；否则按 PUBLIC_BASE_PATH 返回子路径相对地址。
func (h *Handler) landingURL(uuid string) string {
	path := "/landing/" + uuid
	if h.Config.PublicBaseURL != "" {
		return h.Config.PublicBaseURL + path
	}
	if h.Config.PublicBasePath != "" {
		return h.Config.PublicBasePath + path
	}
	return path
}
