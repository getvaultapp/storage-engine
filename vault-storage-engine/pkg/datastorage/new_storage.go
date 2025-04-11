package datastorage

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/compression"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/encryption"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/erasurecoding"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/proofofinclusion"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/sharding"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type NewStorage interface {
	NewStoreData(db *sql.DB, data []byte, bucketID, objectID, filePath string, store sharding.ShardStore, cfg *config.Config, locations []string, logger *zap.Logger) (string, map[string]string, []string, error)
	NewRetrieveData(db *sql.DB, bucketID, objectID, versionID string, store sharding.ShardStore, cfg *config.Config, logger *zap.Logger) ([]byte, string, error)
	NewStoreDataWithVersion(db *sql.DB, data []byte, bucketID, objectID, versionID, filePath string, store sharding.ShardStore, cfg *config.Config, locations []string, logger *zap.Logger) (string, map[string]string, []string, error)
}

// LookupStorageNodes uses the discovery service to find storage nodes for a given key
func LookupStorageNodes(key string, logger *zap.Logger) ([]string, error) {
	// Query the discovery services' lookup endpoint
	lookupURL := fmt.Sprintf("https://localhost:8080/lookup?key=%s", key)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(lookupURL)
	if err != nil {
		logger.Error("Failed to lookup storage nodes", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	var nodes []struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		logger.Error("Failed to decode the storage node responses", zap.Error(err))
		return nil, err
	}

	var urls []string
	for _, n := range nodes {
		urls = append(urls, n.Address)
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("no storage nodes available")
	}

	return urls, nil
}

// NewStoreData stores an object by distributing it across storage nodes
// The function compresses, encrypts, shards, and distributes data across storage nodes
// It records metadata about shard locations and proofs in the database
func NewStoreData(db *sql.DB, data []byte, bucketID, objectID, filePath string, store sharding.ShardStore, cfg *config.Config, locations []string, logger *zap.Logger) (string, map[string]string, []string, error) {
	var bucketExists bool
	query := `SELECT EXISTS(SELECT 1 FROM buckets WHERE bucket_id = ?)`
	err := db.QueryRow(query, bucketID).Scan(&bucketExists)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to check if bucket exists; %w", err)
	}

	if !bucketExists {
		return "", nil, nil, fmt.Errorf("bucket %s does not exists", bucketID)
	}

	versionID := uuid.New().String()

	compressedData, err := compression.Compress(data)
	if err != nil {
		return "", nil, nil, fmt.Errorf("compression failed, %w", err)
	}

	key := cfg.EncryptionKey
	cipherText, err := encryption.Encrypt(compressedData, key)
	if err != nil {
		return "", nil, nil, fmt.Errorf("encryption failed: %w", err)
	}

	shards, err := erasurecoding.Encode(cipherText)
	if err != nil {
		return "", nil, nil, fmt.Errorf("erasure coding failed: %w", err)
	}

	tree, err := proofofinclusion.BuildMerkleTree(shards)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to build Merkle tree: %w", err)
	}

	// Let's find available storage nodes through the discovey service instead of hardcoded location
	storageNodes, err := LookupStorageNodes(objectID, logger)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to lookup storage nodes: %w", err)
	}

	// Check if we have enough storage nodes available
	if len(storageNodes) < len(shards) {
		return "", nil, nil, fmt.Errorf("not enough storage nodes availale: need %d, found %d", len(shards), len(storageNodes))
	}

	// Create an HTTP client for communicating with the storage nodes
	// In real time production we'll need to configure mTLS here
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	shardLocations := make(map[string]string)
	for idx, shard := range shards {
		nodeURL := storageNodes[idx%len(storageNodes)] // We'll use Round-Robin distrubuition here for now

		uploadURL := fmt.Sprintf("%s/shards/%s/%s/%d", nodeURL, objectID, versionID, idx)

		// Costruct the upload URL for the shard data
		req, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(shard))
		if err != nil {
			logger.Error("failed to create shard upload request",
				zap.Int("shard", idx),
				zap.String("url", uploadURL),
				zap.Error(err))
			return "", nil, nil, fmt.Errorf("failed to create a upload request for shard %d: %w", idx, err)
		}

		req.Header.Set("Content-Type", "application/octet-stream")

		// Send request to storage nodes
		resp, err := httpClient.Do(req)
		if err != nil {
			logger.Error("failed to upload shard",
				zap.Int("shard", idx),
				zap.String("node", nodeURL),
				zap.Error(err))
			return "", nil, nil, fmt.Errorf("storage node returned error status %d for shard %d", resp.StatusCode, idx)
		}

		resp.Body.Close()
		shardLocations[fmt.Sprintf("shard_%d", idx)] = nodeURL

		logger.Info("shard uploaded successfully",
			zap.Int("shard", idx),
			zap.String("node: ", nodeURL),
		)
	}

	var proofs []string
	for _, shard := range shards {
		proof, err := proofofinclusion.GetProof(tree, shard)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to get proof: %w", err)
		}
		proofs = append(proofs, proof)
	}

	// Save the object metadata in the database
	metadata := bucket.VersionMetadata{
		BucketID:       bucketID,
		ObjectID:       objectID,
		VersionID:      versionID,
		Filename:       filepath.Base(filePath),
		Filesize:       fmt.Sprintf("%d", len(data)), // We'll store the actual filesize here
		Format:         strings.TrimPrefix(filepath.Ext(filePath), "."),
		CreationDate:   time.Now().Format(time.RFC3339),
		ShardLocations: shardLocations,
		Proofs:         utils.ConvertSliceToMap(proofs),
	}

	root_version, _ := bucket.GetRootVersion(db, objectID)
	err = bucket.AddVersion(db, bucketID, objectID, versionID, root_version, metadata, cipherText)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to register object in bucket: %w", err)
	}

	logger.Info("Object stored successfully across storage nodes",
		zap.String("object_id", objectID),
		zap.String("version_id", versionID))

	return versionID, shardLocations, proofs, nil
}

// NewRetrieveData fetches an object from storage nodes and reconstructs it
// The function looks up metadata, retrieves shards from storage nodes,
// reconstructs the data using erasure coding, then decrypts and decompresses it
func NewRetrieveData(db *sql.DB, bucketID, objectID, versionID string, store sharding.ShardStore, cfg *config.Config, logger *zap.Logger) ([]byte, string, error) {
	// Fetch metadata from the requested object
	metadata, err := bucket.GetObjectMetadata(db, objectID, versionID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to retrieve metadata: %w", err)
	}

	// Create HTTP Client for retrieving shards
	// We'll need to configure mTLS here
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Prepare an array for shards
	totalShards := erasurecoding.DataShards + erasurecoding.ParityShards
	shards := make([][]byte, totalShards)
	missing := 0

	// Retrieve the shards from the storage nodes
	for shardKey, nodeURL := range metadata.ShardLocations {
		shardIdxStr := strings.TrimPrefix(shardKey, "shard_")
		shardIdx, err := strconv.Atoi(shardIdxStr)
		if err != nil {
			logger.Warn("Invalid shard index",
				zap.String("shardKey", shardKey),
				zap.Error(err))
			missing++
			continue
		}

		// Construct the URL for retrieving the shards
		downloadURL := fmt.Sprintf("%s/shards/%s/%s/%d", nodeURL, objectID, versionID, shardIdx)

		// Send GET request to storage nodes
		req, err := http.NewRequest("GET", downloadURL, nil)
		if err != nil {
			logger.Warn("failed to create shard download request",
				zap.Int("shard", shardIdx),
				zap.String("url", downloadURL),
				zap.Error(err))
			missing++
			continue
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			logger.Warn("failed to download shard",
				zap.Int("shard", shardIdx),
				zap.String("node", nodeURL),
				zap.Error(err))
			missing++
			continue
		}

		if resp.StatusCode != http.StatusOK {
			logger.Warn("Storage nodes returned error",
				zap.Int("shard", shardIdx),
				zap.String("node", nodeURL),
				zap.Int("status", resp.StatusCode))
			resp.Body.Close()
			missing++
			continue
		}

		shard, err := utils.ReadAllWithBuffer(resp.Body)
		resp.Body.Close()

		if err != nil {
			logger.Warn("Failed to read shard data",
				zap.Int("shard", shardIdx),
				zap.String("node", nodeURL),
				zap.Error(err))
			missing++
			continue
		}

		// Store shard in our array
		shards[shardIdx] = shard
		logger.Info("Shard retrieved successfully",
			zap.Int("shard", shardIdx),
			zap.String("node", nodeURL))
	}

	// Check if we have enough shards to reconstruct the data
	if missing > erasurecoding.ParityShards {
		return nil, "", fmt.Errorf("insufficient shards for reconstruction: missing %d shards", missing)
	}

	// Reconstruct the original encrypted data using erasure coding
	cipherText, err := erasurecoding.Decode(shards)
	if err != nil {
		return nil, "", fmt.Errorf("erasure decoding failed; %w", err)
	}

	// Get the encryption key and reconstruct the data
	key, err := bucket.GetEncryptionKey(cfg)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get the encryption key, %w", err)
	}

	data, err := encryption.Decrypt(cipherText, key)
	if err != nil {
		return nil, "", fmt.Errorf("decryption failed: %w", err)
	}

	// Decompress the decrypted data
	plainText, err := compression.Decompress(data)
	if err != nil {
		return nil, "", fmt.Errorf("decompression failed, %w", err)
	}

	// Let's fetch filename from our database
	var filename string
	err = db.QueryRow(`SELECT filename FROM objects WWHERE if = ?`, objectID).Scan(&filename)
	if err != nil {
		return nil, "", fmt.Errorf("failed to retrieve filename, %w", err)
	}

	logger.Info("Object retrieved and reconstructed successfully",
		zap.String("object_id", objectID),
		zap.String("version_id", versionID))

	return plainText, filename, nil
}

// StoreDataWithVersion stores data with a specified version ID
// It follows the same flow as StoreData but uses the provided version ID instead of generating a new one
func NewStoreDataWithVersion(db *sql.DB, data []byte, bucketID, objectID, versionID, filePath string, store sharding.ShardStore, cfg *config.Config, locations []string, logger *zap.Logger) (string, map[string]string, []string, error) {
	var bucketExists bool
	query := `SELECT EXISTS(SELECT 1 FROM buckets WHERE bucket_id = ?)`
	err := db.QueryRow(query, bucketID).Scan(&bucketExists)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to check if bucket exists; %w", err)
	}

	if !bucketExists {
		return "", nil, nil, fmt.Errorf("bucket %s does not exists", bucketID)
	}

	compressedData, err := compression.Compress(data)
	if err != nil {
		return "", nil, nil, fmt.Errorf("compression failed, %w", err)
	}

	key := cfg.EncryptionKey
	cipherText, err := encryption.Encrypt(compressedData, key)
	if err != nil {
		return "", nil, nil, fmt.Errorf("encryption failed: %w", err)
	}

	shards, err := erasurecoding.Encode(cipherText)
	if err != nil {
		return "", nil, nil, fmt.Errorf("erasure coding failed: %w", err)
	}

	tree, err := proofofinclusion.BuildMerkleTree(shards)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to build Merkle tree: %w", err)
	}

	// Let's find available storage nodes through the discovey service instead of hardcoded location
	storageNodes, err := LookupStorageNodes(objectID, logger)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to lookup storage nodes: %w", err)
	}

	// Check if we have enough storage nodes available
	if len(storageNodes) < len(shards) {
		return "", nil, nil, fmt.Errorf("not enough storage nodes availale: need %d, found %d", len(shards), len(storageNodes))
	}

	// Create an HTTP client for communicating with the storage nodes
	// In real time production we'll need to configure mTLS here
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	shardLocations := make(map[string]string)
	for idx, shard := range shards {
		nodeURL := storageNodes[idx%len(storageNodes)] // We'll use Round-Robin distrubuition here for now

		uploadURL := fmt.Sprintf("%s/shards/%s/%s/%d", nodeURL, objectID, versionID, idx)

		// Costruct the upload URL for the shard data
		req, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(shard))
		if err != nil {
			logger.Error("failed to create shard upload request",
				zap.Int("shard", idx),
				zap.String("url", uploadURL),
				zap.Error(err))
			return "", nil, nil, fmt.Errorf("failed to create a upload request for shard %d: %w", idx, err)
		}

		req.Header.Set("Content-Type", "application/octet-stream")

		// Send request to storage nodes
		resp, err := httpClient.Do(req)
		if err != nil {
			logger.Error("failed to upload shard",
				zap.Int("shard", idx),
				zap.String("node", nodeURL),
				zap.Error(err))
			return "", nil, nil, fmt.Errorf("storage node returned error status %d for shard %d", resp.StatusCode, idx)
		}

		resp.Body.Close()
		shardLocations[fmt.Sprintf("shard_%d", idx)] = nodeURL

		logger.Info("shard uploaded successfully",
			zap.Int("shard", idx),
			zap.String("node: ", nodeURL),
		)
	}

	var proofs []string
	for _, shard := range shards {
		proof, err := proofofinclusion.GetProof(tree, shard)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to get proof: %w", err)
		}
		proofs = append(proofs, proof)
	}

	// Save the object metadata in the database
	metadata := bucket.VersionMetadata{
		BucketID:       bucketID,
		ObjectID:       objectID,
		VersionID:      versionID,
		Filename:       filepath.Base(filePath),
		Filesize:       fmt.Sprintf("%d", len(data)), // We'll store the actual filesize here
		Format:         strings.TrimPrefix(filepath.Ext(filePath), "."),
		CreationDate:   time.Now().Format(time.RFC3339),
		ShardLocations: shardLocations,
		Proofs:         utils.ConvertSliceToMap(proofs),
	}

	root_version, _ := bucket.GetRootVersion(db, objectID)
	err = bucket.AddVersion(db, bucketID, objectID, versionID, root_version, metadata, cipherText)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to register object in bucket: %w", err)
	}

	logger.Info("Object stored successfully across storage nodes",
		zap.String("object_id", objectID),
		zap.String("version_id", versionID))

	return versionID, shardLocations, proofs, nil
}
