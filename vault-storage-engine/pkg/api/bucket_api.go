package api

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/datastorage"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/sharding"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Bucket Handlers
func ListBucketsHandler(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)

	buckets, err := bucket.ListAllBuckets(db)
	if err != nil {
		log.Printf("Error listing buckets: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list buckets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"buckets": buckets})
}

func CreateBucketHandler(c *gin.Context) {
	var createRequest struct {
		BucketID string `json:"bucket_id" binding:"required"`
		OwnerID  string `json:"owner_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&createRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := c.MustGet("db").(*sql.DB)

	err := bucket.CreateBucket(db, createRequest.BucketID, createRequest.OwnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bucket"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Bucket created successfully", "bucket_id": createRequest.BucketID})
}

func GetBucketHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")

	db := c.MustGet("db").(*sql.DB)
	bucket, err := bucket.GetBucket(db, bucketID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bucket not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bucket": bucket})
}

func DeleteBucketHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")

	db := c.MustGet("db").(*sql.DB)
	cfg := c.MustGet("config").(*config.Config)
	logger := c.MustGet("logger").(*zap.Logger)

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)

	err := datastorage.DeleteBucket(db, bucketID, store, logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bucket"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bucket deleted successfully", "bucket_id": bucketID})
}
