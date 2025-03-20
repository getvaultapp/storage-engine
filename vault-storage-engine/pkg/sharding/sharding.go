package sharding

import (
	"fmt"
	"os"
	"path/filepath"
)

// ShardStore is an interface for storing shards
type ShardStore interface {
	StoreShard(objectID string, shardIdx int, shard []byte, location string) error
	RetrieveShard(objectID string, shardIdx int, location string) ([]byte, error)
	DeleteShard(objectID string, shardIdx int, location string) error
}

// LocalShardStore is a local implementation of ShardStore
type LocalShardStore struct {
	BasePath string
}

// NewLocalShardStore creates a new LocalShardStore
func NewLocalShardStore(basePath string) *LocalShardStore {
	return &LocalShardStore{BasePath: basePath}
}

// StoreShard stores a shard locally
func (store *LocalShardStore) StoreShard(objectID string, shardIdx int, shard []byte, location string) error {
	shardPath := filepath.Join(store.BasePath, location, fmt.Sprintf("%s_shard_%d", objectID, shardIdx))
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
func (store *LocalShardStore) RetrieveShard(objectID string, shardIdx int, location string) ([]byte, error) {
	shardPath := filepath.Join(store.BasePath, location, fmt.Sprintf("%s_shard_%d", objectID, shardIdx))
	shard, err := os.ReadFile(shardPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read shard from file: %w", err)
	}
	return shard, nil
}

// Delete shards from their locations
func (store *LocalShardStore) DeleteShard(objectID string, shardIdx int, location string) error {
	if location == "" {
		return fmt.Errorf("invalid storage location")
	}
	shardPath := filepath.Join(store.BasePath, location, fmt.Sprintf("%s_shard_%d", objectID, shardIdx))

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
