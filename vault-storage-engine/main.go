package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/api"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/utils"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
		os.Exit(1)
	}
}

func main() {
	// Initialize configuration
	initConfig()

	// Convert viper.Viper to config.Config
	cfg := utils.ConvertViperToConfig(viper.GetViper())

	// Set up logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Database connection
	db, err := sql.Open("sqlite3", cfg.Database)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	defer db.Close()

	// Initialize the router
	router := api.SetupRouter(db, cfg, logger)

	// Start the server
	if err := router.Run(); err != nil {
		log.Fatalf("Error starting the server: %v", err)
	}
}
