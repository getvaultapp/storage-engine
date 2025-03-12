package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/getvault-mvp/vault-base/pkg/datastorage"
	"github.com/urfave/cli/v2"
)

func storeCommand(c *cli.Context) error {
	if c.NArg() < 3 {
		return fmt.Errorf("usage: store <bucket_id> <file_path>")
	}

	bucketID := c.Args().Get(0)
	filePath := c.Args().Get(1)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	versionID, err := datastorage.StoreData(db, data, bucketID, filepath.Base(filePath), store, cfg, logger)
	if err != nil {
		return fmt.Errorf("store failed: %w", err)
	}

	fmt.Printf("Stored file as version %s in bucket %s\n", versionID, bucketID)
	return nil
}
