package main

import "github.com/getvaultapp/vault-storage-engine/cmd/vault_cli"

func main() {
	/* r := gin.Default()
	r.LoadHTMLFiles("index.html")
	r.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Welcome to Vault",
		})
	})

	r.Run(":8080") // This should listen and serve on this address */

	vault_cli.RunCli()
}
