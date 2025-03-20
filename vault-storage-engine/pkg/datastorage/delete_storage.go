package datastorage

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/getvaultapp/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/vault-storage-engine/pkg/sharding"
	"go.uber.org/zap"
)

// Delete a bucket
func DeleteBucket(db *sql.DB, bucketID string, store sharding.ShardStore, logger *zap.Logger) error {
	objects, err := bucket.GetObjectsInBucket(db, bucketID)
	if err != nil {
		return fmt.Errorf("failed to retrieve objects from bucket: %w", err)
	}

	for _, objectID := range objects {
		versionID, err := bucket.GetRootVersion(db, objectID)
		if err != nil {
			return fmt.Errorf("failed to get object version, %w", err)
		}
		err = DeleteObject(db, bucketID, objectID, versionID, store, logger)
		if err != nil {
			logger.Warn("failed to delete object", zap.String("object_id", objectID), zap.String("version_id", versionID), zap.Error(err))
		}
	}

	err = bucket.DeleteBucket(db, bucketID)
	if err != nil {
		return fmt.Errorf("failed to delete bucket from database, %w", err)
	}
	return nil
}

// Deelete an object
func DeleteObject(db *sql.DB, bucketID, objectID, versionID string, store sharding.ShardStore, logger *zap.Logger) error {
	metadata, err := bucket.GetObjectMetadata(db, objectID, versionID)
	if err != nil {
		return fmt.Errorf("failed to retieve metadata file, %w", err)
	}

	for shardKey, location := range metadata.ShardLocations {
		shardIdxStr := strings.TrimPrefix(shardKey, "shard_")
		shardIdx, err := strconv.Atoi(shardIdxStr)
		if err != nil {
			logger.Warn("invalid shard index", zap.String("shardKey", shardKey), zap.Error(err))
			continue
		}
		delShardErr := store.DeleteShard(objectID, shardIdx, location)
		if delShardErr != nil {
			logger.Warn("failed to delete shards", zap.String("shard", shardKey), zap.String("location", location), zap.Error(err))
		}
	}

	err = bucket.DeleteObject(db, bucketID, objectID)
	if err != nil {
		return fmt.Errorf("failed to delete object from database, %w", err)
	}

	return nil
}
