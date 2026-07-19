package service

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	publishStatsBucketDay   = "day"
	publishStatsBucketRange = "range"

	PublishStatsDataEmpty        = "empty"
	PublishStatsDataAccumulating = "accumulating"
	PublishStatsDataReady        = "ready"

	PublishStatsRecommendationClosedLoop           = "closed_loop_blocker"
	PublishStatsRecommendationContent              = "content_blocker"
	PublishStatsRecommendationNoVisits             = "no_visits"
	PublishStatsRecommendationNoPlatformSelections = "no_platform_selections"
	PublishStatsRecommendationFunnelDrop           = "funnel_drop"
	PublishStatsRecommendationAccumulating         = "accumulating"
	PublishStatsRecommendationHealthy              = "healthy"
)

var publishStatsActions = []string{
	ReviewActionPageView,
	ReviewActionPlatformSelect,
	ReviewActionReviewCopy,
	ReviewActionPlatformLinkClick,
}

var publishStatsFunnelActions = []string{
	ReviewActionPageView,
	ReviewActionPlatformSelect,
	ReviewActionPlatformLinkClick,
}

type PublishStatsRange struct {
	Code  string
	Days  int
	Start time.Time
	End   time.Time
}

type PublishStatsAggregateRow struct {
	BucketType   string `gorm:"column:bucket_type"`
	Day          string `gorm:"column:day"`
	ActionType   string `gorm:"column:action_type"`
	SessionCount int64  `gorm:"column:session_count"`
}

type PublishStatsDailyPoint struct {
	Date               string `json:"date"`
	PageViews          int64  `json:"pageViews"`
	PlatformSelections int64  `json:"platformSelections"`
	ReviewCopies       int64  `json:"reviewCopies"`
	PlatformLinkClicks int64  `json:"platformLinkClicks"`
}

type PublishStatsFunnelStage struct {
	Code                string  `json:"code"`
	Label               string  `json:"label"`
	ConversionLabel     string  `json:"conversionLabel,omitempty"`
	Count               int64   `json:"count"`
	ConversionRate      float64 `json:"conversionRate"`
	ConversionAvailable bool    `json:"conversionAvailable"`
}

type PublishStatsRecommendation struct {
	Code         string `json:"code"`
	Title        string `json:"title"`
	Message      string `json:"message"`
	ActionLabel  string `json:"actionLabel"`
	ActionTarget string `json:"actionTarget"`
}

type PublishStatsHealth struct {
	StoreActive           bool
	HasLandingRoute       bool
	HasActivePlatformLink bool
	HasReviewInventory    bool
	GenerationFailed      bool
}

type PublishStatsSnapshot struct {
	Range          string                     `json:"range"`
	RangeStart     string                     `json:"rangeStart"`
	RangeEnd       string                     `json:"rangeEnd"`
	DataState      string                     `json:"dataState"`
	UniqueSessions int64                      `json:"uniqueSessions"`
	Funnel         []PublishStatsFunnelStage  `json:"funnel"`
	DailySeries    []PublishStatsDailyPoint   `json:"dailySeries"`
	Recommendation PublishStatsRecommendation `json:"recommendation"`
}

type PublishStatsQueryRunner interface {
	Query(query string, args ...any) ([]PublishStatsAggregateRow, error)
}

type GormPublishStatsQueryRunner struct {
	DB *gorm.DB
}

func (r GormPublishStatsQueryRunner) Query(query string, args ...any) ([]PublishStatsAggregateRow, error) {
	var rows []PublishStatsAggregateRow
	if r.DB == nil {
		return nil, errors.New("publish stats database is nil")
	}
	if err := r.DB.Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func dashboardStatsLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return loc
}

func ResolvePublishStatsRange(raw string, now time.Time) (PublishStatsRange, error) {
	code := strings.TrimSpace(raw)
	if code == "" {
		code = "7d"
	}
	days := 0
	switch code {
	case "7d":
		days = 7
	case "30d":
		days = 30
	default:
		return PublishStatsRange{}, fmt.Errorf("unsupported publish stats range %q", code)
	}
	loc := dashboardStatsLocation()
	localNow := now.In(loc)
	today := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, loc)
	return PublishStatsRange{
		Code:  code,
		Days:  days,
		Start: today.AddDate(0, 0, -(days - 1)),
		End:   today.AddDate(0, 0, 1),
	}, nil
}

func LoadPublishStatsRows(runner PublishStatsQueryRunner, storeID uint, platformCode string, rangeSpec PublishStatsRange) ([]PublishStatsAggregateRow, error) {
	if runner == nil {
		return nil, errors.New("publish stats query runner is nil")
	}
	query, args := buildPublishStatsQuery(storeID, platformCode, rangeSpec)
	rows, err := runner.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query publish stats: %w", err)
	}
	return rows, nil
}

func buildPublishStatsQuery(storeID uint, platformCode string, rangeSpec PublishStatsRange) (string, []any) {
	platformCode = strings.TrimSpace(platformCode)
	if platformCode != "" {
		return buildPlatformPublishStatsQuery(storeID, platformCode, rangeSpec)
	}
	baseArgs := func() []any {
		args := []any{storeID}
		for _, action := range publishStatsActions {
			args = append(args, action)
		}
		args = append(args, rangeSpec.Start, rangeSpec.End)
		return args
	}
	query := `SELECT 'day' AS bucket_type, DATE(created_at) AS day, action_type,
       COUNT(DISTINCT session_id) AS session_count
FROM review_display_logs
WHERE store_id = ?
  AND action_type IN (?, ?, ?, ?)
  AND created_at >= ? AND created_at < ?
GROUP BY DATE(created_at), action_type
UNION ALL
SELECT 'range' AS bucket_type, '' AS day, action_type,
       COUNT(DISTINCT session_id) AS session_count
FROM review_display_logs
WHERE store_id = ?
  AND action_type IN (?, ?, ?, ?)
  AND created_at >= ? AND created_at < ?
GROUP BY action_type
ORDER BY bucket_type, day, action_type`
	args := append(baseArgs(), baseArgs()...)
	return query, args
}

func buildPlatformPublishStatsQuery(storeID uint, platformCode string, rangeSpec PublishStatsRange) (string, []any) {
	query := `SELECT 'day' AS bucket_type, DATE(created_at) AS day, action_type,
       COUNT(DISTINCT session_id) AS session_count
FROM review_display_logs
WHERE store_id = ? AND action_type = ?
  AND created_at >= ? AND created_at < ?
GROUP BY DATE(created_at), action_type
UNION ALL
SELECT 'day' AS bucket_type, DATE(created_at) AS day, action_type,
       COUNT(DISTINCT session_id) AS session_count
FROM review_display_logs
WHERE store_id = ? AND platform_code = ?
  AND action_type IN (?, ?, ?)
  AND created_at >= ? AND created_at < ?
GROUP BY DATE(created_at), action_type
UNION ALL
SELECT 'range' AS bucket_type, '' AS day, action_type,
       COUNT(DISTINCT session_id) AS session_count
FROM review_display_logs
WHERE store_id = ? AND action_type = ?
  AND created_at >= ? AND created_at < ?
GROUP BY action_type
UNION ALL
SELECT 'range' AS bucket_type, '' AS day, action_type,
       COUNT(DISTINCT session_id) AS session_count
FROM review_display_logs
WHERE store_id = ? AND platform_code = ?
  AND action_type IN (?, ?, ?)
  AND created_at >= ? AND created_at < ?
GROUP BY action_type
ORDER BY bucket_type, day, action_type`
	sharedArgs := []any{storeID, ReviewActionPageView, rangeSpec.Start, rangeSpec.End}
	platformArgs := []any{
		storeID, platformCode,
		ReviewActionPlatformSelect, ReviewActionReviewCopy, ReviewActionPlatformLinkClick,
		rangeSpec.Start, rangeSpec.End,
	}
	args := append([]any{}, sharedArgs...)
	args = append(args, platformArgs...)
	args = append(args, sharedArgs...)
	args = append(args, platformArgs...)
	return query, args
}

func BuildPublishStatsSnapshot(rangeSpec PublishStatsRange, rows []PublishStatsAggregateRow, health PublishStatsHealth, platformCodes ...string) PublishStatsSnapshot {
	platformCode := ""
	if len(platformCodes) > 0 {
		platformCode = strings.TrimSpace(platformCodes[0])
	}
	daily := make([]PublishStatsDailyPoint, 0, rangeSpec.Days)
	byDate := make(map[string]*PublishStatsDailyPoint, rangeSpec.Days)
	for day := rangeSpec.Start; day.Before(rangeSpec.End); day = day.AddDate(0, 0, 1) {
		point := PublishStatsDailyPoint{Date: day.Format("2006-01-02")}
		daily = append(daily, point)
		byDate[point.Date] = &daily[len(daily)-1]
	}
	rangeCounts := map[string]int64{}
	for _, row := range rows {
		if row.SessionCount < 0 {
			continue
		}
		if row.BucketType == publishStatsBucketRange {
			rangeCounts[row.ActionType] = row.SessionCount
			continue
		}
		if row.BucketType != publishStatsBucketDay {
			continue
		}
		point := byDate[row.Day]
		if point == nil {
			continue
		}
		setDailyActionCount(point, row.ActionType, row.SessionCount)
	}

	funnel := make([]PublishStatsFunnelStage, 0, len(publishStatsFunnelActions))
	labels := map[string]string{
		ReviewActionPageView:          "贴卡访问",
		ReviewActionPlatformSelect:    "选择平台",
		ReviewActionPlatformLinkClick: "平台点击",
	}
	for index, action := range publishStatsFunnelActions {
		stage := PublishStatsFunnelStage{Code: action, Label: labels[action], Count: rangeCounts[action]}
		if platformCode != "" && action == ReviewActionPlatformSelect {
			stage.ConversionLabel = "全部访问中选择该平台"
		}
		if index > 0 {
			previous := rangeCounts[publishStatsFunnelActions[index-1]]
			if previous > 0 {
				stage.ConversionAvailable = true
				stage.ConversionRate = roundOneDecimal(float64(stage.Count) / float64(previous) * 100)
			}
		}
		funnel = append(funnel, stage)
	}
	uniqueSessions := rangeCounts[ReviewActionPageView]
	if platformCode != "" {
		uniqueSessions = rangeCounts[ReviewActionPlatformSelect]
	}
	dataState := PublishStatsDataReady
	if uniqueSessions == 0 {
		dataState = PublishStatsDataEmpty
	} else if uniqueSessions < 20 {
		dataState = PublishStatsDataAccumulating
	}
	snapshot := PublishStatsSnapshot{
		Range:          rangeSpec.Code,
		RangeStart:     rangeSpec.Start.Format("2006-01-02"),
		RangeEnd:       rangeSpec.End.AddDate(0, 0, -1).Format("2006-01-02"),
		DataState:      dataState,
		UniqueSessions: uniqueSessions,
		Funnel:         funnel,
		DailySeries:    daily,
	}
	snapshot.Recommendation = choosePublishStatsRecommendation(snapshot, health, platformCode)
	return snapshot
}

func setDailyActionCount(point *PublishStatsDailyPoint, action string, count int64) {
	switch action {
	case ReviewActionPageView:
		point.PageViews = count
	case ReviewActionPlatformSelect:
		point.PlatformSelections = count
	case ReviewActionReviewCopy:
		point.ReviewCopies = count
	case ReviewActionPlatformLinkClick:
		point.PlatformLinkClicks = count
	}
}

func choosePublishStatsRecommendation(snapshot PublishStatsSnapshot, health PublishStatsHealth, platformCode string) PublishStatsRecommendation {
	if !health.StoreActive || !health.HasLandingRoute || !health.HasActivePlatformLink {
		return PublishStatsRecommendation{
			Code: PublishStatsRecommendationClosedLoop, Title: "先恢复顾客评价闭环",
			Message:     "门店状态、NFC 入口或平台官方链接尚未完整可用，顾客可能无法完成平台跳转。",
			ActionLabel: "检查跳转任务", ActionTarget: "platform-links",
		}
	}
	if !health.HasReviewInventory || health.GenerationFailed {
		return PublishStatsRecommendation{
			Code: PublishStatsRecommendationContent, Title: "先补充可用评价",
			Message:     "当前评价库存不足或最近生成任务失败，顾客进入后可能拿不到符合体验的评价。",
			ActionLabel: "检查评价管理", ActionTarget: "reviews",
		}
	}
	if snapshot.UniqueSessions == 0 {
		if platformCode != "" && len(snapshot.Funnel) > 0 && snapshot.Funnel[0].Count > 0 {
			return PublishStatsRecommendation{
				Code: PublishStatsRecommendationNoPlatformSelections, Title: "还没有顾客选择该平台",
				Message:     "当前范围内已有贴卡访问，但还没有顾客选择该平台，先检查平台名称、入口配置和店员引导。",
				ActionLabel: "检查平台入口", ActionTarget: "platform-links",
			}
		}
		return PublishStatsRecommendation{
			Code: PublishStatsRecommendationNoVisits, Title: "先让顾客开始贴卡访问",
			Message:     "当前范围内还没有贴卡访问，先检查 NFC 摆放位置和店员引导。",
			ActionLabel: "查看 NFC 使用建议", ActionTarget: "nfc-guidance",
		}
	}
	if snapshot.UniqueSessions >= 20 {
		if stage, ok := largestFunnelDrop(snapshot.Funnel); ok {
			return PublishStatsRecommendation{
				Code: PublishStatsRecommendationFunnelDrop, Title: "优先改善“" + stage.Label + "”",
				Message:     "这一环节是当前范围内最大的顾客流失点，先优化它最可能提升平台点击转化。",
				ActionLabel: "查看漏斗", ActionTarget: "funnel",
			}
		}
	}
	if snapshot.UniqueSessions < 20 {
		return PublishStatsRecommendation{
			Code: PublishStatsRecommendationAccumulating, Title: "数据积累中",
			Message:     "当前唯一会话少于 20 个，先继续积累真实使用数据，再判断优化方向。",
			ActionLabel: "查看原始数量", ActionTarget: "funnel",
		}
	}
	return PublishStatsRecommendation{
		Code: PublishStatsRecommendationHealthy, Title: "评价流程运行稳定",
		Message:     "当前没有明显单点掉落，继续观察平台点击变化。",
		ActionLabel: "查看趋势", ActionTarget: "trend",
	}
}

func largestFunnelDrop(stages []PublishStatsFunnelStage) (PublishStatsFunnelStage, bool) {
	var selected PublishStatsFunnelStage
	largestDrop := int64(0)
	startIndex := 1
	if len(stages) > 1 && stages[1].ConversionLabel != "" {
		startIndex = 2
	}
	for index := startIndex; index < len(stages); index++ {
		drop := stages[index-1].Count - stages[index].Count
		if drop > largestDrop {
			largestDrop = drop
			selected = stages[index]
		}
	}
	return selected, largestDrop > 0
}

func roundOneDecimal(value float64) float64 {
	return math.Round(value*10) / 10
}
