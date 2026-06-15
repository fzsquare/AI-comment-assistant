package response

import "github.com/gin-gonic/gin"

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": data})
}

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"code": status, "message": message})
}
