package middleware

import (
	"net/http"
	"strings"

	"billing-app/internal/utils"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			return
		}

		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		if len(allowedRoles) > 0 {
			roleAllowed := false
			for _, role := range allowedRoles {
				if role == claims.Role {
					roleAllowed = true
					break
				}
			}
			if !roleAllowed {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied"})
				return
			}
		}

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
