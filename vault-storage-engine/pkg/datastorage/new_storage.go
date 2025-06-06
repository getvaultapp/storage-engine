package datastorage

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

const (
	maxUploadRetries   = 3
	maxDownloadRetries = 3
	baseBackoff        = time.Second
)

type NewStorage interface {
	NewStoreData(db *sql.DB, data []byte, bucketID, objectID, filePath string, store sharding.ShardStore, cfg *config.Config, locations []string, logger *zap.Logger) (string, map[string]string, []string, error)
	NewRetrieveData(db *sql.DB, bucketID, objectID, versionID string, store sharding.ShardStore, cfg *config.Config, logger *zap.Logger) ([]byte, string, error)
	NewStoreDataWithVersion(db *sql.DB, data []byte, bucketID, objectID, versionID, filePath string, store sharding.ShardStore, cfg *config.Config, locations []string, logger *zap.Logger) (string, map[string]string, []string, error)
}

type ShardMetadata struct {
	BucketID       string
	ObjectID       string
	VersionID      string
	ShardLocations map[string]string
}

// LookupStorageNodes uses the discovery service to find storage nodes for a given key
func LookupStorageNodes(logger *zap.Logger) ([]string, error) {
	discoveryPort := os.Getenv("DISCOVERY_PORT")
	discoveryURL := fmt.Sprintf("http://localhost:%s", discoveryPort)
	if discoveryURL == "" {
		discoveryURL = "http://localhost:9000" // fallback
		log.Println("Disovery URL set to default")
	}
	lookupURL := fmt.Sprintf("%s/lookup", discoveryURL)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(lookupURL)

	if err != nil {
		logger.Error("Failed to contact discovery service", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read discovery response body", zap.Error(err))
		return nil, err
	}

	logger.Debug("Raw discovery response", zap.ByteString("body", body))

	var objectNodes []struct {
		Address string `json:"address"`
	}
	if err := json.Unmarshal(body, &objectNodes); err == nil {
		var urls []string
		for _, node := range objectNodes {
			urls = append(urls, node.Address)
		}
		if len(urls) == 0 {
			return nil, fmt.Errorf("discovery returned zero nodes")
		}

		return urls, nil
	}

	// Fallback: try decoding as array of raw strings
	var stringNodes []string
	if err := json.Unmarshal(body, &stringNodes); err == nil {
		if len(stringNodes) == 0 {
			return nil, fmt.Errorf("discovery returned zero node addresses")
		}
		return stringNodes, nil
	}

	// If both failed
	logger.Error("Failed to parse discovery response", zap.ByteString("body", body))
	return nil, fmt.Errorf("invalid format in discovery response")
}

// NewStoreData stores an object by distributing it across storage nodes
// The function compresses, encrypts, shards, and distributes data across storage nodes
// It records metadata about shard locations and proofs in the database
func NewStoreData(
	db *sql.DB,
	data []byte,
	bucketID, objectID, filePath string,
	store sharding.ShardStore,
	cfg *config.Config,
	locations []string,
	logger *zap.Logger) (string, map[string]string, []string, error) {

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
	storageNodes, err := LookupStorageNodes(logger)
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
		// This should get the URL of the nodes storing a particular shard
		nodeURL := storageNodes[idx%len(storageNodes)]
		fmt.Println(nodeURL)
		uploadURL := fmt.Sprintf("%s/shards/%s/%s/%d", nodeURL, objectID, versionID, idx)

		var resp *http.Response
		var err error
		for attempt := 1; attempt <= maxUploadRetries; attempt++ {
			req, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(shard))
			if err != nil {
				logger.Error("failed to create upload request",
					zap.Int("shard", idx), zap.Error(err))
				break
			}

			req.Header.Set("Content-Type", "application/octet-stream")
			resp, err = httpClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusCreated {
				resp.Body.Close()
				break
			}
			if resp != nil {
				resp.Body.Close()
			}

			logger.Warn("upload failed, retrying...",
				zap.Int("shard", idx),
				zap.String("node", nodeURL),
				zap.Int("attempt", attempt),
				zap.Error(err))

			time.Sleep(time.Duration(attempt) * baseBackoff)
		}

		if resp == nil || resp.StatusCode != http.StatusCreated {
			return "", nil, nil, fmt.Errorf("failed to upload shard %d after retries: %w", idx, err)
		}

		shardLocations[fmt.Sprintf("shard_%d", idx)] = nodeURL
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

	// At the end of NewStoreData:
	locBytes, err := json.Marshal(shardLocations)
	fmt.Printf("locbyte: %s\n", string(locBytes))
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to encode shard locations: %w", err)
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
			missing++
			continue
		}

		downloadURL := fmt.Sprintf("%s/shards/%s/%s/%d", nodeURL, objectID, versionID, shardIdx)

		var resp *http.Response
		var shard []byte
		var downloadErr error

		for attempt := 1; attempt <= maxDownloadRetries; attempt++ {
			req, err := http.NewRequest("GET", downloadURL, nil)
			if err != nil {
				logger.Warn("failed to create download request",
					zap.Int("shard", shardIdx), zap.Error(err))
				break
			}

			resp, err = httpClient.Do(req)
			if err != nil || resp.StatusCode != http.StatusOK {
				if resp != nil {
					resp.Body.Close()
				}
				logger.Warn("download failed, retrying...",
					zap.Int("shard", shardIdx),
					zap.String("node", nodeURL),
					zap.Int("attempt", attempt),
					zap.Error(err))
				time.Sleep(time.Duration(attempt) * baseBackoff)
				continue
			}

			shard, downloadErr = utils.ReadAllWithBuffer(resp.Body)
			resp.Body.Close()
			if downloadErr == nil {
				break
			}

			logger.Warn("read shard failed, retrying...",
				zap.Int("shard", shardIdx),
				zap.String("node", nodeURL),
				zap.Int("attempt", attempt),
				zap.Error(downloadErr))
			time.Sleep(time.Duration(attempt) * baseBackoff)
		}

		if downloadErr != nil {
			logger.Warn("permanent download failure",
				zap.Int("shard", shardIdx),
				zap.String("node", nodeURL),
				zap.Error(downloadErr))
			missing++
			continue
		}

		shards[shardIdx] = shard
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

	logger.Info("Shard download completed",
		zap.Int("total_shards", totalShards),
		zap.Int("missing", missing))

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
	storageNodes, err := LookupStorageNodes(logger)
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
		nodeURL := storageNodes[idx%len(storageNodes)]
		uploadURL := fmt.Sprintf("%s/shards/%s/%s/%d", nodeURL, objectID, versionID, idx)

		var resp *http.Response
		var err error
		for attempt := 1; attempt <= maxUploadRetries; attempt++ {
			req, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(shard))
			if err != nil {
				logger.Error("failed to create upload request",
					zap.Int("shard", idx), zap.Error(err))
				break
			}

			req.Header.Set("Content-Type", "application/octet-stream")
			resp, err = httpClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusCreated {
				resp.Body.Close()
				break
			}
			if resp != nil {
				resp.Body.Close()
			}

			logger.Warn("upload failed, retrying...",
				zap.Int("shard", idx),
				zap.String("node", nodeURL),
				zap.Int("attempt", attempt),
				zap.Error(err))

			time.Sleep(time.Duration(attempt) * baseBackoff)
		}

		if resp == nil || resp.StatusCode != http.StatusCreated {
			return "", nil, nil, fmt.Errorf("failed to upload shard %d after retries: %w", idx, err)
		}

		shardLocations[fmt.Sprintf("shard_%d", idx)] = nodeURL
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

	locBytes, err := json.Marshal(shardLocations)

	// This is for deugging
	fmt.Printf("locByte %s", locBytes)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to encode shard locations: %w", err)
	}

	logger.Info("Object stored successfully across storage nodes",
		zap.String("object_id", objectID),
		zap.String("version_id", versionID))

	return versionID, shardLocations, proofs, nil
}

func GetShardMetadata(cfg *config.Config, bucketID, objectID, versionID string) (*ShardMetadata, error) {
	db, err := bucket.InitDB()
	if err != nil {
		log.Println("Failed to initialize database")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %w", err)
	}
	defer db.Close()

	var rawLocations string
	query := `SELECT shard_locations FROM versions WHERE bucket_id = ? AND object_id = ? AND version_id = ?`
	err = db.QueryRow(query, bucketID, objectID, versionID).Scan(&rawLocations)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no metadata found for %s/%s version %s", bucketID, objectID, versionID)
	} else if err != nil {
		return nil, fmt.Errorf("failed to query metadata: %w", err)
	}

	var locations map[string]string
	if err := json.Unmarshal([]byte(rawLocations), &locations); err != nil {
		return nil, fmt.Errorf("failed to decode shard_locations: %w", err)
	}

	return &ShardMetadata{
		BucketID:       bucketID,
		ObjectID:       objectID,
		VersionID:      versionID,
		ShardLocations: locations,
	}, nil
}

func DeleteVersionShardsAcrossNodes(
	bucketID, objectID, versionID string,
	store sharding.ShardStore,
	cfg *config.Config,
	logger *zap.Logger,
) error {
	// We'll lookup the metadata across all shard locations
	metadata, err := GetShardMetadata(cfg, bucketID, objectID, versionID)
	if err != nil {
		return fmt.Errorf("metadata lookup failed: %w", err)
	}

	// Iterate over each shard location
	for shardKey, nodeURL := range metadata.ShardLocations {
		url := fmt.Sprintf("%s/shards/%s/%s/%s", nodeURL, objectID, versionID, shardKey[len("shard_"):])
		req, _ := http.NewRequest("DELETE", url, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Warn("GC: failed to contact storage node", zap.String("url", url), zap.Error(err))
			continue
		}
		resp.Body.Close()
	}
	return nil
}

// This adds property for partial override
// To get the new location we would rerun the lookup and updae the locations on the database
func UpdateShardLocations(cfg *config.Config, bucketID, objectID, versionID string, newLocations map[string]string) error {
	db, err := bucket.InitDB()
	if err != nil {
		log.Println("Failed to initialize the database")
	}
	if err != nil {
		return fmt.Errorf("failed to open DB: %w", err)
	}
	defer db.Close()

	// Get existing shard_locations
	var raw string
	err = db.QueryRow(`
		SELECT shard_locations FROM versions
		WHERE bucket_id = ? AND object_id = ? AND version_id = ?
	`, bucketID, objectID, versionID).Scan(&raw)
	if err != nil {
		return err
	}

	var current map[string]string
	if err := json.Unmarshal([]byte(raw), &current); err != nil {
		return err
	}

	// Merge maps
	for k, v := range newLocations {
		current[k] = v
	}

	merged, _ := json.Marshal(current)
	_, err = db.Exec(`
		UPDATE versions
		SET shard_locations = ?
		WHERE bucket_id = ? AND object_id = ? AND version_id = ?
	`, string(merged), bucketID, objectID, versionID)

	return err
}
