package vault_cli

import (
	"database/sql"
	"fmt"
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

func StoreCommand(c *cli.Context, db *sql.DB, cfg *config.Config, logger *zap.Logger) error {
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

	// Save object metadata in SQLite
	metadata := bucket.VersionMetadata{
		ShardLocations: shardLocations,
		Proofs:         proofsMap,
	}
	err = bucket.AddVersion(db, bucketID, objectID, versionID, metadata, data)
	if err != nil {
		return fmt.Errorf("store failed: %w", err)
	}

	fmt.Printf("Stored file as version %s in bucket %s\n", versionID, bucketID)
	return nil
}
