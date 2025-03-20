package api

import (
	"database/sql"
	"net/http"

	"github.com/getvaultapp/vault-storage-engine/pkg/bucket"
	"github.com/gin-gonic/gin"
)

func ListVersionsHandler(c *gin.Context, db *sql.DB) {
	objectID := c.Param("object_id")

	versions, err := bucket.ListObjectVersions(db, objectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list versions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

func RetrieveVersionHandler(c *gin.Context, db *sql.DB) {
	objectID := c.Param("object_id")
	versionID := c.Param("version_id")

	objectData, err := bucket.GetObjectMetadata(db, objectID, versionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	c.JSON(http.StatusOK, objectData)
}
