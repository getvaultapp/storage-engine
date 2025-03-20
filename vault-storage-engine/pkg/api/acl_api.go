package api

import (
	"database/sql"
	"net/http"

	"github.com/getvaultapp/vault-storage-engine/pkg/acl"
	"github.com/gin-gonic/gin"
)

func GrantAccessHandler(c *gin.Context, db *sql.DB) {
	var req struct {
		UserID     string `json:"user_id"`
		Permission string `json:"permission"` // "read" or "write"
	}

	resourceID := c.Param("bucket_id") // Default to bucket, override for objects

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := acl.AddPermission(db, resourceID, "bucket", req.UserID, req.Permission)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grant access"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Access granted"})
}
