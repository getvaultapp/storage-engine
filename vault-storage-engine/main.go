package main

import (
	"log"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/api"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/auth"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/utils"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}
}

func main() {
	// Initialize configuration
	initConfig()

	// Initialize JWT secret
	auth.InitJWTSecret()

	// Convert viper.Viper to config.Config
	cfg := utils.ConvertViperToConfig(viper.GetViper())

	// Set up logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Initialize the database
	db, err := bucket.InitDB()
	if err != nil {
		log.Fatalf("Error initializing the database: %v", err)
	}
	defer db.Close()

	// Initialize the router
	router := api.SetupRouter(db, cfg, logger)

	// Start the server
	if err := router.Run(cfg.ServerAddress); err != nil {
		log.Fatalf("Error starting the server: %v", err)
	}
}
