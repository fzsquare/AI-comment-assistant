package public

import (
	"crypto/rand"
	"errors"
	"math/big"
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

type lotteryDrawRequest struct {
	SessionID string `json:"sessionId"`
}

type lotteryDrawResponse struct {
	Enabled       bool   `json:"enabled"`
	Drawn         bool   `json:"drawn"`
	Won           bool   `json:"won"`
	PrizeName     string `json:"prizeName"`
	PrizeImageURL string `json:"prizeImageUrl"`
}

func (h *Handler) drawLottery(c *gin.Context) {
	var req lotteryDrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	if req.SessionID == "" {
		response.Error(c, http.StatusBadRequest, "会话不能为空")
		return
	}
	var store model.Store
	if err := h.DB.Where("uuid = ?", c.Param("uuid")).First(&store).Error; err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	if err := h.ReviewPool.VerifyLandingSession(store.UUID, req.SessionID, time.Now()); err != nil {
		response.Error(c, http.StatusBadRequest, "会话已失效，请刷新页面后重试")
		return
	}
	result, err := h.drawLotteryForSession(store, req.SessionID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *Handler) drawLotteryForSession(store model.Store, sessionID string) (lotteryDrawResponse, error) {
	result := lotteryDrawResponse{}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var existing model.StoreLotteryDraw
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("store_id = ? AND session_id = ?", store.ID, sessionID).
			First(&existing).Error; err == nil {
			result = lotteryResultFromDraw(existing)
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var config model.StoreLotteryConfig
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("store_id = ?", store.ID).First(&config).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			result.Enabled = false
			return nil
		} else if err != nil {
			return err
		}
		if !config.Enabled {
			result.Enabled = false
			return nil
		}

		var copiedCount int64
		if err := lotteryEligibilityQuery(tx, store.ID, sessionID).Count(&copiedCount).Error; err != nil {
			return err
		}
		if copiedCount == 0 {
			return errors.New("完成复制并打开门店页面后才能抽奖")
		}

		var prizes []model.StoreLotteryPrize
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("store_id = ? AND enabled = ? AND stock > 0", store.ID, true).
			Order("sort_no asc, id asc").Find(&prizes).Error; err != nil {
			return err
		}
		roll, err := lotteryRoll()
		if err != nil {
			return err
		}
		candidates := make([]service.LotteryPrizeCandidate, 0, len(prizes))
		for _, prize := range prizes {
			candidates = append(candidates, service.LotteryPrizeCandidate{ID: prize.ID, Stock: prize.Stock, WinRate: prize.WinRate})
		}
		candidate, won := service.ChooseLotteryPrize(candidates, roll)
		draw := model.StoreLotteryDraw{StoreID: store.ID, SessionID: sessionID, Outcome: "lost"}
		if won {
			for _, prize := range prizes {
				if prize.ID != candidate.ID {
					continue
				}
				draw.Outcome = "won"
				draw.PrizeID = &prize.ID
				draw.PrizeName = prize.Name
				draw.PrizeImageURL = prize.ImageURL
				if err := tx.Model(&model.StoreLotteryPrize{}).Where("id = ? AND stock > 0", prize.ID).Update("stock", gorm.Expr("stock - 1")).Error; err != nil {
					return err
				}
				break
			}
		}
		if err := tx.Create(&draw).Error; err != nil {
			return err
		}
		result = lotteryResultFromDraw(draw)
		return nil
	})
	return result, err
}

func lotteryEligibilityQuery(db *gorm.DB, storeID uint, sessionID string) *gorm.DB {
	return db.Model(&model.ReviewDisplayLog{}).
		Where("store_id = ? AND session_id = ? AND action_type = ?", storeID, sessionID, "review_copy")
}

func lotteryResultFromDraw(draw model.StoreLotteryDraw) lotteryDrawResponse {
	return lotteryDrawResponse{Enabled: true, Drawn: true, Won: draw.Outcome == "won", PrizeName: draw.PrizeName, PrizeImageURL: draw.PrizeImageURL}
}

func lotteryRoll() (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()) + 1, nil
}
