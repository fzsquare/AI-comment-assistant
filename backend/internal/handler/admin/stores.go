package admin

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/auth"
	"ppk/backend/internal/pkg/response"
	"ppk/backend/internal/pkg/utils"

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
	Account              string `json:"account"`
	Password             string `json:"password"`
	MerchantName         string `json:"merchantName"`
	ContactName          string `json:"contactName"`
	TypeID               uint   `json:"typeId"`
	StoreName            string `json:"storeName"`
	StoreIntro           string `json:"storeIntro"`
	Address              string `json:"address"`
	PrimaryPlatformStyle string `json:"primaryPlatformStyle"`
	BrandTone            string `json:"brandTone"`
	PlatformURL          string `json:"platformUrl"`
}

type adminStoreView struct {
	model.Store
	MerchantAccount string `json:"merchantAccount"`
	MerchantName    string `json:"merchantName"`
	ContactName     string `json:"contactName"`
	PlatformURL     string `json:"platformUrl"`
	LandingURL      string `json:"landingUrl"`
}

func normalizeStoreMutationRequest(req *storeMutationRequest, requirePassword bool) error {
	req.Account = strings.TrimSpace(req.Account)
	req.StoreName = strings.TrimSpace(req.StoreName)
	req.MerchantName = strings.TrimSpace(req.MerchantName)
	req.ContactName = strings.TrimSpace(req.ContactName)
	req.PrimaryPlatformStyle = strings.TrimSpace(req.PrimaryPlatformStyle)
	req.PlatformURL = strings.TrimSpace(req.PlatformURL)
	req.Password = strings.TrimSpace(req.Password)

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
		return nil
	}); err != nil {
		response.Error(c, http.StatusBadRequest, "创建失败：登录账号可能已存在")
		return
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
