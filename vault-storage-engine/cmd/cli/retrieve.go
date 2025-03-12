package main

import (
	"fmt"

	"database/sql"

	"github.com/getvault-mvp/vault-base/pkg/config"
	"github.com/getvault-mvp/vault-base/pkg/datastorage"
	"github.com/getvault-mvp/vault-base/pkg/sharding"
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
	data, err := datastorage.RetrieveData(db, bucketID, objectID, versionID, store, cfg, logger)
	if err != nil {
		return fmt.Errorf("retrieve failed: %w", err)
	}

	fmt.Printf("Retrieved data: %s\n", string(data))
	return nil
}
