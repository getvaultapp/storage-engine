package api

import (
	"database/sql"
	"net/http"

	"github.com/getvaultapp/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/vault-storage-engine/pkg/config"
	"github.com/getvaultapp/vault-storage-engine/pkg/sharding"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	store  sharding.ShardStore
	cfg    *config.Config
	logger *zap.Logger
)

func CreateBucketHandler(c *gin.Context, db *sql.DB) {
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

func GetBucketHandler(c *gin.Context, db *sql.DB) {
	bucketID := c.Param("bucket_id")

	bucketData, err := bucket.GetBucket(db, bucketID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bucket not found"})
		return
	}

	c.JSON(http.StatusOK, bucketData)
}
