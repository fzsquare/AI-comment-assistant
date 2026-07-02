package merchant

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/response"
	"ppk/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type publishStatsResponse struct {
	TotalPublishClicks          int64                `json:"totalPublishClicks"`
	CurrentWeekPublishClicks    int64                `json:"currentWeekPublishClicks"`
	CurrentMonthPublishClicks   int64                `json:"currentMonthPublishClicks"`
	PreviousWeekPublishClicks   int64                `json:"previousWeekPublishClicks"`
	PreviousMonthPublishClicks  int64                `json:"previousMonthPublishClicks"`
	PublishWeekGrowthPercent    float64              `json:"publishWeekGrowthPercent"`
	PublishMonthGrowthPercent   float64              `json:"publishMonthGrowthPercent"`
	TotalCustomerVisits         int64                `json:"totalCustomerVisits"`
	CurrentWeekCustomerVisits   int64                `json:"currentWeekCustomerVisits"`
	CurrentMonthCustomerVisits  int64                `json:"currentMonthCustomerVisits"`
	PreviousWeekCustomerVisits  int64                `json:"previousWeekCustomerVisits"`
	PreviousMonthCustomerVisits int64                `json:"previousMonthCustomerVisits"`
	VisitWeekGrowthPercent      float64              `json:"visitWeekGrowthPercent"`
	VisitMonthGrowthPercent     float64              `json:"visitMonthGrowthPercent"`
	UpdatedAt                   string               `json:"updatedAt"`
	Timezone                    string               `json:"timezone"`
	CurrentWeekStart            string               `json:"currentWeekStart"`
	CurrentWeekEnd              string               `json:"currentWeekEnd"`
	CurrentMonthStart           string               `json:"currentMonthStart"`
	CurrentMonthEnd             string               `json:"currentMonthEnd"`
	PlatformLinksConfigured     bool                 `json:"platformLinksConfigured"`
	ActivePlatformLinkCount     int64                `json:"activePlatformLinkCount"`
	CrawlDataReady              bool                 `json:"crawlDataReady"`
	CrawlDataMessage            string               `json:"crawlDataMessage"`
	WeeklyGuidedShareReady      bool                 `json:"weeklyGuidedShareReady"`
	MonthlyGuidedShareReady     bool                 `json:"monthlyGuidedShareReady"`
	WeeklyGuidedSharePercent    float64              `json:"weeklyGuidedSharePercent"`
	MonthlyGuidedSharePercent   float64              `json:"monthlyGuidedSharePercent"`
	DeviceStats                 service.DeviceStats  `json:"deviceStats"`
	WeeklySeries                []weeklySeriesPoint  `json:"weeklySeries"`
	MonthlySeries               []monthlySeriesPoint `json:"monthlySeries"`
	PartialErrors               []string             `json:"partialErrors"`
}

type weeklySeriesPoint struct {
	WeekStart string `json:"weekStart"`
	WeekEnd   string `json:"weekEnd"`
	Count     int64  `json:"count"`
}

type monthlySeriesPoint struct {
	Month string `json:"month"`
	Count int64  `json:"count"`
}

func (h *Handler) getPublishStats(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	loc := dashboardLocation()
	now := time.Now().In(loc)
	weekStart := startOfWeek(now)
	monthStart := startOfMonth(now)

	total, err := h.countPublishClicks(store.ID, time.Time{}, time.Time{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "发布次数加载失败")
		return
	}
	currentWeek, err := h.countPublishClicks(store.ID, weekStart, weekStart.AddDate(0, 0, 7))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "本周发布次数加载失败")
		return
	}
	currentMonth, err := h.countPublishClicks(store.ID, monthStart, monthStart.AddDate(0, 1, 0))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "本月发布次数加载失败")
		return
	}
	previousWeek, err := h.countPublishClicks(store.ID, weekStart.AddDate(0, 0, -7), weekStart)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "上周发布次数加载失败")
		return
	}
	previousMonth, err := h.countPublishClicks(store.ID, monthStart.AddDate(0, -1, 0), monthStart)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "上月发布次数加载失败")
		return
	}
	totalVisits, err := service.CountReviewLogAction(h.DB, store.ID, service.ReviewActionPageView, time.Time{}, time.Time{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "访问次数加载失败")
		return
	}
	currentWeekVisits, err := service.CountReviewLogAction(h.DB, store.ID, service.ReviewActionPageView, weekStart, weekStart.AddDate(0, 0, 7))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "本周访问次数加载失败")
		return
	}
	currentMonthVisits, err := service.CountReviewLogAction(h.DB, store.ID, service.ReviewActionPageView, monthStart, monthStart.AddDate(0, 1, 0))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "本月访问次数加载失败")
		return
	}
	previousWeekVisits, err := service.CountReviewLogAction(h.DB, store.ID, service.ReviewActionPageView, weekStart.AddDate(0, 0, -7), weekStart)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "上周访问次数加载失败")
		return
	}
	previousMonthVisits, err := service.CountReviewLogAction(h.DB, store.ID, service.ReviewActionPageView, monthStart.AddDate(0, -1, 0), monthStart)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "上月访问次数加载失败")
		return
	}
	deviceStats, err := service.DeviceStatsForReviewLogs(h.DB, store.ID, service.ReviewActionPageView, time.Time{}, time.Time{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "设备占比加载失败")
		return
	}
	weekly, err := h.weeklyPublishSeries(store.ID, weekStart, 12)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "周趋势加载失败")
		return
	}
	monthly, err := h.monthlyPublishSeries(store.ID, monthStart, 12)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "月趋势加载失败")
		return
	}
	var activeLinks int64
	if err := h.DB.Model(&model.StorePlatformLink{}).
		Where("store_id = ? AND status = ?", store.ID, model.StatusEnabled).
		Count(&activeLinks).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "平台链接状态加载失败")
		return
	}
	shareStats := service.CrawlGuidedShareStats(
		h.DB,
		store.ID,
		weekStart,
		weekStart.AddDate(0, 0, 7),
		monthStart,
		monthStart.AddDate(0, 1, 0),
	)

	response.Success(c, publishStatsResponse{
		TotalPublishClicks:          total,
		CurrentWeekPublishClicks:    currentWeek,
		CurrentMonthPublishClicks:   currentMonth,
		PreviousWeekPublishClicks:   previousWeek,
		PreviousMonthPublishClicks:  previousMonth,
		PublishWeekGrowthPercent:    growthPercent(currentWeek, previousWeek),
		PublishMonthGrowthPercent:   growthPercent(currentMonth, previousMonth),
		TotalCustomerVisits:         totalVisits,
		CurrentWeekCustomerVisits:   currentWeekVisits,
		CurrentMonthCustomerVisits:  currentMonthVisits,
		PreviousWeekCustomerVisits:  previousWeekVisits,
		PreviousMonthCustomerVisits: previousMonthVisits,
		VisitWeekGrowthPercent:      growthPercent(currentWeekVisits, previousWeekVisits),
		VisitMonthGrowthPercent:     growthPercent(currentMonthVisits, previousMonthVisits),
		UpdatedAt:                   now.Format(time.RFC3339),
		Timezone:                    loc.String(),
		CurrentWeekStart:            formatDate(weekStart),
		CurrentWeekEnd:              formatDate(weekStart.AddDate(0, 0, 6)),
		CurrentMonthStart:           formatDate(monthStart),
		CurrentMonthEnd:             formatDate(monthStart.AddDate(0, 1, -1)),
		PlatformLinksConfigured:     activeLinks > 0,
		ActivePlatformLinkCount:     activeLinks,
		CrawlDataReady:              shareStats.WeeklyReady || shareStats.MonthlyReady,
		CrawlDataMessage:            shareStats.Message,
		WeeklyGuidedShareReady:      shareStats.WeeklyReady,
		MonthlyGuidedShareReady:     shareStats.MonthlyReady,
		WeeklyGuidedSharePercent:    shareStats.WeeklyPercent,
		MonthlyGuidedSharePercent:   shareStats.MonthlyPercent,
		DeviceStats:                 deviceStats,
		WeeklySeries:                weekly,
		MonthlySeries:               monthly,
		PartialErrors:               []string{},
	})
}

func (h *Handler) countPublishClicks(storeID uint, start time.Time, end time.Time) (int64, error) {
	return service.CountReviewLogAction(h.DB, storeID, service.ReviewActionPlatformLinkClick, start, end)
}

func growthPercent(current int64, previous int64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100
		}
		return 0
	}
	return float64(current-previous) / float64(previous) * 100
}

func (h *Handler) weeklyPublishSeries(storeID uint, currentWeekStart time.Time, weeks int) ([]weeklySeriesPoint, error) {
	points := make([]weeklySeriesPoint, 0, weeks)
	first := currentWeekStart.AddDate(0, 0, -7*(weeks-1))
	for i := 0; i < weeks; i++ {
		start := first.AddDate(0, 0, 7*i)
		end := start.AddDate(0, 0, 7)
		count, err := h.countPublishClicks(storeID, start, end)
		if err != nil {
			return nil, err
		}
		points = append(points, weeklySeriesPoint{
			WeekStart: formatDate(start),
			WeekEnd:   formatDate(end.AddDate(0, 0, -1)),
			Count:     count,
		})
	}
	return points, nil
}

func (h *Handler) monthlyPublishSeries(storeID uint, currentMonthStart time.Time, months int) ([]monthlySeriesPoint, error) {
	points := make([]monthlySeriesPoint, 0, months)
	first := currentMonthStart.AddDate(0, -(months - 1), 0)
	for i := 0; i < months; i++ {
		start := first.AddDate(0, i, 0)
		end := start.AddDate(0, 1, 0)
		count, err := h.countPublishClicks(storeID, start, end)
		if err != nil {
			return nil, err
		}
		points = append(points, monthlySeriesPoint{
			Month: start.Format("2006-01"),
			Count: count,
		})
	}
	return points, nil
}

func dashboardLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Local
	}
	return loc
}

func startOfWeek(t time.Time) time.Time {
	t = t.In(dashboardLocation())
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	day := t.AddDate(0, 0, -(weekday - 1))
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
}

func startOfMonth(t time.Time) time.Time {
	t = t.In(dashboardLocation())
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

type generationPreferenceRequest struct {
	FocusKeywords       []string `json:"focusKeywords"`
	StyleCodes          []string `json:"styleCodes"`
	DiversityDimensions []string `json:"diversityDimensions"`
	ReferenceReviews    []string `json:"referenceReviews"`
	LengthVariance      string   `json:"lengthVariance"`
}

type generationPreferenceResponse struct {
	Configured          bool     `json:"configured"`
	FocusKeywords       []string `json:"focusKeywords"`
	StyleCodes          []string `json:"styleCodes"`
	DiversityDimensions []string `json:"diversityDimensions"`
	ReferenceReviews    []string `json:"referenceReviews"`
	LengthVariance      string   `json:"lengthVariance"`
	UpdatedAt           string   `json:"updatedAt,omitempty"`
}

func (h *Handler) getGenerationPreferences(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	payload, err := h.loadGenerationPreferenceResponse(store.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成方向加载失败")
		return
	}
	response.Success(c, payload)
}

func (h *Handler) saveGenerationPreferences(c *gin.Context) {
	store, err := h.currentStore(c)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门店不存在")
		return
	}
	var req generationPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	prefs, err := normalizeGenerationPreferenceRequest(req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	focusKeywords, _ := json.Marshal(prefs.FocusKeywords)
	styleCodes, _ := json.Marshal(prefs.StyleCodes)
	diversityDimensions, _ := json.Marshal(prefs.DiversityDimensions)
	referenceReviews, _ := json.Marshal(prefs.ReferenceReviews)

	var row model.StoreGenerationPreference
	err = h.DB.Where("store_id = ?", store.ID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = model.StoreGenerationPreference{
			StoreID:             store.ID,
			FocusKeywords:       string(focusKeywords),
			StyleCodes:          string(styleCodes),
			DiversityDimensions: string(diversityDimensions),
			ReferenceReviews:    string(referenceReviews),
			LengthVariance:      prefs.LengthVariance,
		}
		if err := h.DB.Create(&row).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "生成方向保存失败")
			return
		}
		response.Success(c, preferenceResponseFromRow(row, true))
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成方向保存失败")
		return
	}
	row.FocusKeywords = string(focusKeywords)
	row.StyleCodes = string(styleCodes)
	row.DiversityDimensions = string(diversityDimensions)
	row.ReferenceReviews = string(referenceReviews)
	row.LengthVariance = prefs.LengthVariance
	if err := h.DB.Save(&row).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "生成方向保存失败")
		return
	}
	response.Success(c, preferenceResponseFromRow(row, true))
}

func (h *Handler) loadGenerationPreferenceResponse(storeID uint) (generationPreferenceResponse, error) {
	var row model.StoreGenerationPreference
	if err := h.DB.Where("store_id = ?", storeID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return generationPreferenceResponse{
				Configured:          false,
				FocusKeywords:       []string{},
				StyleCodes:          []string{"natural"},
				DiversityDimensions: []string{"customer_identity"},
				LengthVariance:      "wide",
			}, nil
		}
		return generationPreferenceResponse{}, err
	}
	return preferenceResponseFromRow(row, true), nil
}

func preferenceResponseFromRow(row model.StoreGenerationPreference, configured bool) generationPreferenceResponse {
	diversityDimensions := decodePreferenceList(row.DiversityDimensions)
	if len(diversityDimensions) == 0 {
		diversityDimensions = []string{"customer_identity"}
	}
	return generationPreferenceResponse{
		Configured:          configured,
		FocusKeywords:       decodePreferenceList(row.FocusKeywords),
		StyleCodes:          decodePreferenceList(row.StyleCodes),
		DiversityDimensions: diversityDimensions,
		ReferenceReviews:    decodePreferenceList(row.ReferenceReviews),
		LengthVariance:      service.GenerationPreferences{LengthVariance: row.LengthVariance}.WithDefaults().LengthVariance,
		UpdatedAt:           row.UpdatedAt.Format(time.RFC3339),
	}
}

func normalizeGenerationPreferenceRequest(req generationPreferenceRequest) (service.GenerationPreferences, error) {
	focus, err := normalizeLimitedList(req.FocusKeywords, 8, 40, "重点方向")
	if err != nil {
		return service.GenerationPreferences{}, err
	}
	styles, err := normalizeStyleCodes(req.StyleCodes)
	if err != nil {
		return service.GenerationPreferences{}, err
	}
	diversityDimensions, err := normalizeDiversityDimensions(req.DiversityDimensions)
	if err != nil {
		return service.GenerationPreferences{}, err
	}
	references, err := normalizeLimitedList(req.ReferenceReviews, 5, 300, "参考评论")
	if err != nil {
		return service.GenerationPreferences{}, err
	}
	lengthVariance := strings.TrimSpace(req.LengthVariance)
	if lengthVariance == "" || lengthVariance == "default" {
		lengthVariance = "wide"
	}
	if lengthVariance != "wide" {
		return service.GenerationPreferences{}, errors.New("字数波动参数不支持")
	}
	return service.GenerationPreferences{
		FocusKeywords:       focus,
		StyleCodes:          styles,
		DiversityDimensions: diversityDimensions,
		ReferenceReviews:    references,
		LengthVariance:      lengthVariance,
	}, nil
}

func normalizeStyleCodes(values []string) ([]string, error) {
	allowed := map[string]bool{
		"natural":          true,
		"detail_rich":      true,
		"young_casual":     true,
		"restrained":       true,
		"regular_customer": true,
	}
	styles, err := normalizeLimitedList(values, 3, 40, "语气")
	if err != nil {
		return nil, err
	}
	if len(styles) == 0 {
		return []string{"natural"}, nil
	}
	for _, style := range styles {
		if !allowed[style] {
			return nil, errors.New("语气选项不支持")
		}
	}
	return styles, nil
}

func normalizeDiversityDimensions(values []string) ([]string, error) {
	allowed := map[string]bool{
		"customer_identity":    true,
		"dining_scene":         true,
		"content_angle":        true,
		"expression_structure": true,
	}
	dimensions, err := normalizeLimitedList(values, 4, 40, "多样化方向")
	if err != nil {
		return nil, err
	}
	if len(dimensions) == 0 {
		return []string{"customer_identity"}, nil
	}
	for _, dimension := range dimensions {
		if !allowed[dimension] {
			return nil, errors.New("多样化方向不支持")
		}
	}
	return dimensions, nil
}

func normalizeLimitedList(values []string, maxItems int, maxRunes int, label string) ([]string, error) {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if utf8.RuneCountInString(value) > maxRunes {
			return nil, errors.New(label + "过长")
		}
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	if len(out) > maxItems {
		return nil, errors.New(label + "数量过多")
	}
	return out, nil
}

func decodePreferenceList(raw string) []string {
	var values []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &values); err != nil {
		return []string{}
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
