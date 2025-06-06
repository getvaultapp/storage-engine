// cmd/construction_node/main.go
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/datastorage"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/sharding"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/utils"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// --- DISCOVERY / P2P ---
type NodeInfo struct {
	NodeID   string `json:"node_id"`
	NodeType string `json:"node_type"`
	Address  string `json:"address"`
	LastSeen int64  `json:"last_seen"`
	Hash     uint32 `json:"-"`
}

type Peer struct {
	NodeID         string            `json:"node_id"`
	Address        string            `json:"address"`
	NodeType       string            `json:"node_type"`
	Capabilities   map[string]string `json:"capabilities"`
	LastHeartbeat  time.Time         `json:"last_heartbeat"`
	AvailableSpace int64             `json:"available_space,omitempty"`
}

var (
	nodeRegistry = make(map[string]NodeInfo)
	registryLock sync.RWMutex
	peerList     []Peer
	peerLock     sync.RWMutex
)

// --- DISCOVERY / P2P: Helpers ---
func registerHandler(w http.ResponseWriter, r *http.Request) {
	var node NodeInfo
	json.NewDecoder(r.Body).Decode(&node)
	node.LastSeen = time.Now().Unix()
	registryLock.Lock()
	nodeRegistry[node.NodeID] = node
	registryLock.Unlock()
	json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
}

func nodesHandler(w http.ResponseWriter, r *http.Request) {
	registryLock.RLock()
	var nodes []NodeInfo
	for _, node := range nodeRegistry {
		nodes = append(nodes, node)
	}
	registryLock.RUnlock()
	json.NewEncoder(w).Encode(nodes)
}

func GossipRegisterHandler(w http.ResponseWriter, r *http.Request) {
	var p Peer
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	peerLock.Lock()
	for _, existing := range peerList {
		if existing.NodeID == p.NodeID {
			peerLock.Unlock()
			return
		}
	}
	peerList = append(peerList, p)
	peerLock.Unlock()
	w.WriteHeader(http.StatusOK)
}

func GossipListHandler(w http.ResponseWriter, r *http.Request) {
	peerLock.RLock()
	json.NewEncoder(w).Encode(peerList)
	peerLock.RUnlock()
}

func StartHealthCheck() {
	go func() {
		for {
			time.Sleep(30 * time.Second)
			nodeInfo := map[string]interface{}{
				"node_id":   myNodeID,
				"node_type": "construction",
				"address":   fmt.Sprintf("http://localhost:%s", os.Getenv("CONSTRUCTION_PORT")),
				"time":      time.Now().Format(time.RFC3339),
			}
			jsonData, _ := json.Marshal(nodeInfo)
			http.Post("http://localhost:"+os.Getenv("DISCOVERY_PORT")+"/register", "application/json", bytes.NewReader(jsonData))
		}
	}()
}

func StartGossip() {
	go func() {
		for {
			time.Sleep(10 * time.Second)
			peerLock.RLock()
			if len(peerList) == 0 {
				peerLock.RUnlock()
				continue
			}
			target := peerList[rand.Intn(len(peerList))]
			peerLock.RUnlock()

			url := target.Address + "/gossip/peers"
			resp, err := http.Get(url)
			if err != nil {
				continue
			}
			var remotePeers []Peer
			json.NewDecoder(resp.Body).Decode(&remotePeers)
			resp.Body.Close()

			peerLock.Lock()
			for _, rp := range remotePeers {
				found := false
				for _, p := range peerList {
					if p.NodeID == rp.NodeID {
						found = true
						break
					}
				}
				if !found && len(peerList) < 50 {
					peerList = append(peerList, rp)
				}
			}
			peerLock.Unlock()
		}
	}()
}

func startDiscoveryAndP2P() {
	r := mux.NewRouter()
	r.HandleFunc("/register", registerHandler)
	r.HandleFunc("/nodes", nodesHandler)
	r.HandleFunc("/gossip/register", GossipRegisterHandler)
	r.HandleFunc("/gossip/peers", GossipListHandler)
	r.HandleFunc("/lookup", handleLookup)

	StartHealthCheck()
	StartGossip()

	discoveryPort := os.Getenv("DISCOVERY_PORT")
	if discoveryPort == "" {
		discoveryPort = "9000"
		log.Println("Discovery Port Set To Default")
	}

	go func() {
		log.Printf("Starting discovery + gossip server on port %s...\n", discoveryPort)
		srv := &http.Server{
			Addr:    ":" + discoveryPort,
			Handler: r,
			//TLSConfig: tlsConfig,
		}
		log.Fatal(srv.ListenAndServe())
	}()
}

var (
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

func pingPong(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
}

// HTTP handlers for construction node.
// If the node is active then it is okay
func handleHealth(w http.ResponseWriter, r *http.Request) {
	// TODO let's get more advanced means of checking the functionality of a construction node
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

func handleProcessFile(w http.ResponseWriter, r *http.Request, db *sql.DB, store sharding.ShardStore, cfg *config.Config, logger *zap.Logger) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Get the objectID, bucketID and filename from the request
	objectID := r.Header.Get("X-Object-ID")
	bucketID := r.Header.Get("X-Bucket-ID")
	fileName := r.Header.Get("X-Filename")
	if objectID == "" || bucketID == "" || fileName == "" {
		http.Error(w, "Missing required headers", http.StatusBadRequest)
		return
	}

	// Generate a new version ID.
	versionID := uuid.New().String()

	// Register the task in our queue
	registerTask(bucketID, objectID, versionID, fileName, data)

	// Instead of immediately processing, we could implement a work queue
	// But for now, let's claim and process the task immediately
	task, ok := claimTask(objectID)
	if !ok {
		http.Error(w, "Task already assigned", http.StatusConflict)
		return
	}

	// Process the task asynchronously
	go func() {
		// Use the new distributed storage functionality
		_, shardLocations, _, err := datastorage.NewStoreData(
			db,
			task.Data,
			task.BuucketID,
			task.ObjectID,
			task.FileName,
			store,
			cfg,
			[]string{}, // This is not used anymore as nodes are discovered dynamically
			logger,
		)

		if err != nil {
			logger.Error("Failed to store data", zap.Error(err))
			return
		}

		logger.Info("Data stored successfully",
			zap.String("object_id", task.ObjectID),
			zap.String("version_id", task.VersionID),
			zap.Any("shard_locations", shardLocations))
	}()

	response := map[string]string{
		"object_id":  objectID,
		"version_id": versionID,
		"status":     "processing started",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// This should correctly list all versions of the object_id and bucket_id from the console of the construction node
func handleListVersions(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	bucketID := r.URL.Query().Get("bucket_id")
	objectID := r.URL.Query().Get("object_id")

	if bucketID == "" || objectID == "" {
		http.Error(w, "Missing bucket_id or object_id", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`
		SELECT version_id, created_at, shard_locations
		FROM versions
		WHERE bucket_id = ? AND object_id = ?
		ORDER BY created_at DESC
	`, bucketID, objectID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var versions []map[string]interface{}
	for rows.Next() {
		var versionID, createdAt, shardLocations string
		if err := rows.Scan(&versionID, &createdAt, &shardLocations); err != nil {
			continue
		}
		var loc map[string]string
		_ = json.Unmarshal([]byte(shardLocations), &loc)

		versions = append(versions, map[string]interface{}{
			"version_id":      versionID,
			"created_at":      createdAt,
			"shard_locations": loc,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}

func handleDeleteFileFromStorage(w http.ResponseWriter, r *http.Request, db *sql.DB, store sharding.ShardStore, cfg *config.Config, logger *zap.Logger) {

}

func handleReconstructFile(w http.ResponseWriter, r *http.Request, db *sql.DB, store sharding.ShardStore, cfg *config.Config, logger *zap.Logger) {
	var req struct {
		BucketID  string `json:"bucket_id"`
		ObjectID  string `json:"object_id"`
		VersionID string `json:"version_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		log.Printf("%v", err)
		return
	}
	defer r.Body.Close()

	// Use the new distributed retrieve function
	data, fileName, err := datastorage.NewRetrieveData(
		db,
		req.BucketID,
		req.ObjectID,
		req.VersionID,
		store,
		cfg,
		logger,
	)

	if err != nil {
		logger.Error("Reconstruction failed", zap.Error(err))
		http.Error(w, fmt.Sprintf("Reconstruction failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data)
}

// This should work along with the discovery, not the construction port
func handleLookup(w http.ResponseWriter, r *http.Request) {
	storageNodes := []map[string]string{}

	peerLock.RLock()
	for _, p := range peerList {
		if p.NodeType == "storage" {
			storageNodes = append(storageNodes, map[string]string{
				"address": p.Address,
			})
		}
	}
	peerLock.RUnlock()

	// Fallback to nodeRegistry if peerList is empty
	if len(storageNodes) == 0 {
		registryLock.RLock()
		for _, n := range nodeRegistry {
			if n.NodeType == "storage" {
				storageNodes = append(storageNodes, map[string]string{
					"address": n.Address,
				})
			}
		}
		registryLock.RUnlock()
	}

	if len(storageNodes) == 0 {
		http.Error(w, "no storage nodes available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(storageNodes)
}

func registerAllDiscoveredPeersFromRegistry() {
	resp, err := http.Get("http://localhost:" + os.Getenv("DISCOVERY_PORT") + "/nodes")
	if err != nil {
		log.Printf("Failed to fetch /nodes: %v", err)
		return
	}
	defer resp.Body.Close()

	var nodes []NodeInfo
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		log.Printf("Failed to decode /nodes: %v", err)
		return
	}

	for _, node := range nodes {
		if node.NodeType != "storage" || node.NodeID == myNodeID {
			continue
		}
		peer := Peer{
			NodeID:        node.NodeID,
			Address:       node.Address,
			NodeType:      node.NodeType,
			Capabilities:  map[string]string{},
			LastHeartbeat: time.Now(),
		}
		data, _ := json.Marshal(peer)
		http.Post("http://localhost:"+os.Getenv("DISCOVERY_PORT")+"/gossip/register", "application/json", bytes.NewReader(data))
	}
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
	/*tlsConfig, err := utils.LoadTLSConfig("/home/tnxl/storage-engine/vault-storage-engine/certs/server.crt",
		"/home/tnxl/storage-engine/vault-storage-engine/certs/server.key",
		"/home/tnxl/storage-engine/vault-storage-engine/certs/ca.crt", true)
	if err != nil {
		log.Fatalf("TLS config error: %v", err)
	} */

	/* mtlsClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	} */

	db, err := bucket.InitDB()
	if err != nil {
		log.Fatalf("DB init error: %v", err)
	}
	defer db.Close()

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Logger init error: %v", err)
	}

	startDiscoveryAndP2P()

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		// Run once on startup
		registerAllDiscoveredPeersFromRegistry()

		for {
			select {
			case <-ticker.C:
				registerAllDiscoveredPeersFromRegistry()
			}
		}
	}()

	r := mux.NewRouter()
	r.HandleFunc("/ping", pingPong).Methods("GET")
	r.HandleFunc("/health", handleHealth).Methods("GET")
	r.HandleFunc("/info", handleInfo).Methods("GET")
	r.HandleFunc("/lookup", handleLookup).Methods("GET")
	r.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		handleProcessFile(w, r, db, store, cfg, logger)
	}).Methods("POST")
	r.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		handleDeleteFileFromStorage(w, r, db, store, cfg, logger)
	}).Methods("DELETE") // This should delete a file from all storage locations
	r.HandleFunc("/reconstruct", func(w http.ResponseWriter, r *http.Request) {
		handleReconstructFile(w, r, db, store, cfg, logger)
	}).Methods("POST")
	r.HandleFunc("/versions", func(w http.ResponseWriter, r *http.Request) {
		handleListVersions(w, r, db)
	}).Methods("GET")

	port := os.Getenv("CONSTRUCTION_PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
		//TLSConfig: tlsConfig,
	}

	logger.Info("Starting Construction Node", zap.String("node_id", myNodeID), zap.String("port", port))
	log.Fatal(srv.ListenAndServe())
}
