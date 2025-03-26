package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/gin-gonic/gin"
)

func ListVersionsHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")
	objectID := c.Param("objectID")

	db := c.MustGet("db").(*sql.DB)

	versions, err := bucket.ListObjectVersions(db, objectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list versions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bucket": bucketID, "objectID": objectID, "versions": versions})
}

// Returns the metadata of an object version
func RetrieveVersionHandler(c *gin.Context) {
	objectID := c.Param("objectID")
	versionID := c.Param("versionID")

	db := c.MustGet("db").(*sql.DB)

	objectMetadata, err := bucket.GetObjectMetadata(db, objectID, versionID)
	fmt.Println(err)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bucket_id": objectMetadata.BucketID,
		"object_id":       objectMetadata.ObjectID,
		"version_id":      objectMetadata.VersionID,
		"filename":        objectMetadata.Filename,
		"filesize":        objectMetadata.Filesize,
		"format":          objectMetadata.CreationDate,
		"creation_date":   objectMetadata.CreationDate,
		"data":            objectMetadata.Data,
		"shard_locations": objectMetadata.ShardLocations,
		"proofs":          objectMetadata.Proofs})
}

func DownloadMetadata(c *gin.Context) {
	bucketID := c.Param("bucketID")
	objectID := c.Param("objectID")
	versionID := c.Param("versionID")

	db := c.MustGet("db").(*sql.DB)

	metadatafilename := fmt.Sprintf("%s-%s-%s.metadata.json", bucketID, objectID, versionID)

	err := bucket.ReadMetadataJson(db, bucketID, objectID, versionID, metadatafilename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve metadata for object"})
		return
	}

	tmpFile, err := os.CreateTemp("", "object-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temporary file"})
		return
	}
	defer tmpFile.Close()

	c.FileAttachment(tmpFile.Name(), metadatafilename)
}
