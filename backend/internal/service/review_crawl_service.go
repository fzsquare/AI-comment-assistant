package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ppk/backend/internal/config"
	"ppk/backend/internal/model"

	"gorm.io/gorm"
)

const (
	ReviewCrawlPlatformMeituan = "meituan"
	reviewCrawlWindowDays      = 7
)

type ReviewCrawlService struct {
	DB     *gorm.DB
	Config config.Config
	Client *ReviewCrawlClient
	Now    func() time.Time
}

type ReviewCrawlRunResult struct {
	Batch   *model.StoreReviewCrawlBatch `json:"batch,omitempty"`
	Skipped bool                         `json:"skipped"`
	Message string                       `json:"message"`
}

func NewReviewCrawlService(db *gorm.DB, cfg config.Config) *ReviewCrawlService {
	return &ReviewCrawlService{
		DB:     db,
		Config: cfg,
		Client: NewReviewCrawlClient(cfg),
		Now:    time.Now,
	}
}

func NormalizeReviewCrawlPlatform(platform string) (string, error) {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		platform = ReviewCrawlPlatformMeituan
	}
	if platform != ReviewCrawlPlatformMeituan {
		return "", errors.New("第一版仅支持美团评论采集")
	}
	return platform, nil
}

func (s *ReviewCrawlService) RunForStore(ctx context.Context, storeID uint, triggerType string) (ReviewCrawlRunResult, error) {
	if s == nil || s.DB == nil {
		return ReviewCrawlRunResult{}, errors.New("review crawl service unavailable")
	}
	var cfg model.StoreReviewCrawlConfig
	if err := s.DB.Where("store_id = ?", storeID).First(&cfg).Error; err != nil {
		return ReviewCrawlRunResult{}, errors.New("门店未配置评论采集")
	}
	return s.RunConfig(ctx, cfg, triggerType)
}

func (s *ReviewCrawlService) RunDueCrawls(ctx context.Context, limit int) (int, error) {
	if s == nil || s.DB == nil || strings.TrimSpace(s.Config.ReviewCrawlServiceURL) == "" {
		return 0, nil
	}
	if limit <= 0 {
		limit = 20
	}
	now := s.now()
	var configs []model.StoreReviewCrawlConfig
	if err := s.DB.Table("store_review_crawl_configs").
		Select("store_review_crawl_configs.*").
		Joins("JOIN stores ON stores.id = store_review_crawl_configs.store_id").
		Joins("JOIN merchant_users ON merchant_users.id = stores.merchant_user_id").
		Where("store_review_crawl_configs.enabled = ? AND (store_review_crawl_configs.next_crawl_at IS NULL OR store_review_crawl_configs.next_crawl_at <= ?) AND stores.status = ? AND merchant_users.status = ?", true, now, model.StatusEnabled, model.StatusEnabled).
		Order("store_review_crawl_configs.next_crawl_at asc, store_review_crawl_configs.id asc").
		Limit(limit).
		Find(&configs).Error; err != nil {
		return 0, err
	}
	count := 0
	for _, cfg := range configs {
		if _, err := s.RunConfig(ctx, cfg, model.CrawlTriggerScheduled); err == nil {
			count++
		}
	}
	return count, nil
}

func (s *ReviewCrawlService) StartScheduler(ctx context.Context) {
	if s == nil || strings.TrimSpace(s.Config.ReviewCrawlServiceURL) == "" {
		return
	}
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			_, _ = s.RunDueCrawls(ctx, 20)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *ReviewCrawlService) RunConfig(ctx context.Context, cfg model.StoreReviewCrawlConfig, triggerType string) (ReviewCrawlRunResult, error) {
	if !cfg.Enabled {
		return ReviewCrawlRunResult{}, errors.New("门店未启用评论采集")
	}
	if strings.TrimSpace(cfg.ExternalShopID) == "" {
		return ReviewCrawlRunResult{}, errors.New("请先配置美团商家 ID")
	}
	if strings.TrimSpace(s.Config.ReviewCrawlServiceURL) == "" {
		_ = s.markConfigFailure(cfg.ID, "service_not_configured", "crawl_start", "评论采集服务未配置")
		return ReviewCrawlRunResult{}, errors.New("评论采集服务未配置")
	}
	triggerType = strings.TrimSpace(triggerType)
	if triggerType == "" {
		triggerType = model.CrawlTriggerManual
	}

	now := s.now()
	if triggerType == model.CrawlTriggerManual && cfg.LastStatus == model.CrawlStatusSuccess && cfg.NextCrawlAt != nil && cfg.NextCrawlAt.After(now) {
		return ReviewCrawlRunResult{Skipped: true, Message: "本周期已采集"}, nil
	}

	var running int64
	if err := s.DB.Model(&model.StoreReviewCrawlBatch{}).
		Where("config_id = ? AND status = ?", cfg.ID, model.CrawlStatusRunning).
		Count(&running).Error; err != nil {
		return ReviewCrawlRunResult{}, err
	}
	if running > 0 {
		return ReviewCrawlRunResult{}, errors.New("评论采集任务正在运行")
	}

	attemptNo := 1
	var previous int64
	_ = s.DB.Model(&model.StoreReviewCrawlBatch{}).Where("config_id = ?", cfg.ID).Count(&previous).Error
	if previous > 0 {
		attemptNo = int(previous) + 1
	}
	windowEnd := now
	windowStart := now.AddDate(0, 0, -reviewCrawlWindowDays)
	isBaseline := cfg.BaselineCompletedAt == nil
	batch := model.StoreReviewCrawlBatch{
		ConfigID:               cfg.ID,
		StoreID:                cfg.StoreID,
		PlatformCode:           cfg.PlatformCode,
		ExternalShopIDSnapshot: cfg.ExternalShopID,
		TriggerType:            triggerType,
		AttemptNo:              attemptNo,
		IsBaseline:             isBaseline,
		WindowDays:             reviewCrawlWindowDays,
		WindowStartAt:          &windowStart,
		WindowEndAt:            &windowEnd,
		StartedAt:              &now,
		Status:                 model.CrawlStatusRunning,
	}
	if err := s.DB.Create(&batch).Error; err != nil {
		return ReviewCrawlRunResult{}, err
	}
	_ = s.DB.Model(&model.StoreReviewCrawlConfig{}).Where("id = ?", cfg.ID).Updates(map[string]any{
		"last_status":        model.CrawlStatusRunning,
		"last_error_message": "",
	}).Error

	rows, runErr := s.fetchRows(ctx, cfg.ExternalShopID)
	if runErr != nil {
		_ = s.failBatch(&batch, runErr)
		return ReviewCrawlRunResult{Batch: &batch}, runErr
	}

	if err := s.importRows(&cfg, &batch, rows); err != nil {
		crawlErr := crawlFailure("import_failed", "import", true, err.Error())
		_ = s.failBatch(&batch, crawlErr)
		return ReviewCrawlRunResult{Batch: &batch}, crawlErr
	}
	return ReviewCrawlRunResult{Batch: &batch, Message: "采集完成"}, nil
}

func (s *ReviewCrawlService) fetchRows(ctx context.Context, externalShopID string) ([]ReviewCrawlImportRow, error) {
	if err := s.Client.Start(ctx, externalShopID); err != nil {
		return nil, crawlFailure("start_failed", "crawl_start", true, err.Error())
	}
	ready := false
	var lastStatus string
	for attempt := 0; attempt < s.Config.ReviewCrawlPollMaxAttempts; attempt++ {
		status, err := s.Client.Status(ctx, externalShopID)
		if err != nil {
			return nil, crawlFailure("status_failed", "status_poll", true, err.Error())
		}
		lastStatus = status
		if strings.EqualFold(status, "true") {
			ready = true
			break
		}
		timer := time.NewTimer(time.Duration(s.Config.ReviewCrawlPollIntervalSeconds) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, crawlFailure("context_canceled", "status_poll", true, ctx.Err().Error())
		case <-timer.C:
		}
	}
	if !ready {
		return nil, crawlFailure("status_timeout", "status_poll", true, "评论采集任务超时，最后状态："+lastStatus)
	}
	data, err := s.Client.Download(ctx, externalShopID)
	if err != nil {
		return nil, crawlFailure("download_failed", "download", true, err.Error())
	}
	rows, err := ParseReviewCrawlXLSX(data)
	if err != nil {
		return nil, crawlFailure("parse_failed", "parse", false, err.Error())
	}
	return rows, nil
}

func (s *ReviewCrawlService) importRows(cfg *model.StoreReviewCrawlConfig, batch *model.StoreReviewCrawlBatch, rows []ReviewCrawlImportRow) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		reviews := make([]model.ExternalStoreReview, 0, len(rows))
		matched := 0
		for _, row := range rows {
			review := model.ExternalStoreReview{
				BatchID:          batch.ID,
				StoreID:          cfg.StoreID,
				PlatformCode:     cfg.PlatformCode,
				SourceReviewRef:  row.SourceReviewRef,
				UserName:         row.UserName,
				RatingRaw:        row.RatingRaw,
				RatingNormalized: row.RatingNormalized,
				ReviewTime:       row.ReviewTime,
				Content:          row.Content,
				IsBaseline:       batch.IsBaseline,
			}
			match := s.matchExternalReview(tx, review)
			if match.MatchedFeedbackID != nil {
				review.MatchedFeedbackID = match.MatchedFeedbackID
				review.MatchedReviewItemID = match.MatchedReviewItemID
				review.MatchScore = match.Score
				review.MatchReason = match.Reason
				review.MatchSource = match.Source
				review.MatchAlgorithmVersion = match.AlgorithmVersion
				matched++
			}
			reviews = append(reviews, review)
		}
		if len(reviews) > 0 {
			if err := tx.Create(&reviews).Error; err != nil {
				return err
			}
		}
		finished := s.now()
		updates := map[string]any{
			"finished_at":          finished,
			"status":               model.CrawlStatusSuccess,
			"raw_row_count":        len(rows),
			"inserted_row_count":   len(reviews),
			"matched_review_count": matched,
			"failure_code":         "",
			"failure_stage":        "",
			"retryable":            false,
			"error_message":        "",
		}
		if err := tx.Model(&model.StoreReviewCrawlBatch{}).Where("id = ?", batch.ID).Updates(updates).Error; err != nil {
			return err
		}
		batch.FinishedAt = &finished
		batch.Status = model.CrawlStatusSuccess
		batch.RawRowCount = len(rows)
		batch.InsertedRowCount = len(reviews)
		batch.MatchedReviewCount = matched

		next := finished.AddDate(0, 0, reviewCrawlWindowDays)
		configUpdates := map[string]any{
			"last_status":        model.CrawlStatusSuccess,
			"last_error_message": "",
			"last_crawled_at":    finished,
			"next_crawl_at":      next,
		}
		if cfg.BaselineCompletedAt == nil {
			configUpdates["baseline_completed_at"] = finished
		}
		return tx.Model(&model.StoreReviewCrawlConfig{}).Where("id = ?", cfg.ID).Updates(configUpdates).Error
	})
}

type externalReviewMatch struct {
	MatchedFeedbackID   *uint
	MatchedReviewItemID *uint
	Score               float64
	Reason              string
	Source              string
	AlgorithmVersion    string
}

func (s *ReviewCrawlService) matchExternalReview(tx *gorm.DB, review model.ExternalStoreReview) externalReviewMatch {
	content := strings.TrimSpace(review.Content)
	if content == "" {
		return externalReviewMatch{}
	}
	var rows []model.ReviewFeedback
	query := tx.Where("store_id = ? AND platform_code = ? AND feedback_type = ?", review.StoreID, review.PlatformCode, model.ReviewFeedbackAccepted).
		Order("created_at desc").
		Limit(100)
	if review.ReviewTime != nil {
		end := review.ReviewTime.AddDate(0, 0, 14)
		query = query.Where("created_at <= ?", end)
	}
	if err := query.Find(&rows).Error; err != nil {
		return externalReviewMatch{}
	}
	best := externalReviewMatch{}
	for _, row := range rows {
		candidate := strings.TrimSpace(row.EditedContent)
		source := "edited_content"
		if candidate == "" {
			candidate = strings.TrimSpace(row.ContentSnapshot)
			source = "content_snapshot"
		}
		result := CompareReviewText(content, candidate)
		if !result.Matched || result.Score < best.Score {
			continue
		}
		feedbackID := row.ID
		reviewItemID := row.ReviewItemID
		best = externalReviewMatch{
			MatchedFeedbackID:   &feedbackID,
			MatchedReviewItemID: &reviewItemID,
			Score:               result.Score,
			Reason:              result.Reason,
			Source:              source,
			AlgorithmVersion:    result.AlgorithmVersion,
		}
	}
	return best
}

func (s *ReviewCrawlService) failBatch(batch *model.StoreReviewCrawlBatch, err error) error {
	cerr := asCrawlFailure(err)
	finished := s.now()
	batch.FinishedAt = &finished
	batch.Status = model.CrawlStatusFailed
	batch.FailureCode = cerr.Code
	batch.FailureStage = cerr.Stage
	batch.Retryable = cerr.Retryable
	batch.ErrorMessage = cerr.Message
	if dbErr := s.DB.Model(&model.StoreReviewCrawlBatch{}).Where("id = ?", batch.ID).Updates(map[string]any{
		"finished_at":   finished,
		"status":        model.CrawlStatusFailed,
		"failure_code":  cerr.Code,
		"failure_stage": cerr.Stage,
		"retryable":     cerr.Retryable,
		"error_message": cerr.Message,
	}).Error; dbErr != nil {
		return dbErr
	}
	return s.markConfigFailure(batch.ConfigID, cerr.Code, cerr.Stage, cerr.Message)
}

func (s *ReviewCrawlService) markConfigFailure(configID uint, _ string, _ string, message string) error {
	return s.DB.Model(&model.StoreReviewCrawlConfig{}).Where("id = ?", configID).Updates(map[string]any{
		"last_status":        model.CrawlStatusFailed,
		"last_error_message": message,
	}).Error
}

func (s *ReviewCrawlService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

type CrawlShareStats struct {
	WeeklyReady    bool
	MonthlyReady   bool
	WeeklyPercent  float64
	MonthlyPercent float64
	Message        string
}

func CrawlGuidedShareStats(db *gorm.DB, storeID uint, weekStart time.Time, weekEnd time.Time, monthStart time.Time, monthEnd time.Time) CrawlShareStats {
	stats := CrawlShareStats{Message: "数据积累中"}
	if db == nil || storeID == 0 {
		return stats
	}
	var cfg model.StoreReviewCrawlConfig
	if err := db.Where("store_id = ? AND enabled = ?", storeID, true).First(&cfg).Error; err != nil {
		return stats
	}
	if cfg.BaselineCompletedAt == nil || cfg.LastStatus != model.CrawlStatusSuccess {
		return stats
	}
	stats.WeeklyReady, stats.WeeklyPercent = crawlGuidedShareForPeriod(db, storeID, cfg.PlatformCode, maxTime(weekStart, *cfg.BaselineCompletedAt), weekEnd)
	stats.MonthlyReady, stats.MonthlyPercent = crawlGuidedShareForPeriod(db, storeID, cfg.PlatformCode, maxTime(monthStart, *cfg.BaselineCompletedAt), monthEnd)
	if stats.WeeklyReady || stats.MonthlyReady {
		stats.Message = ""
	}
	return stats
}

func crawlGuidedShareForPeriod(db *gorm.DB, storeID uint, platformCode string, start time.Time, end time.Time) (bool, float64) {
	var totalReviews int64
	if err := db.Model(&model.ExternalStoreReview{}).
		Where("store_id = ? AND platform_code = ? AND is_baseline = ? AND review_time >= ? AND review_time < ?", storeID, platformCode, false, start, end).
		Count(&totalReviews).Error; err != nil || totalReviews <= 0 {
		return false, 0
	}
	var clicks int64
	if err := db.Model(&model.ReviewDisplayLog{}).
		Where("store_id = ? AND platform_code = ? AND action_type = ? AND created_at >= ? AND created_at < ?", storeID, platformCode, ReviewActionPlatformLinkClick, start, end).
		Count(&clicks).Error; err != nil {
		return false, 0
	}
	return true, float64(clicks) / float64(totalReviews) * 100
}

func maxTime(a time.Time, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

type crawlFailureError struct {
	Code      string
	Stage     string
	Retryable bool
	Message   string
}

func (e crawlFailureError) Error() string {
	return e.Message
}

func crawlFailure(code string, stage string, retryable bool, message string) crawlFailureError {
	return crawlFailureError{Code: code, Stage: stage, Retryable: retryable, Message: message}
}

func asCrawlFailure(err error) crawlFailureError {
	var cerr crawlFailureError
	if errors.As(err, &cerr) {
		return cerr
	}
	return crawlFailure("unknown", "unknown", true, err.Error())
}

type ReviewCrawlClient struct {
	baseURL          string
	httpClient       *http.Client
	maxDownloadBytes int64
}

func NewReviewCrawlClient(cfg config.Config) *ReviewCrawlClient {
	return &ReviewCrawlClient{
		baseURL: strings.TrimRight(cfg.ReviewCrawlServiceURL, "/"),
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.ReviewCrawlHTTPTimeoutSeconds) * time.Second,
			Transport: &http.Transport{
				Proxy: nil,
			},
		},
		maxDownloadBytes: int64(cfg.ReviewCrawlMaxDownloadBytes),
	}
}

func (c *ReviewCrawlClient) Start(ctx context.Context, externalShopID string) error {
	payload, _ := json.Marshal(map[string]string{"id": externalShopID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/crawl", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("crawl start status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *ReviewCrawlClient) Status(ctx context.Context, externalShopID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/status/"+externalShopID, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("status response %d", resp.StatusCode)
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.Status, nil
}

func (c *ReviewCrawlClient) Download(ctx context.Context, externalShopID string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/download/"+externalShopID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download response %d", resp.StatusCode)
	}
	limit := c.maxDownloadBytes
	if limit <= 0 {
		limit = 5 * 1024 * 1024
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("download exceeds limit %d bytes", limit)
	}
	return data, nil
}
