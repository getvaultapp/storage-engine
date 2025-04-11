package sharding

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ShardStore is an interface for storing shards
type ShardStore interface {
	StoreShard(objectID, versionID string, shardIdx int, shard []byte, location string) error
	RetrieveShard(objectID, versionID string, shardIdx int, location string) ([]byte, error)
	DeleteShard(objectID string, shardIdx int, location string) error
	DeleteShardByVersion(objectID, versionID string, shardIdx int, location string) error
}

// LocalShardStore is a local implementation of ShardStore
type LocalShardStore struct {
	BasePath string
}

// TODO: UPDATE THIS FUNCTION PLS
func ShardExists(objectID, versionID string, shardIdx int, nodeID string) (bool, error) {
	return true, nil
}

// NewLocalShardStore creates a new LocalShardStore
func NewLocalShardStore(basePath string) *LocalShardStore {
	return &LocalShardStore{BasePath: basePath}
}

// StoreShard stores a shard locally
func (store *LocalShardStore) StoreShard(objectID, versionID string, shardIdx int, shard []byte, location string) error {
	// Record versions with each shard
	shardPath := filepath.Join(store.BasePath, location, fmt.Sprintf("%s-v(%s)_shard_%d", objectID, versionID, shardIdx))
	err := os.MkdirAll(filepath.Dir(shardPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory for shard: %w", err)
	}

	err = os.WriteFile(shardPath, shard, 0644)
	if err != nil {
		return fmt.Errorf("failed to write shard to file: %w", err)
	}
	return nil
}

// RetrieveShard retrieves a shard locally
func (store *LocalShardStore) RetrieveShard(objectID, versionID string, shardIdx int, location string) ([]byte, error) {
	// Record version with each shard
	shardPath := filepath.Join(store.BasePath, location, fmt.Sprintf("%s-v(%s)_shard_%d", objectID, versionID, shardIdx))
	shard, err := os.ReadFile(shardPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read shard from file: %w", err)
	}
	return shard, nil
}

// Only delete shards of a particular version_id
func (store *LocalShardStore) DeleteShardByVersion(objectID, versionID string, shardIdx int, location string) error {
	if location == "" {
		return fmt.Errorf("invalid storage location")
	}
	shardPath := filepath.Join(store.BasePath, location, fmt.Sprintf("%s-v(%s)_shard_%d", objectID, versionID, shardIdx))

	err := os.Remove(shardPath)
	if err != nil {
		// Let's check if the shard does not exists
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete shard file, %w", err)
	}
	return nil
}

// Delete all shards of the same object_id
func (store *LocalShardStore) DeleteShard(objectID string, shardIdx int, location string) error {
	if location == "" {
		return fmt.Errorf("invalid storage location")
	}

	shardDir := filepath.Join(store.BasePath, location)

	// Read all files in the directory
	files, err := os.ReadDir(shardDir)
	if err != nil {
		return fmt.Errorf("failed to read shard directory: %w", err)
	}

	// Iterate and delete matching shards
	for _, file := range files {
		if strings.HasPrefix(file.Name(), objectID+"-v(") {
			shardPath := filepath.Join(shardDir, file.Name())

			err := os.Remove(shardPath)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete shard file %s: %w", shardPath, err)
			}
		}
	}
	return nil
}
