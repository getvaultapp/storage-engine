package main

import (
	"encoding/json"
	"hash/fnv"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"
)

type NodeInfo struct {
	NodeID   string `json:"node_id"`
	NodeType string `json:"node_type"`
	Address  string `json:"address"`
	LastSeen int64  `json:"last_seen"`
	Hash     uint32 `json:"-"`
}

var (
	nodeRegistry = make(map[string]NodeInfo)
	registryLock sync.RWMutex
)

func main() {
	http.HandleFunc("/register", RegisterHandler)
	http.HandleFunc("/nodes", nodesHandler)
	http.HandleFunc("/lookup", lookupHandler)
	go cleanupStaleNodes()

	log.Println("Discovery service started on :8000")
	log.Fatal(http.ListenAndServeTLS(":8000", "/home/tnxl/storage-engine/vault-storage-engine/nodes/certs/server.crt", "/home/tnxl/storage-engine/vault-storage-engine/nodes/certs/server.key", nil))
	//log.Fatal(http.ListenAndServe(":8000", nil))
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	var node NodeInfo
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	node.LastSeen = time.Now().Unix()
	node.Hash = hashString(node.NodeID)
	registryLock.Lock()
	nodeRegistry[node.NodeID] = node
	registryLock.Unlock()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
}

func nodesHandler(w http.ResponseWriter, r *http.Request) {
	registryLock.RLock()
	nodes := make([]NodeInfo, 0, len(nodeRegistry))

	for _, node := range nodeRegistry {
		//fmt.Println(nodes)
		nodes = append(nodes, node)
	}
	registryLock.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

func lookupHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Missing key parameter", http.StatusBadRequest)
		return
	}
	h := hashString(key)
	registryLock.RLock()
	var nodes []NodeInfo
	for _, node := range nodeRegistry {
		nodes = append(nodes, node)
	}
	registryLock.RUnlock()
	sort.Slice(nodes, func(i, j int) bool {
		return distance(h, nodes[i].Hash) < distance(h, nodes[j].Hash)
	})
	if len(nodes) == 0 {
		http.Error(w, "No nodes available", http.StatusNotFound)
		return
	}
	// Return the top 3 closest nodes.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes[:min(len(nodes), 3)])
}

func hashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func distance(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

func cleanupStaleNodes() {
	for {
		time.Sleep(60 * time.Second)
		now := time.Now().Unix()
		registryLock.Lock()
		for id, node := range nodeRegistry {
			if now-node.LastSeen > 120 {
				delete(nodeRegistry, id)
			}
		}
		registryLock.Unlock()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
