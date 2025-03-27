package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/datastorage"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/sharding"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Object Handlers
func ListObjectsHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")

	db := c.MustGet("db").(*sql.DB)

	objects, err := bucket.ListObjects(db, bucketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list objects"})
		return
	}

	for _, object := range objects {
		c.JSON(http.StatusOK, gin.H{"bucket_id": bucketID, "objects": object})
	}
	//c.JSON(http.StatusOK, gin.H{"bucket_id": bucketID, "objects": objects})
}

// Store a file and upload it into a bucket along with it's own versionID
func UploadObjectHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file"})
		return
	}

	db := c.MustGet("db").(*sql.DB)
	cfg := c.MustGet("config").(*config.Config)
	logger := c.MustGet("logger").(*zap.Logger)

	// Save the uploaded file to a temporary location
	tempFilePath := filepath.Join("/tmp", file.Filename)
	if err := c.SaveUploadedFile(file, tempFilePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	data, err := os.ReadFile(tempFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)
	objectID := uuid.New().String()

	_, shardLocations, proofs, err := datastorage.StoreData(db, data, bucketID, objectID, file.Filename, store, cfg, cfg.ShardLocations, logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store object"})
		return
	}

	proofsMap := utils.ConvertSliceToMap(proofs)

	c.JSON(http.StatusCreated, gin.H{
		"message":         "Object uploaded successfully",
		"bucket_id":       bucketID,
		"object_id":       objectID,
		"object_name":     file.Filename,
		"shard_locations": shardLocations,
		"proofs":          proofsMap,
	})
}

// Download an object, this should retrieve the latest version of an object
func GetObjectHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")
	objectID := c.Param("objectID")

	// How do we get the latest version of an object properly

	db := c.MustGet("db").(*sql.DB)
	cfg := c.MustGet("config").(*config.Config)
	logger := c.MustGet("logger").(*zap.Logger)

	// This should get the latest_version of the object
	// Let's check to make sure it works correctly
	versionID := bucket.GetLatestVersion(db, objectID)
	fmt.Println("latest version: ", versionID)

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)
	data, filename, err := datastorage.RetrieveData(db, bucketID, objectID, versionID, store, cfg, logger)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Object not found"})
		return
	}

	tmpFile, err := os.CreateTemp("", "object-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temporary file"})
		return
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write data to temporary file"})
		return
	}

	c.FileAttachment(tmpFile.Name(), filename)
}

// Download a particular version of an object
func GetObjectByVersionHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")
	objectID := c.Param("objectID")
	versionID := c.Param("versionID")

	db := c.MustGet("db").(*sql.DB)
	cfg := c.MustGet("config").(*config.Config)
	logger := c.MustGet("logger").(*zap.Logger)

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)
	data, filename, err := datastorage.RetrieveData(db, bucketID, objectID, versionID, store, cfg, logger)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Object not found"})
		return
	}

	tmpFile, err := os.CreateTemp("", "object-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temporary file"})
		return
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write data to temporary file"})
		return
	}

	c.FileAttachment(tmpFile.Name(), filename)
}

// This stores a nw version of an object, it'll give it a new version
func UpdateObjectVersionHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")
	objectID := c.Param("objectID")

	var updateRequest struct {
		Filename string `json:"filename" binding:"required"`
	}

	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := c.MustGet("db").(*sql.DB)
	cfg := c.MustGet("config").(*config.Config)
	logger := c.MustGet("logger").(*zap.Logger)

	getfile, versionID, err := bucket.UpdateFileVersionIfItExists(db, updateRequest.Filename, bucketID, objectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check if file exists"})
		return
	}

	if updateRequest.Filename != getfile {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	data, err := os.ReadFile(updateRequest.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)
	locations := cfg.ShardLocations

	versionID, _, _, err = datastorage.StoreDataWithVersion(db, data, bucketID, objectID, versionID, filepath.Base(updateRequest.Filename), store, cfg, locations, logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store updated object"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Object updated successfully", "bucket_id": bucketID, "object_id": objectID, "version _id": versionID})
}

// Deletes all versions of an object ... This isn't really working well yet
func DeleteObjectHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")
	objectID := c.Param("objectID")

	db := c.MustGet("db").(*sql.DB)
	cfg := c.MustGet("config").(*config.Config)
	logger := c.MustGet("logger").(*zap.Logger)

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)
	err := datastorage.DeleteObject(db, bucketID, objectID, store, logger)
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete object"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Object deleted successfully", "bucket_id": bucketID, "object_id": objectID})
}

// This deletes a particular version of an object
func DeleteObjectByVersionHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")
	objectID := c.Param("objectID")
	versionID := c.Param("versionID")

	db := c.MustGet("db").(*sql.DB)
	cfg := c.MustGet("config").(*config.Config)
	logger := c.MustGet("logger").(*zap.Logger)

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)
	err := datastorage.DeleteObjectByVersion(db, bucketID, objectID, versionID, store, logger)
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete object version"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Object version deleted successfully", "bucket_id": bucketID, "object_id": objectID, "version_id": versionID})
}
