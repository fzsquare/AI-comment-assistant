package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ppk/backend/internal/pkg/auth"

	"github.com/gin-gonic/gin"
)

func TestAuthRequiredRejectsDisabledAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const secret = "test-secret-with-enough-length-for-hmac"
	token, err := auth.GenerateToken(42, "merchant", secret)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	r := gin.New()
	r.GET("/protected", AuthRequired(secret, StaticStatusChecker(false), "merchant"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
