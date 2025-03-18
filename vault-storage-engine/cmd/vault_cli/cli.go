package vault_cli

import (
	"log"
	"os"

	"github.com/getvault-mvp/vault-base/pkg/bucket"
	"github.com/getvault-mvp/vault-base/pkg/config"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

// This function contains the full working CLI, from storage to retrieval and creating a new bucket instance
func RunCli() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := config.LoadConfig()

	db, err := bucket.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	app := &cli.App{
		Name:  "Vault",
		Usage: "Store and Retrieve Data With Vault Storage Engine",
		Commands: []*cli.Command{
			{
				Name:  "create-bucket",
				Usage: "Create an empty bucket. Usage: create-bucket <bukcet_id> <owner_id>",
				Action: func(c *cli.Context) error {
					return newBucketCommand(c, db, cfg, logger)
				},
			},
			{
				Name:  "store-object",
				Usage: "Store objects in valid buckets. Usage: store <bucket_id> <file_path>",
				Action: func(c *cli.Context) error {
					return storeCommand(c, db, cfg, logger)
				},
			},
			{
				Name:  "get-object",
				Usage: "Retrieves a valid object from it's bucket. Usage: retrieve <bucket_id> <object_id> <version_id>",
				Action: func(c *cli.Context) error {
					return RetrieveCommand(c, db, cfg, logger)
				},
			},
			{
				Name:  "read-metadata",
				Usage: "Returns metadata.json for objects",
				Action: func(ctx *cli.Context) error {
					return bucket.ReadMetadataJson("metadata.json")
				},
			},
			{
				Name: "list-buckets",
				Action: func(ctx *cli.Context) error {
					return ListAllBuckets()
				},
			},
		},
	}

	/* r := api.SetupRouter(db) */

	/* go func() {
		if err := r.Run(cfg.ServerAddress); err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	}()
	*/
	if err := app.Run(os.Args); err != nil {
		logger.Fatal("CLI failed", zap.Error(err))
	}
}
