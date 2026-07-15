package service

import (
	"strings"
	"testing"
	"time"
)

func TestLandingSessionIsSignedForOneStoreAndExpires(t *testing.T) {
	pool := ReviewPoolService{SessionSecret: strings.Repeat("s", 32)}
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

	token, err := pool.IssueLandingSession("store-a", now)
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	if err := pool.VerifyLandingSession("store-a", token, now.Add(time.Hour)); err != nil {
		t.Fatalf("verify issued session: %v", err)
	}
	if err := pool.VerifyLandingSession("store-b", token, now.Add(time.Hour)); err == nil {
		t.Fatal("session must be bound to one store")
	}
	if err := pool.VerifyLandingSession("store-a", token+"forged", now.Add(time.Hour)); err == nil {
		t.Fatal("forged session must be rejected")
	}
	if err := pool.VerifyLandingSession("store-a", token, now.Add(25*time.Hour)); err == nil {
		t.Fatal("expired session must be rejected")
	}
	if err := pool.VerifyLandingSession("store-a", token, now.Add(24*time.Hour)); err == nil {
		t.Fatal("session must be invalid at its expiry instant")
	}
}

func TestLandingSessionsAreUniqueAndFitPersistedColumns(t *testing.T) {
	pool := ReviewPoolService{SessionSecret: strings.Repeat("s", 32)}
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	storeUUID := "123e4567-e89b-12d3-a456-426614174000"

	first, err := pool.IssueLandingSession(storeUUID, now)
	if err != nil {
		t.Fatalf("issue first session: %v", err)
	}
	second, err := pool.IssueLandingSession(storeUUID, now)
	if err != nil {
		t.Fatalf("issue second session: %v", err)
	}
	if first == second {
		t.Fatal("different visitors issued in the same second must have unique sessions")
	}
	if len(first) > 128 || len(second) > 128 {
		t.Fatalf("session tokens must fit VARCHAR(128), lengths=%d,%d", len(first), len(second))
	}
}
