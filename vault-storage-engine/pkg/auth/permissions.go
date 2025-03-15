package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PermissionMiddleware checks if the user has the required role to access a resource
func PermissionMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			c.Abort()
			return
		}
		c.Next()
	}
}
