// construction_node/main.go
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/datastorage"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/sharding"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/utils"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func main() {
	cfg := config.LoadConfig()

	nodeID := os.Getenv("NODE_ID")
	nodeType := os.Getenv("NODE_TYPE")
	if nodeID == "" || nodeType != "construction" {
		log.Fatalf("NODE_ID must be set and NODE_TYPE must be 'construction'")
	}

	// This should initialize tracing
	cleanup := utils.InitTracer("vault-construction")
	defer cleanup()

	// We'll setup mutual TLS for outboubd calls to storage nodes (if needed) and for this server
	tlsConfig, err := utils.LoadTLSConfig("certs/server.crt", "certs/server.key", "certs/ca.crt", true)
	if err != nil {
		log.Fatalf("failed to load the TLS config, %v", err)
	}

	// Create an HTTP clients with mTLS for shard distrubuition
	mtlsClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// We'll pass mtlsClient to datastorage if needed to send shard storage requests to storage nodes

	//db, err := sql.Open("sqlite3", "./vault.db")

	db, err := bucket.InitDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Local shard store for temporary processing (in production, this may be a remote call)
	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/health", handleHealth).Methods("GET")
	r.HandleFunc("/info", handleInfo).Methods("GET")
	r.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		handleProcessFile(w, r, db, store, cfg, logger)
	}).Methods("POST")
	r.HandleFunc("/reconstruct", func(w http.ResponseWriter, r *http.Request) {
		handleReconstructFile(w, r, db, store, cfg, logger)
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

	// Inidcate that we are already running the construction server
	logger.Info("Starting Construction Node with mTLS", zap.String("port", port))
	log.Fatal(srv.ListenAndServeTLS("", ""))
}

// otelMiddleware propagates tracing context.
func otelMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create new span for incoming request.
		ctx, span := otel.Tracer("vault-construction").Start(r.Context(), r.RequestURI)
		defer span.End()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

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

func handleProcessFile(w http.ResponseWriter, r *http.Request, db *sql.DB, store sharding.ShardStore, cfg *config.Config, logger *zap.Logger) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Expect headers: X-Object-ID, X-Bucket-ID, X-Filename
	objectID := r.Header.Get("X-Object-ID")
	bucketID := r.Header.Get("X-Bucket-ID")
	filename := r.Header.Get("X-Filename")
	if objectID == "" || bucketID == "" || filename == "" {
		http.Error(w, "Missing required headers", http.StatusBadRequest)
		return
	}

	versionID, shardLocations, proofs, err := datastorage.StoreData(db, data, bucketID, objectID, filename, store, cfg, cfg.ShardLocations, logger)
	if err != nil {
		http.Error(w, fmt.Sprintf("Processing failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"object_id":       objectID,
		"version_id":      versionID,
		"shard_locations": shardLocations,
		"proofs":          proofs,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleReconstructFile(w http.ResponseWriter, r *http.Request, db *sql.DB, store sharding.ShardStore, cfg *config.Config, logger *zap.Logger) {
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

	data, filename, err := datastorage.RetrieveData(db, req.BucketID, req.ObjectID, req.VersionID, store, cfg, logger)
	if err != nil {
		http.Error(w, fmt.Sprintf("Reconstruction failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data)
}
