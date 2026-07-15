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
	Range                       string                             `json:"range"`
	RangeStart                  string                             `json:"rangeStart"`
	RangeEnd                    string                             `json:"rangeEnd"`
	DataState                   string                             `json:"dataState"`
	UniqueSessions              int64                              `json:"uniqueSessions"`
	Funnel                      []service.PublishStatsFunnelStage  `json:"funnel"`
	DailySeries                 []service.PublishStatsDailyPoint   `json:"dailySeries"`
	Recommendation              service.PublishStatsRecommendation `json:"recommendation"`
	PlatformCode                string                             `json:"platformCode"`
	PlatformName                string                             `json:"platformName"`
	TotalPublishClicks          int64                              `json:"totalPublishClicks"`
	CurrentWeekPublishClicks    int64                              `json:"currentWeekPublishClicks"`
	CurrentMonthPublishClicks   int64                              `json:"currentMonthPublishClicks"`
	PreviousWeekPublishClicks   int64                              `json:"previousWeekPublishClicks"`
	PreviousMonthPublishClicks  int64                              `json:"previousMonthPublishClicks"`
	PublishWeekGrowthPercent    float64                            `json:"publishWeekGrowthPercent"`
	PublishMonthGrowthPercent   float64                            `json:"publishMonthGrowthPercent"`
	TotalCustomerVisits         int64                              `json:"totalCustomerVisits"`
	CurrentWeekCustomerVisits   int64                              `json:"currentWeekCustomerVisits"`
	CurrentMonthCustomerVisits  int64                              `json:"currentMonthCustomerVisits"`
	PreviousWeekCustomerVisits  int64                              `json:"previousWeekCustomerVisits"`
	PreviousMonthCustomerVisits int64                              `json:"previousMonthCustomerVisits"`
	VisitWeekGrowthPercent      float64                            `json:"visitWeekGrowthPercent"`
	VisitMonthGrowthPercent     float64                            `json:"visitMonthGrowthPercent"`
	UpdatedAt                   string                             `json:"updatedAt"`
	Timezone                    string                             `json:"timezone"`
	CurrentWeekStart            string                             `json:"currentWeekStart"`
	CurrentWeekEnd              string                             `json:"currentWeekEnd"`
	CurrentMonthStart           string                             `json:"currentMonthStart"`
	CurrentMonthEnd             string                             `json:"currentMonthEnd"`
	PlatformLinksConfigured     bool                               `json:"platformLinksConfigured"`
	ActivePlatformLinkCount     int64                              `json:"activePlatformLinkCount"`
	CrawlDataReady              bool                               `json:"crawlDataReady"`
	CrawlDataMessage            string                             `json:"crawlDataMessage"`
	WeeklyGuidedShareReady      bool                               `json:"weeklyGuidedShareReady"`
	MonthlyGuidedShareReady     bool                               `json:"monthlyGuidedShareReady"`
	WeeklyGuidedSharePercent    float64                            `json:"weeklyGuidedSharePercent"`
	MonthlyGuidedSharePercent   float64                            `json:"monthlyGuidedSharePercent"`
	DeviceStats                 service.DeviceStats                `json:"deviceStats"`
	WeeklySeries                []weeklySeriesPoint                `json:"weeklySeries"`
	MonthlySeries               []monthlySeriesPoint               `json:"monthlySeries"`
	DataSource                  string                             `json:"dataSource"`
	DataSourceLabel             string                             `json:"dataSourceLabel"`
	PartialErrors               []string                           `json:"partialErrors"`
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
	rangeSpec, err := service.ResolvePublishStatsRange(c.Query("range"), now)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "range 仅支持 7d 或 30d")
		return
	}
	weekStart := startOfWeek(now)
	monthStart := startOfMonth(now)
	platformCode, platformName, err := h.resolveStatsPlatform(store.ID, c.Query("platformCode"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "平台未启用")
		return
	}
	rows, err := service.LoadPublishStatsRows(service.GormPublishStatsQueryRunner{DB: h.DB}, store.ID, platformCode, rangeSpec)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "看板统计加载失败")
		return
	}
	var activeLinks int64
	if err := h.DB.Model(&model.StorePlatformLink{}).
		Where("store_id = ? AND status = ?", store.ID, model.StatusEnabled).
		Count(&activeLinks).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "平台链接状态加载失败")
		return
	}
	health, err := h.publishStatsHealth(*store, platformCode, activeLinks)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "看板状态加载失败")
		return
	}
	snapshot := service.BuildPublishStatsSnapshot(rangeSpec, rows, health, platformCode)
	legacy, err := loadLegacyPublishStats(gormLegacyStatsCounter{DB: h.DB, StoreID: store.ID}, platformCode, now)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "历史统计加载失败")
		return
	}
	visitAction := service.ReviewActionPageView
	devicePlatformCode := ""
	if platformCode != "" {
		visitAction = service.ReviewActionPlatformSelect
		devicePlatformCode = platformCode
	}
	deviceStats, err := service.DeviceStatsForReviewLogsByPlatform(h.DB, store.ID, visitAction, devicePlatformCode, time.Time{}, time.Time{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "设备占比加载失败")
		return
	}
	shareStats := service.CrawlGuidedShareStats(
		h.DB,
		store.ID,
		platformCode,
		weekStart,
		weekStart.AddDate(0, 0, 7),
		monthStart,
		monthStart.AddDate(0, 1, 0),
	)

	response.Success(c, publishStatsResponse{
		Range:                       snapshot.Range,
		RangeStart:                  snapshot.RangeStart,
		RangeEnd:                    snapshot.RangeEnd,
		DataState:                   snapshot.DataState,
		UniqueSessions:              snapshot.UniqueSessions,
		Funnel:                      snapshot.Funnel,
		DailySeries:                 snapshot.DailySeries,
		Recommendation:              snapshot.Recommendation,
		PlatformCode:                platformCode,
		PlatformName:                platformName,
		TotalPublishClicks:          legacy.totalClicks,
		CurrentWeekPublishClicks:    legacy.currentWeekClicks,
		CurrentMonthPublishClicks:   legacy.currentMonthClicks,
		PreviousWeekPublishClicks:   legacy.previousWeekClicks,
		PreviousMonthPublishClicks:  legacy.previousMonthClicks,
		PublishWeekGrowthPercent:    growthPercent(legacy.currentWeekClicks, legacy.previousWeekClicks),
		PublishMonthGrowthPercent:   growthPercent(legacy.currentMonthClicks, legacy.previousMonthClicks),
		TotalCustomerVisits:         legacy.totalVisits,
		CurrentWeekCustomerVisits:   legacy.currentWeekVisits,
		CurrentMonthCustomerVisits:  legacy.currentMonthVisits,
		PreviousWeekCustomerVisits:  legacy.previousWeekVisits,
		PreviousMonthCustomerVisits: legacy.previousMonthVisits,
		VisitWeekGrowthPercent:      growthPercent(legacy.currentWeekVisits, legacy.previousWeekVisits),
		VisitMonthGrowthPercent:     growthPercent(legacy.currentMonthVisits, legacy.previousMonthVisits),
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
		WeeklySeries:                legacy.weeklySeries,
		MonthlySeries:               legacy.monthlySeries,
		DataSource:                  service.ReviewAnalyticsDataSource,
		DataSourceLabel:             service.ReviewAnalyticsDataSourceLabel,
		PartialErrors:               []string{},
	})
}

func (h *Handler) publishStatsHealth(store model.Store, platformCode string, activeLinks int64) (service.PublishStatsHealth, error) {
	var boundTags int64
	if err := h.DB.Model(&model.NFCTag{}).
		Where("store_id = ? AND status = ?", store.ID, model.TagStatusBound).
		Count(&boundTags).Error; err != nil {
		return service.PublishStatsHealth{}, err
	}
	var enabledPlatforms []string
	platformQuery := h.DB.Model(&model.StorePlatformLink{}).
		Where("store_id = ? AND status = ?", store.ID, model.StatusEnabled)
	if platformCode != "" {
		platformQuery = platformQuery.Where("platform_code = ?", platformCode)
	}
	if err := platformQuery.Distinct().Pluck("platform_code", &enabledPlatforms).Error; err != nil {
		return service.PublishStatsHealth{}, err
	}
	type platformInventoryRow struct {
		PlatformCode string
		Count        int64
	}
	var inventoryRows []platformInventoryRow
	reviewQuery := h.DB.Model(&model.ReviewItem{}).
		Select("platform_style AS platform_code, COUNT(*) AS count").
		Where("store_id = ? AND status = ? AND is_dispatched = ?", store.ID, model.ReviewStatusAvailable, false)
	if platformCode != "" {
		reviewQuery = reviewQuery.Where("platform_style = ?", platformCode)
	}
	if err := reviewQuery.Group("platform_style").Scan(&inventoryRows).Error; err != nil {
		return service.PublishStatsHealth{}, err
	}
	inventoryByPlatform := make(map[string]int64, len(inventoryRows))
	for _, row := range inventoryRows {
		inventoryByPlatform[row.PlatformCode] = row.Count
	}

	latestTaskIDs := h.DB.Model(&model.ReviewGenerationTask{}).
		Select("MAX(id)").
		Where("store_id = ?", store.ID)
	if platformCode != "" {
		latestTaskIDs = latestTaskIDs.Where("platform_style = ?", platformCode)
	}
	latestTaskIDs = latestTaskIDs.Group("platform_style")
	var tasks []model.ReviewGenerationTask
	if err := h.DB.Where("id IN (?)", latestTaskIDs).Find(&tasks).Error; err != nil {
		return service.PublishStatsHealth{}, err
	}
	latestStatusByPlatform := make(map[string]string, len(enabledPlatforms))
	for _, task := range tasks {
		if _, exists := latestStatusByPlatform[task.PlatformStyle]; !exists {
			latestStatusByPlatform[task.PlatformStyle] = task.Status
		}
	}
	hasInventory, generationFailed := summarizePlatformContentHealth(enabledPlatforms, inventoryByPlatform, latestStatusByPlatform)
	return service.PublishStatsHealth{
		StoreActive:           store.Status == model.StatusEnabled,
		HasLandingRoute:       strings.TrimSpace(store.UUID) != "" && boundTags > 0,
		HasActivePlatformLink: activeLinks > 0,
		HasReviewInventory:    hasInventory,
		GenerationFailed:      generationFailed,
	}, nil
}

func summarizePlatformContentHealth(enabledPlatforms []string, inventoryByPlatform map[string]int64, latestStatusByPlatform map[string]string) (bool, bool) {
	hasInventory := len(enabledPlatforms) > 0
	generationFailed := false
	for _, platformCode := range enabledPlatforms {
		if inventoryByPlatform[platformCode] <= 0 {
			hasInventory = false
		}
		status := latestStatusByPlatform[platformCode]
		if status == model.TaskStatusFailed || status == model.TaskStatusPartialFailed {
			generationFailed = true
		}
	}
	return hasInventory, generationFailed
}

type legacyPublishStatsValues struct {
	totalClicks, currentWeekClicks, currentMonthClicks, previousWeekClicks, previousMonthClicks int64
	totalVisits, currentWeekVisits, currentMonthVisits, previousWeekVisits, previousMonthVisits int64
	weeklySeries                                                                                []weeklySeriesPoint
	monthlySeries                                                                               []monthlySeriesPoint
}

type legacyStatsCounter interface {
	Count(actionType string, platformCode string, start time.Time, end time.Time) (int64, error)
}

type gormLegacyStatsCounter struct {
	DB      *gorm.DB
	StoreID uint
}

func (counter gormLegacyStatsCounter) Count(actionType string, platformCode string, start time.Time, end time.Time) (int64, error) {
	return service.CountReviewLogActionByPlatform(counter.DB, counter.StoreID, actionType, platformCode, start, end)
}

func loadLegacyPublishStats(counter legacyStatsCounter, platformCode string, now time.Time) (legacyPublishStatsValues, error) {
	weekStart := startOfWeek(now)
	previousWeekStart := weekStart.AddDate(0, 0, -7)
	monthStart := startOfMonth(now)
	previousMonthStart := monthStart.AddDate(0, -1, 0)
	values := legacyPublishStatsValues{}
	visitAction := service.ReviewActionPageView
	visitPlatformCode := ""
	if strings.TrimSpace(platformCode) != "" {
		visitAction = service.ReviewActionPlatformSelect
		visitPlatformCode = platformCode
	}
	load := func(target *int64, actionType string, actionPlatform string, start time.Time, end time.Time) error {
		count, err := counter.Count(actionType, actionPlatform, start, end)
		if err != nil {
			return err
		}
		*target = count
		return nil
	}
	periods := []struct {
		target   *int64
		action   string
		platform string
		start    time.Time
		end      time.Time
	}{
		{&values.totalClicks, service.ReviewActionPlatformLinkClick, platformCode, time.Time{}, time.Time{}},
		{&values.totalVisits, visitAction, visitPlatformCode, time.Time{}, time.Time{}},
		{&values.currentWeekClicks, service.ReviewActionPlatformLinkClick, platformCode, weekStart, weekStart.AddDate(0, 0, 7)},
		{&values.previousWeekClicks, service.ReviewActionPlatformLinkClick, platformCode, previousWeekStart, weekStart},
		{&values.currentMonthClicks, service.ReviewActionPlatformLinkClick, platformCode, monthStart, monthStart.AddDate(0, 1, 0)},
		{&values.previousMonthClicks, service.ReviewActionPlatformLinkClick, platformCode, previousMonthStart, monthStart},
		{&values.currentWeekVisits, visitAction, visitPlatformCode, weekStart, weekStart.AddDate(0, 0, 7)},
		{&values.previousWeekVisits, visitAction, visitPlatformCode, previousWeekStart, weekStart},
		{&values.currentMonthVisits, visitAction, visitPlatformCode, monthStart, monthStart.AddDate(0, 1, 0)},
		{&values.previousMonthVisits, visitAction, visitPlatformCode, previousMonthStart, monthStart},
	}
	for _, period := range periods {
		if err := load(period.target, period.action, period.platform, period.start, period.end); err != nil {
			return legacyPublishStatsValues{}, err
		}
	}

	values.weeklySeries = make([]weeklySeriesPoint, 0, 12)
	firstWeek := weekStart.AddDate(0, 0, -11*7)
	for index := 0; index < 12; index++ {
		start := firstWeek.AddDate(0, 0, index*7)
		end := start.AddDate(0, 0, 7)
		count, err := counter.Count(service.ReviewActionPlatformLinkClick, platformCode, start, end)
		if err != nil {
			return legacyPublishStatsValues{}, err
		}
		values.weeklySeries = append(values.weeklySeries, weeklySeriesPoint{WeekStart: formatDate(start), WeekEnd: formatDate(end.AddDate(0, 0, -1)), Count: count})
	}

	values.monthlySeries = make([]monthlySeriesPoint, 0, 12)
	firstMonth := monthStart.AddDate(0, -11, 0)
	for index := 0; index < 12; index++ {
		start := firstMonth.AddDate(0, index, 0)
		end := start.AddDate(0, 1, 0)
		count, err := counter.Count(service.ReviewActionPlatformLinkClick, platformCode, start, end)
		if err != nil {
			return legacyPublishStatsValues{}, err
		}
		values.monthlySeries = append(values.monthlySeries, monthlySeriesPoint{Month: start.Format("2006-01"), Count: count})
	}
	return values, nil
}

func (h *Handler) resolveStatsPlatform(storeID uint, requested string) (string, string, error) {
	platformCode := strings.TrimSpace(requested)
	if platformCode == "" || platformCode == "all" {
		return "", "全部平台", nil
	}
	var link model.StorePlatformLink
	if err := h.DB.Where("store_id = ? AND platform_code = ? AND status = ?", storeID, platformCode, model.StatusEnabled).First(&link).Error; err != nil {
		return "", "", err
	}
	name := strings.TrimSpace(link.PlatformName)
	if name == "" {
		name = platformCode
	}
	return platformCode, name, nil
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
