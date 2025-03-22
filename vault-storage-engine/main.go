package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

//import vault_cli "github.com/getvaultapp/storage-engine/vault-storage-engine/run_cli/cli_cmd"

func main() {
	r := gin.Default()
	r.LoadHTMLFiles("index.html")
	r.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Welcome to Vault",
		})
	})

	err := r.Run(":8080") // This should listen and serve on this address
	if err != nil {
		panic("Failed to start server: " + err.Error())
	}
	//vault_cli.RunCli()
}
