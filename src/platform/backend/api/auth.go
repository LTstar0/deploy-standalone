package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// DeployToken is the globally configured auth token
var DeployToken string

// AuthMiddleware intercepts requests and validates the publisher token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// If DeployToken is not set, skip authentication (open mode)
		if DeployToken == "" {
			c.Next()
			return
		}

		var token string

		// 1. Check custom X-Deploy-Token header
		token = c.GetHeader("X-Deploy-Token")

		// 2. Check Authorization header
		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
					token = authHeader[7:]
				} else {
					token = authHeader
				}
			}
		}

		// 3. Check query string parameter (fallback, mostly for WebSockets)
		if token == "" {
			token = c.Query("token")
		}

		// Validate token
		if token != DeployToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "未授权，请输入正确的发布 Token",
			})
			return
		}

		c.Next()
	}
}
