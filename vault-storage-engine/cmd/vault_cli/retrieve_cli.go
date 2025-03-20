package vault_cli

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/getvaultapp/vault-storage-engine/pkg/config"
	"github.com/getvaultapp/vault-storage-engine/pkg/datastorage"
	"github.com/getvaultapp/vault-storage-engine/pkg/sharding"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

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
