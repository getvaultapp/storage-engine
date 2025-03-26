package object_cli

import (
	"database/sql"
	"fmt"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/datastorage"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/sharding"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func DeleteObject(c *cli.Context, db *sql.DB, cfg *config.Config, logger *zap.Logger) error {
	if c.NArg() != 2 {
		return fmt.Errorf("usage: delete-object <bucket_id> <object_id>")
	}

	bucketID := c.Args().Get(0)
	objectID := c.Args().Get(1)

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)
	err := datastorage.DeleteObject(db, bucketID, objectID, store, logger)
	if err != nil {
		return fmt.Errorf("failed to delete object")
	}

	fmt.Printf("Successfully deleted all versions of object %s\n", objectID)

	return nil
}
