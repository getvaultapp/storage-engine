package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/api"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/utils"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

// We modify the API server to use mutual TLS and initialize tracing
// We'll need to make sure that the certificates are signed properly
func main() {
	// Load configuration and create logger.
	cfg := config.LoadConfig()
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}

	// Initialize tracing.
	cleanup := utils.InitTracer("vault-api")
	defer cleanup()

	db, err := sql.Open("sqlite3", "./vault.db")
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}
	defer db.Close()

	router := api.SetupRouter(db, cfg, logger)

	// Set up mutual TLS.
	tlsConfig, err := utils.LoadTLSConfig("certs/server.crt", "certs/server.key", "certs/ca.crt", true)
	if err != nil {
		logger.Fatal("failed to load TLS config", zap.Error(err))
	}

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "9000"
	}

	srv := &http.Server{
		Addr:      ":" + port,
		Handler:   router,
		TLSConfig: tlsConfig,
	}
	logger.Info("Starting API Gateway with mTLS", zap.String("port", port))
	log.Fatal(srv.ListenAndServeTLS("", ""))
}
