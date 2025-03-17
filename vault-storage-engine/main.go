package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/getvault-mvp/vault-base/pkg/bucket"
	"github.com/getvault-mvp/vault-base/pkg/config"
	"github.com/getvault-mvp/vault-base/pkg/datastorage"
	"github.com/getvault-mvp/vault-base/pkg/sharding"
	"github.com/getvault-mvp/vault-base/pkg/utils"
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
	locations := []string{
		"/mnt/disk1/shards",
		"/mnt/disk2/shards",
		"/mnt/disk3/shards",
		"/mnt/disk4/shards",
		"/mnt/disk5/shards",
		"/mnt/disk6/shards",
		"/mnt/disk7/shards",
		"/mnt/disk8/shards",
	}
	objectID := uuid.New().String() // Generate a unique object ID

	// Shard and store data
	versionID, shardLocations, proofs, err := datastorage.StoreData(db, data, bucketID, objectID, filepath.Base(filePath), store, cfg, locations, logger)
	if err != nil {
		return fmt.Errorf("store failed: %w", err)
	}

	// Convert proofs from []string to map[string]string
	proofsMap := utils.ConvertSliceToMap(proofs)

	// Validate that shardLocations and proofsMap are correctly populated
	if len(shardLocations) == 0 || len(proofsMap) == 0 {
		return fmt.Errorf("store failed: shardLocations or proofsMap cannot be empty")
	}

	/* // Save object metadata in SQLite
	metadata := bucket.VersionMetadata{
		ShardLocations: shardLocations,
		Proofs:         proofsMap,
	}

	root_version, _ := bucket.GetRootVersion(db, objectID) */

	owner := "default_owner" // Replace with actual owner if available
	err = bucket.CreateBucket(db, bucketID, owner)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	/* err = bucket.AddVersion(db, bucketID, objectID, versionID, root_version, metadata, data)
	if err != nil {
		return fmt.Errorf("store failed: %w", err)
	} */

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
	data, filename, err := datastorage.RetrieveData(db, bucketID, objectID, versionID, store, cfg, logger)
	if err != nil {
		return fmt.Errorf("retrieve failed: %w", err)
	}

	// Write the retrieved data to a file with the original filename
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write retrieved data to file: %w", err)
	}

	fmt.Printf("Retrieved data and stored it in file: %s\n", filename)
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
		Name:  "Vault",
		Usage: "Store and Retrieve Data With Vault Storage Engine",
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

	/* r := api.SetupRouter(db) */

	/* go func() {
		if err := r.Run(cfg.ServerAddress); err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	}()
	*/
	if err := app.Run(os.Args); err != nil {
		logger.Fatal("CLI failed", zap.Error(err))
	}
}
