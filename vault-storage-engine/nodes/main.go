package main

import (
	"log"
	"net/http"
	"os"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/api"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/utils"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

// API gateway for our latest services
func main() {
	cfg := config.LoadConfig()
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Logger error: %v", err)
	}

	cleanup := utils.InitTracer("vault-api")
	defer cleanup()

	// Initialize database
	db, err := bucket.InitDB()
	if err != nil {
		logger.Fatal("DB connection error", zap.Error(err))
	}
	defer db.Close()

	router := api.SetupRouter(db, cfg, logger)

	tlsConfig, err := utils.LoadTLSConfig("certs/server.crt", "certs/server.key", "certs/ca.crt", true)
	if err != nil {
		logger.Fatal("TLS config error", zap.Error(err))
	}

	port := os.Getenv("API_PORT")
	if port == "" {
		//port = "9000"
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
