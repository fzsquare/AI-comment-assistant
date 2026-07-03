package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ppk/backend/internal/model"
)

func TestAgentReviewGeneratorSendsInternalToken(t *testing.T) {
	const token = "agent-token-with-enough-entropy"
	var gotToken string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Agent-Internal-Token")
		_ = json.NewEncoder(w).Encode(agentResponse{
			Platform: "dianping",
			Produced: 1,
			Items:    []agentItem{{Content: "味道很好，服务也很周到。", Grade: "B"}},
		})
	}))
	defer srv.Close()

	generator := NewAgentReviewGenerator(srv.URL, "B", token)
	_, err := generator.Generate(model.Store{PrimaryPlatformStyle: "dianping"}, nil, 1)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if gotToken != token {
		t.Fatalf("got token header %q, want %q", gotToken, token)
	}
}

func TestUnavailableReviewGeneratorNeverReturnsMockReviews(t *testing.T) {
	generator := NewUnavailableReviewGenerator("AGENT_SERVICE_URL is required")

	items, err := generator.Generate(model.Store{StoreName: "真实门店", PrimaryPlatformStyle: "dianping"}, nil, 10)
	if err == nil {
		t.Fatal("expected unavailable generator to return an error")
	}
	if len(items) != 0 {
		t.Fatalf("got %d generated items, want none", len(items))
	}
	if !strings.Contains(err.Error(), "AGENT_SERVICE_URL") {
		t.Fatalf("got error %q, want it to explain missing AGENT_SERVICE_URL", err.Error())
	}
}

func TestAgentReviewGeneratorPreservesMeituanPlatform(t *testing.T) {
	var gotPayload agentRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("Decode request failed: %v", err)
		}
		_ = json.NewEncoder(w).Encode(agentResponse{
			Platform: "meituan",
			Produced: 1,
			Items:    []agentItem{{Content: "套餐分量足，服务也很自然。", Grade: "B"}},
		})
	}))
	defer srv.Close()

	generator := NewAgentReviewGenerator(srv.URL, "B", "agent-token")
	items, err := generator.Generate(model.Store{PrimaryPlatformStyle: "meituan"}, nil, 1)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if gotPayload.Platform != "meituan" {
		t.Fatalf("got platform %q, want meituan", gotPayload.Platform)
	}
	if len(items) != 1 || items[0].PlatformStyle != "meituan" {
		t.Fatalf("got items %#v, want one meituan item", items)
	}
}

func TestAgentReviewGeneratorSendsFeedbackExamples(t *testing.T) {
	var gotPayload agentRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("Decode request failed: %v", err)
		}
		_ = json.NewEncoder(w).Encode(agentResponse{
			Platform: "dianping",
			Produced: 1,
			Items:    []agentItem{{Content: "服务挺自然，整体体验不错。", Grade: "B"}},
		})
	}))
	defer srv.Close()

	generator := NewAgentReviewGenerator(srv.URL, "B", "agent-token")
	_, err := generator.GenerateWithFeedback(
		model.Store{PrimaryPlatformStyle: "dianping"},
		nil,
		ReviewGenerationFeedback{
			Accepted: []string{"接受样本：蟹肉饱满，服务员会主动换盘。"},
			Rejected: []string{"不喜欢样本：太像广告，夸得太满。"},
		},
		1,
	)
	if err != nil {
		t.Fatalf("GenerateWithFeedback returned error: %v", err)
	}
	if gotPayload.Feedback == nil {
		t.Fatal("expected feedback payload")
	}
	if len(gotPayload.Feedback.Accepted) != 1 || gotPayload.Feedback.Accepted[0] != "接受样本：蟹肉饱满，服务员会主动换盘。" {
		t.Fatalf("accepted feedback got %#v", gotPayload.Feedback.Accepted)
	}
	if len(gotPayload.Feedback.Rejected) != 1 || gotPayload.Feedback.Rejected[0] != "不喜欢样本：太像广告，夸得太满。" {
		t.Fatalf("rejected feedback got %#v", gotPayload.Feedback.Rejected)
	}
}

func TestAgentReviewGeneratorSendsGenerationPreferences(t *testing.T) {
	var gotPayload agentRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("Decode request failed: %v", err)
		}
		_ = json.NewEncoder(w).Encode(agentResponse{
			Platform: "dianping",
			Produced: 1,
			Items:    []agentItem{{Content: "蟹很入味，服务也自然。", Grade: "B"}},
		})
	}))
	defer srv.Close()

	generator := NewAgentReviewGenerator(srv.URL, "B", "agent-token")
	_, err := generator.GenerateWithContext(
		model.Store{PrimaryPlatformStyle: "dianping"},
		nil,
		ReviewGenerationContext{
			Feedback: ReviewGenerationFeedback{
				Accepted: []string{"接受样本：服务员会主动换盘。"},
			},
			Preferences: GenerationPreferences{
				FocusKeywords:       []string{"香辣蟹", "服务热情"},
				StyleCodes:          []string{"natural", "detail_rich"},
				DiversityDimensions: []string{"customer_identity", "content_angle"},
				ReferenceReviews:    []string{"蟹很入味，服务员会主动帮忙换盘。"},
				LengthVariance:      "wide",
			},
		},
		1,
	)
	if err != nil {
		t.Fatalf("GenerateWithContext returned error: %v", err)
	}
	if gotPayload.GenerationPreferences == nil {
		t.Fatal("expected generation_preferences payload")
	}
	if got := gotPayload.GenerationPreferences.FocusKeywords; len(got) != 2 || got[0] != "香辣蟹" || got[1] != "服务热情" {
		t.Fatalf("focus keywords got %#v", got)
	}
	if got := gotPayload.GenerationPreferences.StyleCodes; len(got) != 2 || got[0] != "natural" || got[1] != "detail_rich" {
		t.Fatalf("style codes got %#v", got)
	}
	if got := gotPayload.GenerationPreferences.DiversityDimensions; len(got) != 2 || got[0] != "customer_identity" || got[1] != "content_angle" {
		t.Fatalf("diversity dimensions got %#v", got)
	}
	if gotPayload.GenerationPreferences.LengthVariance != "wide" {
		t.Fatalf("length variance got %q, want wide", gotPayload.GenerationPreferences.LengthVariance)
	}
	if gotPayload.Feedback == nil || len(gotPayload.Feedback.Accepted) != 1 {
		t.Fatalf("feedback should still be sent with preferences, got %#v", gotPayload.Feedback)
	}
}

func TestAgentReviewGeneratorIncludesTaskIDHeader(t *testing.T) {
	var gotTaskID string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTaskID = r.Header.Get("X-Generation-Task-ID")
		_ = json.NewEncoder(w).Encode(agentResponse{
			Platform: "dianping",
			Produced: 1,
			Items:    []agentItem{{Content: "服务挺自然，整体体验不错。", Grade: "B"}},
		})
	}))
	defer srv.Close()

	generator := NewAgentReviewGenerator(srv.URL, "B", "agent-token")
	_, err := generator.GenerateWithContext(
		model.Store{PrimaryPlatformStyle: "dianping"},
		nil,
		ReviewGenerationContext{TaskID: 42},
		1,
	)
	if err != nil {
		t.Fatalf("GenerateWithContext returned error: %v", err)
	}
	if gotTaskID != "42" {
		t.Fatalf("task id header got %q, want 42", gotTaskID)
	}
}

func TestAgentReviewGeneratorIncludesErrorBodyForNonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"detail":"LLM_API_KEY is required"}`))
	}))
	defer srv.Close()

	generator := NewAgentReviewGenerator(srv.URL, "B", "agent-token")
	_, err := generator.Generate(model.Store{PrimaryPlatformStyle: "dianping"}, nil, 1)
	if err == nil {
		t.Fatal("expected non-OK response to return an error")
	}
	if !strings.Contains(err.Error(), "503") || !strings.Contains(err.Error(), "LLM_API_KEY is required") {
		t.Fatalf("error got %q, want status and response body", err.Error())
	}
}
