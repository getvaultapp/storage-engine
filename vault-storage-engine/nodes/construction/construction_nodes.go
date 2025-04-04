// cmd/construction_node/main.go
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/compression"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/datastorage"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/encryption"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/erasurecoding"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/proofofinclusion"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/sharding"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/utils"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

var (
	// For simplicity, tasks are registered here.
	taskQueue   = make(map[string]PendingTask)
	taskQueueMu sync.Mutex
	myNodeID    = os.Getenv("NODE_ID")
)

// PendingTask represents a file processing task.
type PendingTask struct {
	BuucketID string
	ObjectID  string
	VersionID string
	Data      []byte
	FileName  string
	CreatedAt time.Time
	Assigned  bool
}

func registerTask(bucketID, objectID, versionID, fileName string, data []byte) {
	taskQueueMu.Lock()
	defer taskQueueMu.Unlock()
	taskQueue[objectID] = PendingTask{
		BuucketID: bucketID,
		ObjectID:  objectID,
		VersionID: versionID,
		Data:      data,
		FileName:  fileName,
		CreatedAt: time.Now(),
		Assigned:  false,
	}
}

func claimTask(objectID string) (PendingTask, bool) {
	taskQueueMu.Lock()
	defer taskQueueMu.Unlock()
	task, exists := taskQueue[objectID]
	if exists && !task.Assigned {
		task.Assigned = true
		taskQueue[objectID] = task
		return task, true
	}
	return PendingTask{}, false
}

func processTask(task PendingTask, db *sql.DB, store sharding.ShardStore, cfg *config.Config, logger *zap.Logger, mtlsClient *http.Client) {
	logger.Info("Processing task", zap.String("object_id", task.ObjectID))
	// Process the file through compression, encryption, erasure coding, and Merkle tree generation.
	compressedData, err := compression.Compress(task.Data)
	if err != nil {
		logger.Error("Compression failed", zap.Error(err))
		return
	}

	key := cfg.EncryptionKey
	cipherText, err := encryption.Encrypt(compressedData, key)
	if err != nil {
		logger.Error("Encryption failed", zap.Error(err))
		return
	}

	shards, err := erasurecoding.Encode(cipherText)
	if err != nil {
		logger.Error("Erasure coding failed", zap.Error(err))
		return
	}

	tree, err := proofofinclusion.BuildMerkleTree(shards)
	if err != nil {
		logger.Error("Merkle tree build failed", zap.Error(err))
		return
	}

	// This should discover storage nodes by looking up available nodes via DHT
	storageNodes, err := lookupStorageNodes(task.ObjectID)
	if err != nil {
		logger.Error("Storage node lookup failed", zap.Error(err))
		return
	}
	if len(storageNodes) < len(shards) {
		logger.Error("Not enough storage nodes available", zap.Int("required", len(shards)), zap.Int("found", len(storageNodes)))
		return
	}

	shardLocations := make(map[string]string)
	for idx, shard := range shards {
		nodeURL := storageNodes[idx%len(storageNodes)]
		uploadURL := fmt.Sprintf("%s/shards/%s/%s/%d", nodeURL, task.ObjectID, task.VersionID, idx)
		req, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(shard))
		if err != nil {
			logger.Error("Failed to create shard upload request", zap.Error(err))
			return
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		resp, err := mtlsClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusCreated {
			logger.Error("Failed to upload shard", zap.Int("shard", idx), zap.Error(err))
			return
		}
		resp.Body.Close()
		shardLocations[fmt.Sprintf("shard_%d", idx)] = nodeURL
	}

	// Generate proofs for each shard.
	var proofs []string
	for _, shard := range shards {
		proof, err := proofofinclusion.GetProof(tree, shard)
		if err != nil {
			logger.Error("Failed to get proof", zap.Error(err))
			return
		}
		proofs = append(proofs, proof)
	}

	// Save metadata in the database.
	metadata := bucket.VersionMetadata{
		BucketID:       "example-bucket", // We'll need to get the actual bucketID instead of hardcoding it
		ObjectID:       task.ObjectID,
		VersionID:      task.VersionID,
		Filename:       task.FileName,
		Filesize:       "",
		Format:         "", // Could derive from file extension.
		CreationDate:   time.Now().Format(time.RFC3339),
		ShardLocations: shardLocations,
		Proofs:         utils.ConvertSliceToMap(proofs),
	}
	rootVersion, _ := bucket.GetRootVersion(db, task.ObjectID)
	err = bucket.AddVersion(db, "example-bucket", task.ObjectID, task.VersionID, rootVersion, metadata, cipherText)
	if err != nil {
		logger.Error("Failed to add version to DB", zap.Error(err))
		return
	}
	err = bucket.AddObject(db, "example-bucket", task.ObjectID, task.FileName)
	if err != nil {
		logger.Error("Failed to register object in DB", zap.Error(err))
		return
	}
	logger.Info("Task processed successfully", zap.String("object_id", task.ObjectID))
}

func lookupStorageNodes(key string) ([]string, error) {
	// For local testing, query the discovery service's lookup endpoint.
	lookupURL := fmt.Sprintf("https://localhost:8000/lookup?key=%s", key)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(lookupURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var nodes []struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return nil, err
	}

	var urls []string
	for _, n := range nodes {
		urls = append(urls, n.Address)
	}
	return urls, nil
}

// HTTP handlers for construction node.
// If the node is active then it is okay
func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]string{
		"node_id":   os.Getenv("NODE_ID"),
		"node_type": os.Getenv("NODE_TYPE"),
		"time":      time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func handleProcessFile(w http.ResponseWriter, r *http.Request, db *sql.DB, store sharding.ShardStore, cfg *config.Config, logger *zap.Logger, mtlsClient *http.Client) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	objectID := r.Header.Get("X-Object-ID")
	bucketID := r.Header.Get("X-Bucket-ID")
	fileName := r.Header.Get("X-Filename")
	if objectID == "" || bucketID == "" || fileName == "" {
		http.Error(w, "Missing required headers", http.StatusBadRequest)
		return
	}

	// Generate a new version ID.
	versionID := uuid.New().String()
	registerTask(bucketID, objectID, versionID, fileName, data)
	// Claim and process the task immediately (for local testing).
	task, ok := claimTask(objectID)
	if !ok {
		http.Error(w, "Task already assigned", http.StatusConflict)
		return
	}

	// Process the task.
	processTask(task, db, store, cfg, logger, mtlsClient)

	response := map[string]string{
		"object_id":  objectID,
		"version_id": versionID,
		"status":     "processing started",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// This should get the shards from storage nodes
func handleReconstructFile(w http.ResponseWriter, r *http.Request, db *sql.DB, store sharding.ShardStore, cfg *config.Config, logger *zap.Logger, mtlsClient *http.Client) {
	var req struct {
		BucketID  string `json:"bucket_id"`
		ObjectID  string `json:"object_id"`
		VersionID string `json:"version_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	data, fileName, err := datastorage.RetrieveData(db, req.BucketID, req.ObjectID, req.VersionID, store, cfg, logger)
	if err != nil {
		http.Error(w, fmt.Sprintf("Reconstruction failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data)
}

func main() {
	cfg := config.LoadConfig()
	nodeType := os.Getenv("NODE_TYPE")
	if nodeType != "construction" {
		log.Fatalf("NODE_TYPE must be 'construction'")
	}
	myNodeID = os.Getenv("NODE_ID")
	if myNodeID == "" {
		log.Fatalf("NODE_ID must be set")
	}

	// Initialize tracing and mTLS.
	cleanup := utils.InitTracer("vault-construction")
	defer cleanup()
	tlsConfig, err := utils.LoadTLSConfig("/home/tnxl/storage-engine/vault-storage-engine/nodes/certs/server.crt", "/home/tnxl/storage-engine/vault-storage-engine/nodes/certs/server.key", "/home/tnxl/storage-engine/vault-storage-engine/nodes/certs/ca.crt", true)
	if err != nil {
		log.Fatalf("TLS config error: %v", err)
	}

	mtlsClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	db, err := bucket.InitDB() // Initialize your DB (e.g., SQLite)
	if err != nil {
		log.Fatalf("DB init error: %v", err)
	}
	defer db.Close()

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Logger init error: %v", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/health", handleHealth).Methods("GET")
	r.HandleFunc("/info", handleInfo).Methods("GET")
	r.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		handleProcessFile(w, r, db, store, cfg, logger, mtlsClient)
	}).Methods("POST")
	r.HandleFunc("/reconstruct", func(w http.ResponseWriter, r *http.Request) {
		handleReconstructFile(w, r, db, store, cfg, logger, mtlsClient)
	}).Methods("POST")

	port := os.Getenv("CONSTRUCTION_PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:      ":" + port,
		Handler:   r,
		TLSConfig: tlsConfig,
	}

	logger.Info("Starting Construction Node", zap.String("node_id", myNodeID), zap.String("port", port))
	log.Fatal(srv.ListenAndServeTLS("", ""))
}
