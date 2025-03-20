package vault_cli

import (
	"database/sql"
	"fmt"

	"github.com/getvaultapp/vault-storage-engine/pkg/config"
	"github.com/getvaultapp/vault-storage-engine/pkg/datastorage"
	"github.com/getvaultapp/vault-storage-engine/pkg/sharding"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func deleteBucket(c *cli.Context, db *sql.DB, cfg *config.Config, logger *zap.Logger) error {
	if c.NArg() != 1 {
		return fmt.Errorf("usage: delete-bucket <bucket_id>")
	}

	bucketID := c.Args().Get(0)

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)

	err := datastorage.DeleteBucket(db, bucketID, store, logger)
	if err != nil {
		return fmt.Errorf("failed to delete bucket")
	}

	return nil
}
