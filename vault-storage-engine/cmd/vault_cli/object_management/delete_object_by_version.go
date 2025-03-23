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

func DeleteObjectByVersion(c *cli.Context, db *sql.DB, cfg *config.Config, logger *zap.Logger) error {
	if c.NArg() != 3 {
		return fmt.Errorf("usage: delete-object-version <bucket_id> <object_id> <object_version_id>")
	}

	bucketID := c.Args().Get(0)
	objectID := c.Args().Get(1)
	versionID := c.Args().Get(2)

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)
	err := datastorage.DeleteObjectByVersion(db, bucketID, objectID, versionID, store, logger)
	if err != nil {
		return fmt.Errorf("failed to delete object")
	}

	fmt.Printf("Successfully deleted object %s version (%s)\n", objectID, versionID)

	return nil
}
