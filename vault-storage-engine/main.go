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

	err := r.Run("0.0.0.0:5000") // This should listen and serve on this address
	if err != nil {
		panic("Failed to start server: " + err.Error())
	}
	*/
	vault_cli.RunCli()
}
