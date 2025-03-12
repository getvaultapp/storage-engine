package main

import (
	"log"
	"os"

	"github.com/getvault-mvp/vault-base/pkg/api"
	"github.com/getvault-mvp/vault-base/pkg/bucket"
	"github.com/getvault-mvp/vault-base/pkg/config"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := config.LoadConfig()

	db, err := bucket.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "store",
				Aliases: []string{"s"},
				Usage:   "Store data. Usage: store <filename_or_directory> <storage-location-configuration>",
				Action:  storeCommand,
			},
			{
				Name:    "retrieve",
				Aliases: []string{"r"},
				Usage:   "Retrieve Data From Metadata File. Usage: retrieve <metadatafile>",
				Action:  retrieveCommand,
			},
		},
	}

	r := api.SetupRouter(db)

	go func() {
		if err := r.Run(cfg.ServerAddress); err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	}()

	if err := app.Run(os.Args); err != nil {
		logger.Fatal("CLI failed", zap.Error(err))
	}
}
