package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ppk/backend/internal/model"
)

// 文案服务响应体上限，防止异常/被劫持的端点流式返回超大 body 耗尽内存
const maxAgentRespBytes = 8 << 20 // 8MB

// AgentReviewGenerator 通过 HTTP 调用 Python 文案 agent 服务生成评价，
// 实现 ReviewGenerator 接口。只把达到入池等级（默认 B 及以上）的文案返回。
type AgentReviewGenerator struct {
	BaseURL       string
	MinGrade      string
	InternalToken string
	Client        *http.Client
}

func NewAgentReviewGenerator(baseURL, minGrade, internalToken string) *AgentReviewGenerator {
	if minGrade == "" {
		minGrade = "B"
	}
	return &AgentReviewGenerator{
		BaseURL:       strings.TrimRight(baseURL, "/"),
		MinGrade:      minGrade,
		InternalToken: internalToken,
		// 自评循环 + 批量，单次可能较慢，给足超时
		Client: &http.Client{Timeout: 180 * time.Second},
	}
}

type agentStore struct {
	StoreName    string `json:"store_name"`
	IndustryType string `json:"industry_type"`
	StoreIntro   string `json:"store_intro"`
	BrandTone    string `json:"brand_tone"`
	Address      string `json:"address"`
}

type agentRequest struct {
	Store        agentStore     `json:"store"`
	Keywords     []string       `json:"keywords"`
	Platform     string         `json:"platform"`
	Count        int            `json:"count"`
	Satisfaction string         `json:"satisfaction"`
	Feedback     *agentFeedback `json:"feedback,omitempty"`
}

type agentFeedback struct {
	Accepted []string `json:"accepted,omitempty"`
	Rejected []string `json:"rejected,omitempty"`
}

type agentItem struct {
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Score   int      `json:"score"`
	Grade   string   `json:"grade"`
}

type agentResponse struct {
	Platform string      `json:"platform"`
	Produced int         `json:"produced"`
	Items    []agentItem `json:"items"`
}

var validPlatforms = map[string]bool{"dianping": true, "meituan": true, "xiaohongshu": true, "douyin": true}
var gradeRank = map[string]int{"D": 0, "C": 1, "B": 2, "A": 3, "S": 4}

func (g *AgentReviewGenerator) Generate(store model.Store, keywords []model.StoreKeyword, targetCount int) ([]model.ReviewItem, error) {
	return g.GenerateWithFeedback(store, keywords, ReviewGenerationFeedback{}, targetCount)
}

func (g *AgentReviewGenerator) GenerateWithFeedback(store model.Store, keywords []model.StoreKeyword, feedback ReviewGenerationFeedback, targetCount int) ([]model.ReviewItem, error) {
	if !strings.HasPrefix(g.BaseURL, "http://") && !strings.HasPrefix(g.BaseURL, "https://") {
		return nil, fmt.Errorf("非法的文案服务地址: %q", g.BaseURL)
	}

	platform := store.PrimaryPlatformStyle
	if !validPlatforms[platform] {
		platform = "dianping" // 兜底：未知风格走点评
	}

	kw := make([]string, 0, len(keywords))
	for _, k := range keywords {
		kw = append(kw, k.Keyword)
	}

	payload, err := json.Marshal(agentRequest{
		Store: agentStore{
			StoreName:    store.StoreName,
			IndustryType: store.IndustryType,
			StoreIntro:   store.StoreIntro,
			BrandTone:    store.BrandTone,
			Address:      store.Address,
		},
		Keywords:     kw,
		Platform:     platform,
		Count:        targetCount,
		Satisfaction: "比较满意",
		Feedback:     agentFeedbackFrom(feedback),
	})
	if err != nil {
		return nil, fmt.Errorf("构造文案请求失败: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, g.BaseURL+"/generate-reviews", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("构造文案 HTTP 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if g.InternalToken != "" {
		req.Header.Set("X-Agent-Internal-Token", g.InternalToken)
	}

	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用文案服务失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("文案服务返回状态 %d", resp.StatusCode)
	}

	var out agentResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxAgentRespBytes)).Decode(&out); err != nil {
		return nil, fmt.Errorf("解析文案服务响应失败: %w", err)
	}

	minRank := gradeRank[strings.ToUpper(g.MinGrade)]
	items := make([]model.ReviewItem, 0, len(out.Items))
	for _, it := range out.Items {
		if strings.TrimSpace(it.Content) == "" {
			continue
		}
		if gradeRank[strings.ToUpper(it.Grade)] < minRank {
			continue // 未达入池等级，丢弃（C/D 不入池）
		}
		items = append(items, model.ReviewItem{
			StoreID:       store.ID,
			PlatformStyle: platform,
			Content:       it.Content,
			Tags:          strings.Join(it.Tags, ","),
			SourceType:    "ai",
			Status:        model.ReviewStatusAvailable,
		})
	}
	return items, nil
}

func agentFeedbackFrom(feedback ReviewGenerationFeedback) *agentFeedback {
	if len(feedback.Accepted) == 0 && len(feedback.Rejected) == 0 {
		return nil
	}
	return &agentFeedback{
		Accepted: feedback.Accepted,
		Rejected: feedback.Rejected,
	}
}
