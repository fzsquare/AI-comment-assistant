package merchant

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type lotteryPrizeRequest struct {
	Name     string `json:"name"`
	ImageURL string `json:"imageUrl"`
	Stock    int    `json:"stock"`
	WinRate  int    `json:"winRate"`
	Enabled  bool   `json:"enabled"`
}

type lotteryConfigRequest struct {
	Enabled bool                  `json:"enabled"`
	Prizes  []lotteryPrizeRequest `json:"prizes"`
}

type lotteryConfigResponse struct {
	Enabled bool                      `json:"enabled"`
	Prizes  []model.StoreLotteryPrize `json:"prizes"`
}

func (h *Handler) getLotteryConfig(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var config model.StoreLotteryConfig
	if err := h.DB.Where("store_id = ?", store.ID).First(&config).Error; err != nil && err != gorm.ErrRecordNotFound {
		response.Error(c, http.StatusInternalServerError, "抽奖配置加载失败")
		return
	}
	var prizes []model.StoreLotteryPrize
	if err := h.DB.Where("store_id = ?", store.ID).Order("sort_no asc, id asc").Find(&prizes).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "奖品加载失败")
		return
	}
	response.Success(c, lotteryConfigResponse{Enabled: config.Enabled, Prizes: prizes})
}

func validateLotteryConfig(req lotteryConfigRequest) error {
	if len(req.Prizes) > 12 {
		return gin.Error{Err: errLottery("奖品最多 12 个"), Type: gin.ErrorTypeBind}
	}
	totalRate := 0
	for _, prize := range req.Prizes {
		name := strings.TrimSpace(prize.Name)
		if name == "" || utf8.RuneCountInString(name) > 64 {
			return gin.Error{Err: errLottery("请填写 64 字以内的奖品名称"), Type: gin.ErrorTypeBind}
		}
		if len(strings.TrimSpace(prize.ImageURL)) > 500 {
			return gin.Error{Err: errLottery("奖品图片地址过长"), Type: gin.ErrorTypeBind}
		}
		if prize.Stock < 0 || prize.Stock > 100000 {
			return gin.Error{Err: errLottery("奖品库存需在 0 到 100000 之间"), Type: gin.ErrorTypeBind}
		}
		if prize.WinRate < 0 || prize.WinRate > 100 {
			return gin.Error{Err: errLottery("中奖概率需在 0 到 100 之间"), Type: gin.ErrorTypeBind}
		}
		if prize.Enabled {
			totalRate += prize.WinRate
		}
	}
	if totalRate > 100 {
		return gin.Error{Err: errLottery("已启用奖品的中奖概率合计不能超过 100%"), Type: gin.ErrorTypeBind}
	}
	if req.Enabled && totalRate == 0 {
		return gin.Error{Err: errLottery("开启抽奖前，请至少配置一个有中奖概率的奖品"), Type: gin.ErrorTypeBind}
	}
	return nil
}

type lotteryError string

func (e lotteryError) Error() string  { return string(e) }
func errLottery(message string) error { return lotteryError(message) }

func (h *Handler) saveLotteryConfig(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var req lotteryConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := validateLotteryConfig(req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	prizes := make([]model.StoreLotteryPrize, 0, len(req.Prizes))
	for index, prize := range req.Prizes {
		prizes = append(prizes, model.StoreLotteryPrize{StoreID: store.ID, Name: strings.TrimSpace(prize.Name), ImageURL: strings.TrimSpace(prize.ImageURL), Stock: prize.Stock, WinRate: prize.WinRate, SortNo: index + 1, Enabled: prize.Enabled})
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		config := model.StoreLotteryConfig{StoreID: store.ID, Enabled: req.Enabled}
		if err := tx.Where("store_id = ?", store.ID).Assign(config).FirstOrCreate(&config).Error; err != nil {
			return err
		}
		if err := tx.Where("store_id = ?", store.ID).Delete(&model.StoreLotteryPrize{}).Error; err != nil {
			return err
		}
		if len(prizes) > 0 {
			return tx.Create(&prizes).Error
		}
		return nil
	}); err != nil {
		response.Error(c, http.StatusInternalServerError, "抽奖配置保存失败")
		return
	}
	response.Success(c, lotteryConfigResponse{Enabled: req.Enabled, Prizes: prizes})
}
