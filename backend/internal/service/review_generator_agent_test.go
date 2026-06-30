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
