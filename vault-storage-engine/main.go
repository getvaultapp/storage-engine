package main

import (
	"log"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/api"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"

	//vault_cli "github.com/getvaultapp/storage-engine/vault-storage-engine/run_cli/cli_cmd"
	"go.uber.org/zap"
)

func main() {
	// Set up logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	db, err := bucket.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize the router
	router := api.SetupRouter(db, cfg, logger)

	// Start the server
	if err := router.Run(cfg.ServerAddress); err != nil {
		log.Fatalf("Error starting the server: %v", err)
	}

	// Setup CLI
	//vault_cli.RunCli()

	/* jwt, _ := auth.GenerateJWT("hello", "hello@email.com", "user")
	println(jwt)

	username, _ := auth.GetUsernameFromToken(jwt)
	println(username) */
}
