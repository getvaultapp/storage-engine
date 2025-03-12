package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/getvault-mvp/vault-base/pkg/bucket"
	"github.com/getvault-mvp/vault-base/pkg/datastorage"
	"github.com/urfave/cli/v2"
)

func main() {
	db, err := bucket.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "store",
				Usage: "Store a file in a bucket",
				Action: func(c *cli.Context) error {
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
				},
			},
			{
				Name:  "retrieve",
				Usage: "Retrieve a file from a bucket",
				Action: func(c *cli.Context) error {
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
				},
			},
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
