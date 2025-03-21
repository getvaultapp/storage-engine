package object_cli

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/getvaultapp/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/vault-storage-engine/pkg/config"
	"github.com/getvaultapp/vault-storage-engine/pkg/datastorage"
	"github.com/getvaultapp/vault-storage-engine/pkg/sharding"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func UpdateByVersion(c *cli.Context, db *sql.DB, cfg *config.Config, logger *zap.Logger) error {
	if c.NArg() != 3 {
		return fmt.Errorf("usage: update-object <bucket_id> <object_id> <filename>")
	}

	bucketID := c.Args().Get(0)
	objectID := c.Args().Get(1)
	originalFile := c.Args().Get(2)

	getfile, versionID, err := bucket.UpdateFileVersionIfItExists(db, originalFile, bucketID, objectID)
	if err != nil {
		return fmt.Errorf("failed to check if the %s exists", originalFile)
	}

	// set the versionID
	version := versionID

	// if the filename gotten from the metadata correlates with the user-provided filename
	// go ahead and store it accorfingly
	if originalFile == getfile {
		fmt.Println("Object Exists. Updating ...")
		data, err := os.ReadFile(originalFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		// Setup a storage component for handling shards
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

		// make use of the predefined versionID returned by UpdateFileVersionIfItExists
		_, _, _, err = datastorage.StoreDataWithVersion(db, data, bucketID, objectID, version, filepath.Base(originalFile), store, cfg, locations, logger)
		if err != nil {
			return fmt.Errorf("failed to store updated object, %w", err)
		}
	} else {
		return fmt.Errorf("object does not have filename, %s: %w", originalFile, err)
	}
	return nil
}
