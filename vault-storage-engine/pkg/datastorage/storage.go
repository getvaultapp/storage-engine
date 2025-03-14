package datastorage

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/getvault-mvp/vault-base/pkg/bucket"
	"github.com/getvault-mvp/vault-base/pkg/config"
	"github.com/getvault-mvp/vault-base/pkg/encryption"
	"github.com/getvault-mvp/vault-base/pkg/erasurecoding"
	"github.com/getvault-mvp/vault-base/pkg/proofofinclusion"
	"github.com/getvault-mvp/vault-base/pkg/sharding"
	"go.uber.org/zap"
)

// StoreData stores an object inside a bucket
func StoreData(db *sql.DB, data []byte, bucketID, objectID, filePath string, store sharding.ShardStore, cfg *config.Config, locations []string, logger *zap.Logger) (string, error) {
	versionID := fmt.Sprintf("v%d", time.Now().Unix()) // Generate unique version ID

	// Encrypt data
	key := cfg.EncryptionKey
	cipherText, err := encryption.Encrypt(data, key)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %w", err)
	}

	// Erasure code the encrypted data
	shards, err := erasurecoding.Encode(cipherText)
	if err != nil {
		return "", fmt.Errorf("erasure coding failed: %w", err)
	}

	// Generate Merkle proofs
	tree, err := proofofinclusion.BuildMerkleTree(shards)
	if err != nil {
		return "", fmt.Errorf("failed to build Merkle tree: %w", err)
	}

	// Store shards
	shardLocations := make(map[string]string)
	for idx, shard := range shards {
		// Add debugging statements
		fmt.Printf("Storing shard %d, shard length: %d\n", idx, len(shard))
		if idx >= len(locations) {
			return "", fmt.Errorf("index out of range: idx=%d, locations length=%d", idx, len(locations))
		}
		location := locations[idx] // Use configured storage locations
		err := store.StoreShard(objectID, idx, shard, location)
		if err != nil {
			return "", fmt.Errorf("failed to store shard %d: %w", idx, err)
		}
		shardLocations[fmt.Sprintf("shard_%d", idx)] = location
	}

	// Generate proof hashes
	var proofs []string
	for _, shard := range shards {
		proof, err := proofofinclusion.GetProof(tree, shard)
		if err != nil {
			return "", fmt.Errorf("failed to get proof: %w", err)
		}
		proofs = append(proofs, proof)
	}

	// Save object metadata in SQLite
	metadata := bucket.VersionMetadata{
		ShardLocations: shardLocations,
		Proofs:         proofs,
	}
	err = bucket.AddVersion(db, bucketID, objectID, versionID, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to add version to database: %w", err)
	}

	// Ensure object exists in the database
	err = bucket.AddObject(db, bucketID, objectID)
	if err != nil {
		return "", fmt.Errorf("failed to register object in bucket: %w", err)
	}

	fmt.Printf("Stored object %s (version %s) in bucket %s\n", objectID, versionID, bucketID)
	return versionID, nil
}

// RetrieveData fetches an object from a bucket and reconstructs it
func RetrieveData(db *sql.DB, bucketID, objectID, versionID string, store sharding.ShardStore, cfg *config.Config, logger *zap.Logger) ([]byte, error) {
	// Fetch metadata
	metadata, err := bucket.GetObjectMetadata(db, objectID, versionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve metadata: %w", err)
	}

	// Retrieve shards
	totalShards := erasurecoding.DataShards + erasurecoding.ParityShards
	shards := make([][]byte, totalShards)
	missing := 0

	for shardKey, location := range metadata.ShardLocations {
		shardIdxStr := strings.TrimPrefix(shardKey, "shard_")
		shardIdx, err := strconv.Atoi(shardIdxStr)
		if err != nil {
			logger.Warn("Invalid shard index", zap.String("shardKey", shardKey), zap.Error(err))
			missing++
			continue
		}
		shard, err := store.RetrieveShard(objectID, shardIdx, location)
		if err != nil {
			logger.Warn("Shard retrieval failed", zap.String("shard", shardKey), zap.String("location", location))
			missing++
		} else {
			shards[shardIdx] = shard
		}
	}

	// Check if we have enough shards to reconstruct
	if missing > erasurecoding.ParityShards {
		return nil, fmt.Errorf("insufficient shards for reconstruction")
	}

	// Reconstruct file
	cipherText, err := erasurecoding.Decode(shards)
	if err != nil {
		return nil, fmt.Errorf("erasure decoding failed: %w", err)
	}

	// Decrypt file
	key, err := bucket.GetEncryptionKey(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}
	plainText, err := encryption.Decrypt(cipherText, key)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plainText, nil
}
