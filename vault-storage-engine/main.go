package main

import vault_cli "github.com/getvaultapp/vault-storage-engine/run_cli/cli_cmd"

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
