package main

import (
	"fmt"

	"github.com/getvault-mvp/vault-base/pkg/datastorage"
	"github.com/urfave/cli/v2"
)

func retrieveCommand(c *cli.Context) error {
	if c.NArg() < 3 {
		return fmt.Errorf("usage: retrieve <bucket_id> <object_id> <version_id>")
	}

	bucketID := c.Args().Get(0)
	objectID := c.Args().Get(1)
	versionID := c.Args().Get(2)

	data, err := datastorage.RetrieveData(db, bucketID, objectID, versionID, store, cfg, logger)
	if err != nil {
		return fmt.Errorf("retrieve failed: %w", err)
	}

	fmt.Printf("Retrieved file from bucket %s, object %s, version %s\n", bucketID, objectID, versionID)
	return nil
}
