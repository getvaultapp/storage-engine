// storage_node/main.go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/sharding"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

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
		log.Fatalf("failed to create logger: %v", err)
	}

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

	// Endpoint to store a shard
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

		// Use nodeID as the location for this storage node.
		err = store.StoreShard(objectID, versionID, shardIdx, data, nodeID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to store shard: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}).Methods("PUT")

	// Endpoint to retrieve a shard
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

	// Endpoint to delete a shard (by version)
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
	logger.Info("Starting Storage Node", zap.String("node_id", nodeID), zap.String("port", port))
	log.Fatal(http.ListenAndServe(":"+port, r))
}
