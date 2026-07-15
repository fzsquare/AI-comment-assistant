package router

import (
	"strings"
	"testing"

	"ppk/backend/internal/config"
	"ppk/backend/internal/model"
)

func TestBuildReviewPoolServiceDoesNotUseMockWhenAgentMissing(t *testing.T) {
	pool := buildReviewPoolService(config.Config{AgentServiceURL: "", JWTSecret: "router-session-secret-with-32-bytes"}, nil)
	if pool.SessionSecret == "" {
		t.Fatal("landing sessions must use the configured secret")
	}

	items, err := pool.Generator.Generate(model.Store{StoreName: "真实门店", PrimaryPlatformStyle: "dianping"}, nil, 3)
	if err == nil {
		t.Fatal("expected generator error when AGENT_SERVICE_URL is missing")
	}
	if len(items) != 0 {
		t.Fatalf("got %d items, want none", len(items))
	}
	if !strings.Contains(err.Error(), "AGENT_SERVICE_URL") {
		t.Fatalf("got error %q, want missing AGENT_SERVICE_URL", err.Error())
	}
}
