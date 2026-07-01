package middleware

import (
	"net/http"
	"strings"

	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/auth"
	"ppk/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StatusChecker func(userID uint, role string) bool

func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		c.Writer.Header().Add("Vary", "Origin")
		if _, ok := allowed[origin]; ok {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			if origin != "" {
				if _, ok := allowed[origin]; !ok {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
			}
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func AuthRequired(secret string, checker StatusChecker, roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Error(c, 401, "missing token")
			c.Abort()
			return
		}

		claims, err := auth.ParseToken(strings.TrimPrefix(header, "Bearer "), secret)
		if err != nil {
			response.Error(c, 401, "invalid token")
			c.Abort()
			return
		}

		if len(roles) > 0 {
			allowed := false
			for _, role := range roles {
				if claims.Role == role {
					allowed = true
					break
				}
			}
			if !allowed {
				response.Error(c, 403, "forbidden")
				c.Abort()
				return
			}
		}

		if checker != nil && !checker(claims.UserID, claims.Role) {
			response.Error(c, 401, "invalid token")
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func DBStatusChecker(db *gorm.DB) StatusChecker {
	return func(userID uint, role string) bool {
		if db == nil || userID == 0 {
			return false
		}

		switch role {
		case "admin":
			var user model.AdminUser
			err := db.Where("id = ? AND status = ?", userID, model.StatusEnabled).First(&user).Error
			return err == nil
		case "merchant":
			var user model.MerchantUser
			err := db.Where("id = ? AND status = ?", userID, model.StatusEnabled).First(&user).Error
			return err == nil
		default:
			return false
		}
	}
}

func StaticStatusChecker(enabled bool) StatusChecker {
	return func(uint, string) bool {
		return enabled
	}
}
