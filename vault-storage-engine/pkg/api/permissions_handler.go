package api

import (
	"database/sql"
	"net/http"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/gin-gonic/gin"
)

func SetBucketPermissionsHandler(c *gin.Context, db *sql.DB) {
	bucketID := c.Param("bucket_id")

	var req struct {
		Read  []string `json:"read"`
		Write []string `json:"write"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := bucket.SetBucketPermissions(db, bucketID, req.Read, req.Write)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set permissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permissions updated"})
}
