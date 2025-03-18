package api

import (
	"net/http"

	"github.com/getvault-mvp/vault-base/pkg/auth"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware(secretKey string) gin.HandlerFunc {
	return nil
}
func LoginHandler(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"` // "owner", "reader", "writer"
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Generate JWT token
	token, err := auth.GenerateJWT(req.UserID, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
