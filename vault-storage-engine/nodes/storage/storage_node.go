// cmd/storage_node/main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/sharding"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type NodeInfo struct {
	NodeID   string `json:"node_id"`
	NodeType string `json:"node_type"`
	Address  string `json:"address"`
	LastSeen int64  `json:"last_seen"`
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

// This is an helper function to register all nodes that are discovered
func registerAllDiscoveredPeers(discoveryURL string) {
	resp, err := http.Get(discoveryURL + "/nodes")
	if err != nil {
		log.Printf("Failed to fetch /nodes from discovery: %v", err)
		return
	}
	defer resp.Body.Close()

	var nodes []NodeInfo
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		log.Printf("Failed to decode /nodes: %v", err)
		return
	}

	for _, n := range nodes {
		if n.NodeID == os.Getenv("NODE_ID") {
			continue // skip self
		}
		peer := Peer{
			NodeID:        n.NodeID,
			Address:       n.Address,
			NodeType:      n.NodeType,
			Capabilities:  map[string]string{},
			LastHeartbeat: time.Now(),
		}
		data, _ := json.Marshal(peer)
		http.Post("http://localhost:"+os.Getenv("STORAGE_DISCOVERY_PORT")+"/gossip/register", "application/json", bytes.NewReader(data))
	}
}

func StartHealthCheck() {
	go func() {
		for {
			time.Sleep(30 * time.Second)
			basePath := os.Getenv("SHARD_STORE_BASE_PATH")
			if basePath == "" {
				basePath = "./data"
			}
			var stat syscall.Statfs_t
			_ = syscall.Statfs(basePath, &stat)
			available := stat.Bavail * uint64(stat.Bsize)

			nodeInfo := map[string]interface{}{
				"node_id":         os.Getenv("NODE_ID"),
				"node_type":       "storage",
				"address":         fmt.Sprintf("http://localhost:%s", os.Getenv("STORAGE_PORT")),
				"available_space": available,
				"time":            time.Now().Format(time.RFC3339),
			}
			jsonData, _ := json.Marshal(nodeInfo)
			http.Post("http://localhost:"+os.Getenv("STORAGE_DISCOVERY_PORT"), "application/json", bytes.NewReader(jsonData))
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
	r.HandleFunc("/lookup", handleLookup).Methods("GET")

	StartHealthCheck()
	StartGossip()

	port := os.Getenv("STORAGE_DISCOVERY_PORT")
	if port == "" {
		port = "9100"
	}

	go func() {
		log.Printf("Starting embedded discovery + gossip server on port %s...\n", port)
		srv := &http.Server{
			Addr:    ":" + port,
			Handler: r,
			//TLSConfig: tlsConfig,
		}
		log.Fatal(srv.ListenAndServe())
	}()
}

func main() {
	nodeID := os.Getenv("NODE_ID")
	nodeType := os.Getenv("NODE_TYPE")
	if nodeID == "" || nodeType != "storage" {
		log.Fatalf("NODE_ID must be set and NODE_TYPE must be 'storage'")
	}
	basePath := os.Getenv("SHARD_STORE_BASE_PATH")
	if basePath == "" {
		basePath = "./data"
	}
	store := sharding.NewLocalShardStore(basePath)

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Logger error: %v", err)
	}

	startDiscoveryAndP2P()

	// We'll give embedded server 1s to bind
	time.Sleep(1 * time.Second)

	// Register with Discovery Service.
	discoveryURL := os.Getenv("DISCOVERY_URL")
	//discoveryURL := fmt.Sprintf("")
	if discoveryURL == "" {
		log.Fatal("DISCOVERY_URL must be set for storage node")
	}
	registerWithDiscovery(nodeID, discoveryURL, fmt.Sprintf("http://localhost:%s", os.Getenv("STORAGE_PORT")))

	registerAllDiscoveredPeers(discoveryURL)

	r := mux.NewRouter()
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}).Methods("GET")
	r.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		info := map[string]string{
			"node_id":   nodeID,
			"node_type": nodeType,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}).Methods("GET")

	r.HandleFunc("/diskspace", handleDiskSpace).Methods("GET")

	r.HandleFunc("/verify/{objectID}/{versionID}/{shardIdx}", handleVerifyShard).Methods("GET")

	r.HandleFunc("/shards/{objectID}/{versionID}/{shardIdx}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		objectID := vars["objectID"]
		versionID := vars["versionID"]
		shardIdxStr := vars["shardIdx"]
		shardIdx, err := strconv.Atoi(shardIdxStr)
		if err != nil {
			http.Error(w, "Invalid shard index", http.StatusBadRequest)
			return
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read shard data", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		err = store.StoreShard(objectID, versionID, shardIdx, data, nodeID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to store shard: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}).Methods("PUT")

	r.HandleFunc("/shards/{objectID}/{versionID}/{shardIdx}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		objectID := vars["objectID"]
		versionID := vars["versionID"]
		shardIdxStr := vars["shardIdx"]
		shardIdx, err := strconv.Atoi(shardIdxStr)
		if err != nil {
			http.Error(w, "Invalid shard index", http.StatusBadRequest)
			return
		}
		data, err := store.RetrieveShard(objectID, versionID, shardIdx, nodeID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to retrieve shard: %v", err), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(data)
	}).Methods("GET")

	r.HandleFunc("/shards/{objectID}/{versionID}/{shardIdx}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		objectID := vars["objectID"]
		versionID := vars["versionID"]
		shardIdxStr := vars["shardIdx"]
		shardIdx, err := strconv.Atoi(shardIdxStr)
		if err != nil {
			http.Error(w, "Invalid shard index", http.StatusBadRequest)
			return
		}
		err = store.DeleteShardByVersion(objectID, versionID, shardIdx, nodeID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete shard: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	}).Methods("DELETE")

	port := os.Getenv("STORAGE_PORT")
	if port == "" {
		port = "8080"
	}

	/* tlsConfig, err := utils.LoadTLSConfig(
		"/home/tnxl/storage-engine/vault-storage-engine/certs/server.crt",
		"/home/tnxl/storage-engine/vault-storage-engine/certs/server.key",
		"/home/tnxl/storage-engine/vault-storage-engine/certs/ca.crt",
		true,
	) */
	/* if err != nil {
		log.Fatalf("Failed to load TLS config: %v", err)
	}
	*/
	// start embedded discovery and gossip
	startDiscoveryAndP2P()

	logger.Info("Starting Storage Node", zap.String("node_id", nodeID), zap.String("port", port))
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func registerWithDiscovery(nodeID, discoveryURL, selfAddress string) {
	regURL := discoveryURL + "/register"
	payload := map[string]string{
		"node_id":   nodeID,
		"node_type": "storage",
		"address":   selfAddress,
	}
	data, _ := json.Marshal(payload)
	client := &http.Client{}

	var resp *http.Response
	var err error
	for attempt := 1; attempt <= 5; attempt++ {
		resp, err = client.Post(regURL, "application/json", bytes.NewReader(data))
		if err == nil {
			break
		}
		log.Printf("Discovery registration failed (attempt %d): %v", attempt, err)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	if err != nil {
		log.Printf("Final failure registering with discovery after retries: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("Node %s registered with discovery", nodeID)
}

// Add health check monitoring to report available space
func handleDiskSpace(w http.ResponseWriter, r *http.Request) {
	basePath := os.Getenv("SHARD_STORE_BASE_PATH")

	// Get available disk space on the storage node
	var stat syscall.Statfs_t
	err := syscall.Statfs(basePath, &stat)
	if err != nil {
		http.Error(w, "Failed to get disk statistics", http.StatusInternalServerError)
		return
	}

	// Calculate available space in bytes
	available := stat.Bavail * uint64(stat.Bsize)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"available_bytes": available,
		"available_gb":    float64(available) / (1024 * 1024 * 1024),
	})
}

// Add shard verification endpoint
func handleVerifyShard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	objectID := vars["objectID"]
	versionID := vars["versionID"]
	shardIdxStr := vars["shardIdx"]

	nodeID := os.Getenv("NODE_ID")

	shardIdx, err := strconv.Atoi(shardIdxStr)
	if err != nil {
		http.Error(w, "Invalid shard index", http.StatusBadRequest)
		return
	}

	// Check if the shard exists
	exists, err := shardExists(objectID, versionID, shardIdx, nodeID)
	if err != nil {
		http.Error(w, "Error checking shard", http.StatusInternalServerError)
		log.Printf("Error: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"exists": exists,
	})
}

func shardExists(objectID, versionID string, shardIdx int, nodeID string) (bool, error) {
	shardBase := os.Getenv("SHARD_STORE_BASE_PATH")
	shardPath := filepath.Join(shardBase, nodeID, fmt.Sprintf("%s-v(%s)_shard_%d", objectID, versionID, shardIdx))
	_, err := os.ReadFile(shardPath)
	if os.IsNotExist(err) {
		log.Printf("%s does not have the shard", nodeID)
		return false, fmt.Errorf("file does not exists: %v", err)
	}

	return true, nil
}
