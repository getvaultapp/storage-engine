package api

import (
	"database/sql"
	"net/http"

	"github.com/getvault-mvp/vault-base/pkg/bucket"
	"github.com/getvault-mvp/vault-base/pkg/config"
	"github.com/getvault-mvp/vault-base/pkg/datastorage"
	"github.com/getvault-mvp/vault-base/pkg/sharding"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	store  sharding.ShardStore
	cfg    *config.Config
	logger *zap.Logger
)

func createBucketHandler(c *gin.Context, db *sql.DB) {
	var req struct {
		BucketID string `json:"bucket_id"`
		Owner    string `json:"owner"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := bucket.CreateBucket(db, req.BucketID, req.Owner)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bucket"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Bucket created", "bucket_id": req.BucketID})
}

func getBucketHandler(c *gin.Context, db *sql.DB) {
	bucketID := c.Param("bucket_id")

	bucketData, err := bucket.GetBucket(db, bucketID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bucket not found"})
		return
	}

	c.JSON(http.StatusOK, bucketData)
}

func uploadObjectHandler(c *gin.Context, db *sql.DB) {
	var req struct {
		ObjectID string `json:"object_id"`
		Data     string `json:"data"` // Base64 encoded file data
	}

	bucketID := c.Param("bucket_id")

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Decode Base64 data
	data := []byte(req.Data)

	// Store data using Vault's storage system
	versionID, _, _, err := datastorage.StoreData(db, data, bucketID, req.ObjectID, "uploaded_file", store, cfg, []string{}, logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store object"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Object stored", "version_id": versionID})
}

func retrieveObjectHandler(c *gin.Context, db *sql.DB) {
	objectID := c.Param("object_id")

	// Fetch latest version
	objectData, err := bucket.GetObjectMetadata(db, objectID, "latest")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Object not found"})
		return
	}

	c.JSON(http.StatusOK, objectData)
}

func listVersionsHandler(c *gin.Context, db *sql.DB) {
	objectID := c.Param("object_id")

	versions, err := bucket.ListObjectVersions(db, objectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list versions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

func retrieveVersionHandler(c *gin.Context, db *sql.DB) {
	objectID := c.Param("object_id")
	versionID := c.Param("version_id")

	objectData, err := bucket.GetObjectMetadata(db, objectID, versionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	c.JSON(http.StatusOK, objectData)
}

func setPermissionsHandler(c *gin.Context, db *sql.DB) {
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
