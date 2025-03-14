package api

/* import (
	"database/sql"
	//"fmt"
	"github.com/gin-gonic/gin"
	//"github.com/getvault-mvp/vault-base/pkg/bucket"
	"io/ioutil"
	"net/http"

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
	versionID, err := datastorage.StoreData(db, data, bucketID, objectID, "uploaded_file", store, cfg, locations, logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Store failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"version_id": versionID, "bucket_id": bucketID, "object_id": objectID})
}
*/
