package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/getvault-mvp/vault-base/pkg/api"
	"github.com/getvault-mvp/vault-base/pkg/bucket"
	"github.com/getvault-mvp/vault-base/pkg/config"
	"github.com/getvault-mvp/vault-base/pkg/datastorage"
	"github.com/getvault-mvp/vault-base/pkg/sharding"
	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func storeCommand(c *cli.Context, db *sql.DB, cfg *config.Config, logger *zap.Logger) error {
	if c.NArg() < 2 {
		return fmt.Errorf("usage: store <bucket_id> <file_path>")
	}

	bucketID := c.Args().Get(0)
	filePath := c.Args().Get(1)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)
	// Initialize locations with enough entries
	locations := []string{"location1", "location2", "location3", "location4", "location5", "location6", "location7", "location8"} // Example locations, ensure there are enough
	objectID := uuid.New().String()                                                                                               // Generate a unique object ID
	versionID, err := datastorage.StoreData(db, data, bucketID, objectID, filepath.Base(filePath), store, cfg, locations, logger)
	if err != nil {
		return fmt.Errorf("store failed: %w", err)
	}

	fmt.Printf("Stored file as version %s in bucket %s\n", versionID, bucketID)
	return nil
}

func retrieveCommand(c *cli.Context, db *sql.DB, cfg *config.Config, logger *zap.Logger) error {
	if c.NArg() < 3 {
		return fmt.Errorf("usage: retrieve <bucket_id> <object_id> <version_id>")
	}

	bucketID := c.Args().Get(0)
	objectID := c.Args().Get(1)
	versionID := c.Args().Get(2)

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)
	data, err := datastorage.RetrieveData(db, bucketID, objectID, versionID, store, cfg, logger)
	if err != nil {
		return fmt.Errorf("retrieve failed: %w", err)
	}

	fmt.Printf("Retrieved data: %s\n", string(data))
	return nil
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := config.LoadConfig()

	db, err := bucket.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "store",
				Aliases: []string{"s"},
				Usage:   "Store data. Usage: store <bucket_id> <file_path>",
				Action: func(c *cli.Context) error {
					return storeCommand(c, db, cfg, logger)
				},
			},
			{
				Name:    "retrieve",
				Aliases: []string{"r"},
				Usage:   "Retrieve data. Usage: retrieve <bucket_id> <object_id> <version_id>",
				Action: func(c *cli.Context) error {
					return retrieveCommand(c, db, cfg, logger)
				},
			},
		},
	}

	r := api.SetupRouter(db)

	go func() {
		if err := r.Run(cfg.ServerAddress); err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	}()

	if err := app.Run(os.Args); err != nil {
		logger.Fatal("CLI failed", zap.Error(err))
	}
}
