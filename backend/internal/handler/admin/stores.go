package admin

import (
	"net/http"
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
	var req struct {
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
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	req.Account = strings.TrimSpace(req.Account)
	req.StoreName = strings.TrimSpace(req.StoreName)
	if req.Account == "" || req.Password == "" || req.StoreName == "" || req.TypeID == 0 {
		response.Error(c, http.StatusBadRequest, "登录账号、密码、门店名、类型均为必填")
		return
	}

	var stype model.StoreType
	if err := h.DB.First(&stype, req.TypeID).Error; err != nil {
		response.Error(c, http.StatusBadRequest, "类型不存在")
		return
	}
	industryName := presetIndustryNames[stype.IndustryCode]
	if industryName == "" {
		industryName = "餐饮"
	}
	platformStyle := strings.TrimSpace(req.PrimaryPlatformStyle)
	if platformStyle == "" {
		platformStyle = "dianping"
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "密码处理失败")
		return
	}
	merchantName := strings.TrimSpace(req.MerchantName)
	if merchantName == "" {
		merchantName = req.StoreName
	}
	contactName := strings.TrimSpace(req.ContactName)
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
		PrimaryPlatformStyle: platformStyle,
		BrandTone:            req.BrandTone,
		Status:               model.StatusEnabled,
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&merchant).Error; err != nil {
			return err
		}
		store.MerchantUserID = merchant.ID
		return tx.Create(&store).Error
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

// landingURL 拼接写入 NFC 卡片的落地地址。配置 PublicBaseURL 用绝对地址，否则相对路径。
func (h *Handler) landingURL(uuid string) string {
	path := "/landing/" + uuid
	if h.Config.PublicBaseURL != "" {
		return h.Config.PublicBaseURL + path
	}
	return path
}
