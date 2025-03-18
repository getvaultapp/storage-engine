package api

import (
	"database/sql"
	//"fmt"
	"github.com/gin-gonic/gin"
	//"github.com/getvault-mvp/vault-base/pkg/bucket"
	"io/ioutil"
	"net/http"

	"github.com/getvault-mvp/vault-base/pkg/bucket"
	"github.com/getvault-mvp/vault-base/pkg/config"
	"github.com/getvault-mvp/vault-base/pkg/datastorage"
	"github.com/getvault-mvp/vault-base/pkg/sharding"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func StoreObjectHandler(c *gin.Context, db *sql.DB, cfg *config.Config, logger *zap.Logger) {
	bucketID := c.Param("bucket_id")
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file"})
		return
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)
	// Initialize locations with actual paths
	locations := []string{
		"/mnt/disk1/shards",
		"/mnt/disk2/shards",
		"/mnt/disk3/shards",
		"/mnt/disk4/shards",
		"/mnt/disk5/shards",
		"/mnt/disk6/shards",
		"/mnt/disk7/shards",
		"/mnt/disk8/shards",
	}
	objectID := uuid.New().String() // Generate a unique object ID
	versionID, _, _, err := datastorage.StoreData(db, data, bucketID, objectID, "uploaded_file", store, cfg, locations, logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Store failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"version_id": versionID, "bucket_id": bucketID, "object_id": objectID})
}

func RetrieveObjectHandler(c *gin.Context, db *sql.DB, cfg *config.Config, logger *zap.Logger) {
	bucketID := c.Param("bucket_id")
	objectID := c.Param("object_id")
	versionID := c.Param("version_id")

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)
	data, filename, err := datastorage.RetrieveData(db, bucketID, objectID, versionID, store, cfg, logger)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Object not found"})
		return
	}

	c.Data(http.StatusOK, "application/octet-stream", data)
	c.Header("Content-Disposition", "attachement; filename="+filename)
}

func UploadObjectHandler(c *gin.Context, db *sql.DB) {
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

func GetObjectMetadataHandler(c *gin.Context, db *sql.DB) {
	objectID := c.Param("object_id")

	// Fetch latest version
	objectData, err := bucket.GetObjectMetadata(db, objectID, "latest")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Object not found"})
		return
	}

	c.JSON(http.StatusOK, objectData)
}

func UpdateObjectVersionHandler(c *gin.Context, db *sql.DB) {}
