// discovery_service/main.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type NodeInfo struct {
	NodeID   string `json:"node_id"`
	NodeType string `json:"node_type"`
	Address  string `json:"address"`
	LastSeen int64  `json:"last_seen"`
}

var (
	nodeRegistry = make(map[string]NodeInfo)
	registryLock sync.RWMutex
)

func main() {
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/nodes", nodesHandler)
	// Periodically clean up stale nodes.
	go cleanupStaleNodes()

	log.Println("Discovery service started on :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
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
		nodes = append(nodes, node)
	}
	registryLock.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
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
