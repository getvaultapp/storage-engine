package datastorage

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/getvault-mvp/vault-base/pkg/bucket"
	"github.com/getvault-mvp/vault-base/pkg/config"
	"github.com/getvault-mvp/vault-base/pkg/encryption"
	"github.com/getvault-mvp/vault-base/pkg/erasurecoding"
	"github.com/getvault-mvp/vault-base/pkg/proofofinclusion"
	"github.com/getvault-mvp/vault-base/pkg/sharding"
	"github.com/getvault-mvp/vault-base/pkg/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// StoreData stores an object inside a bucket
func StoreData(db *sql.DB, data []byte, bucketID, objectID, filePath string, store sharding.ShardStore, cfg *config.Config, locations []string, logger *zap.Logger) (string, map[string]string, []string, error) {
	// First check if the bucket exists
	var bucketExists bool

	// Check if the Bucket exists
	query := "SELECT EXISTS(SELECT 1 FROM buckets WHERE bucket_id = ?)"
	err := db.QueryRow(query, bucketID).Scan(&bucketExists)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to check if bucket exists, %w", err)
	}

	if !bucketExists {
		return "", nil, nil, fmt.Errorf("bucket %s does not exists", bucketID)
	}

	// Generate unique version ID
	versionID := uuid.New().String()

	// Encrypt data
	key := cfg.EncryptionKey
	cipherText, err := encryption.Encrypt(data, key)
	if err != nil {
		return "", nil, nil, fmt.Errorf("encryption failed: %w", err)
	}

	// Erasure code the encrypted data
	shards, err := erasurecoding.Encode(cipherText)
	if err != nil {
		return "", nil, nil, fmt.Errorf("erasure coding failed: %w", err)
	}

	// Generate Merkle proofs
	tree, err := proofofinclusion.BuildMerkleTree(shards)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to build Merkle tree: %w", err)
	}

	// Store shards
	shardLocations := make(map[string]string)
	for idx, shard := range shards {
		fmt.Printf("Storing shard %d, shard length: %d\n", idx, len(shard))
		if idx >= len(locations) {
			return "", nil, nil, fmt.Errorf("index out of range: idx=%d, locations length=%d", idx, len(locations))
		}
		location := locations[idx] // Use configured storage locations
		err := store.StoreShard(objectID, idx, shard, location)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to store shard %d: %w", idx, err)
		}
		shardLocations[fmt.Sprintf("shard_%d", idx)] = location
	}

	// Generate proof hashes
	var proofs []string
	for _, shard := range shards {
		proof, err := proofofinclusion.GetProof(tree, shard)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to get proof: %w", err)
		}
		proofs = append(proofs, proof)
	}

	// Save object metadata in SQLite
	metadata := bucket.VersionMetadata{
		BucketID:       bucketID,
		ObjectID:       objectID,
		VersionID:      versionID,
		Filename:       filepath.Base(filePath),
		Filesize:       "",
		Format:         strings.TrimPrefix(filepath.Ext(filePath), "."),
		CreationDate:   time.Now().Format(time.RFC3339),
		ShardLocations: shardLocations,
		Proofs:         utils.ConvertSliceToMap(proofs),
	}

	root_version, _ := bucket.GetRootVersion(db, objectID)
	err = bucket.AddVersion(db, bucketID, objectID, versionID, root_version, metadata, cipherText)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to add version to database: %w", err)
	}

	filename := filepath.Base(filePath)
	// Ensure object exists in the database
	err = bucket.AddObject(db, bucketID, objectID, filename)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to register object in bucket: %w", err)
	}

	fmt.Printf("Stored object %s (version %s) in bucket %s\n", objectID, versionID, bucketID)
	return versionID, shardLocations, proofs, nil
}

// RetrieveData fetches an object from a bucket and reconstructs it
func RetrieveData(db *sql.DB, bucketID, objectID, versionID string, store sharding.ShardStore, cfg *config.Config, logger *zap.Logger) ([]byte, string, error) {
	// Fetch metadata
	metadata, err := bucket.GetObjectMetadata(db, objectID, versionID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to retrieve metadata: %w", err)
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
		return nil, "", fmt.Errorf("insufficient shards for reconstruction")
	}

	// Reconstruct file
	cipherText, err := erasurecoding.Decode(shards)
	if err != nil {
		return nil, "", fmt.Errorf("erasure decoding failed: %w", err)
	}

	// Decrypt file
	key, err := bucket.GetEncryptionKey(cfg)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get encryption key: %w", err)
	}
	plainText, err := encryption.Decrypt(cipherText, key)
	if err != nil {
		return nil, "", fmt.Errorf("decryption failed: %w", err)
	}

	// Fetch filename from the database
	var filename string
	err = db.QueryRow(`SELECT filename FROM objects WHERE id = ?`, objectID).Scan(&filename)
	if err != nil {
		return nil, "", fmt.Errorf("failed to retrieve filename: %w", err)
	}

	return plainText, filename, nil
}
