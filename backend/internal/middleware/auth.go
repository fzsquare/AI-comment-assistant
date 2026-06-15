package middleware

import (
	"strings"

	"ppk/backend/internal/pkg/auth"
	"ppk/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func AuthRequired(secret string, roles ...string) gin.HandlerFunc {
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

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
