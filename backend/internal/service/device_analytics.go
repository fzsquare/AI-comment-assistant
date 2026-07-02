package service

import (
	"math"
	"strings"
	"time"

	"ppk/backend/internal/model"

	"gorm.io/gorm"
)

const (
	ReviewActionPageView          = "page_view"
	ReviewActionPlatformSelect    = "platform_select"
	ReviewActionPlatformLinkClick = "platform_link_click"
)

type DeviceBreakdownItem struct {
	Code    string  `json:"code"`
	Label   string  `json:"label"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

type DeviceStats struct {
	TotalCount int64                 `json:"totalCount"`
	Items      []DeviceBreakdownItem `json:"items"`
}

type DeviceUserAgentCount struct {
	UserAgent string
	Count     int64
}

var deviceLabels = map[string]string{
	"iphone":        "苹果 iPhone",
	"ipad":          "苹果 iPad",
	"huawei":        "华为",
	"honor":         "荣耀",
	"xiaomi":        "小米/Redmi",
	"oppo":          "OPPO/一加/realme",
	"vivo":          "vivo/iQOO",
	"samsung":       "三星",
	"pixel":         "Google Pixel",
	"android_other": "Android 其他",
	"other":         "其他设备",
	"bot":           "爬虫",
	"unknown":       "未知",
}

var deviceOrder = []string{"iphone", "huawei", "honor", "xiaomi", "oppo", "vivo", "samsung", "pixel", "android_other", "ipad", "other", "bot", "unknown"}

func CountReviewLogAction(db *gorm.DB, storeID uint, actionType string, start time.Time, end time.Time) (int64, error) {
	return CountReviewLogActionByPlatform(db, storeID, actionType, "", start, end)
}

func CountReviewLogActionByPlatform(db *gorm.DB, storeID uint, actionType string, platformCode string, start time.Time, end time.Time) (int64, error) {
	query := db.Model(&model.ReviewDisplayLog{}).Where("action_type = ?", actionType)
	if storeID > 0 {
		query = query.Where("store_id = ?", storeID)
	}
	platformCode = strings.TrimSpace(platformCode)
	if platformCode != "" {
		query = query.Where("platform_code = ?", platformCode)
	}
	if !start.IsZero() {
		query = query.Where("created_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("created_at < ?", end)
	}
	var count int64
	return count, query.Count(&count).Error
}

func DeviceStatsForReviewLogs(db *gorm.DB, storeID uint, actionType string, start time.Time, end time.Time) (DeviceStats, error) {
	return DeviceStatsForReviewLogsByPlatform(db, storeID, actionType, "", start, end)
}

func DeviceStatsForReviewLogsByPlatform(db *gorm.DB, storeID uint, actionType string, platformCode string, start time.Time, end time.Time) (DeviceStats, error) {
	query := db.Model(&model.ReviewDisplayLog{}).
		Select("user_agent, COUNT(*) AS count").
		Where("action_type = ?", actionType)
	if storeID > 0 {
		query = query.Where("store_id = ?", storeID)
	}
	platformCode = strings.TrimSpace(platformCode)
	if platformCode != "" {
		query = query.Where("platform_code = ?", platformCode)
	}
	if !start.IsZero() {
		query = query.Where("created_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("created_at < ?", end)
	}

	var rows []DeviceUserAgentCount
	if err := query.Group("user_agent").Scan(&rows).Error; err != nil {
		return DeviceStats{}, err
	}
	return DeviceStatsFromUserAgents(rows), nil
}

func DeviceStatsFromUserAgents(rows []DeviceUserAgentCount) DeviceStats {
	counts := map[string]int64{}
	var total int64
	for _, row := range rows {
		if row.Count <= 0 {
			continue
		}
		code := DeviceCodeFromUserAgent(row.UserAgent)
		counts[code] += row.Count
		total += row.Count
	}

	items := make([]DeviceBreakdownItem, 0, len(deviceOrder))
	for _, code := range deviceOrder {
		count := counts[code]
		if count == 0 {
			continue
		}
		items = append(items, DeviceBreakdownItem{
			Code:    code,
			Label:   deviceLabels[code],
			Count:   count,
			Percent: percentOf(count, total),
		})
	}

	return DeviceStats{TotalCount: total, Items: items}
}

func DeviceCodeFromUserAgent(userAgent string) string {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return "unknown"
	}

	if containsAny(ua, []string{"bot", "spider", "crawler", "slurp", "bingpreview", "headless", "monitor"}) {
		return "bot"
	}
	if containsAny(ua, []string{"iphone", "ipod"}) {
		return "iphone"
	}
	if strings.Contains(ua, "ipad") {
		return "ipad"
	}
	if strings.Contains(ua, "android") {
		return androidDeviceCode(ua)
	}
	if containsAny(ua, []string{"macintosh", "windows nt", "x11", "linux x86_64", "cros", "tablet", "kindle", "silk/", "playbook"}) {
		return "other"
	}
	return "unknown"
}

func androidDeviceCode(ua string) string {
	switch {
	case containsAny(ua, []string{"huawei", "harmonyos", "hw-", "lya-", "ana-", "els-", "mate", "p40", "p50", "p60"}):
		return "huawei"
	case containsAny(ua, []string{"honor", "nth-", "any-", "bkl-", "yok-", "tny-"}):
		return "honor"
	case containsAny(ua, []string{"xiaomi", "redmi", "mi ", "mi-", "m200", "m201", "m210", "m211", "m220", "m230", "poco"}):
		return "xiaomi"
	case containsAny(ua, []string{"oppo", "oneplus", "realme", "cph", "rmx", "pgt-", "phb", "pjs"}):
		return "oppo"
	case containsAny(ua, []string{"vivo", "iqoo", "v20", "v21", "v22", "v23", "v24", "v25"}):
		return "vivo"
	case containsAny(ua, []string{"samsung", "sm-", "gt-", "galaxy"}):
		return "samsung"
	case containsAny(ua, []string{"pixel", "nexus 5", "nexus 6"}):
		return "pixel"
	default:
		return "android_other"
	}
}

func containsAny(value string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func percentOf(count int64, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round((float64(count)/float64(total))*1000) / 10
}
