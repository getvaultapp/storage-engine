package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HomeHandler to test if server is running
func HomeHandler(c *gin.Context) {
	log.Printf("Vault")
	c.JSON(http.StatusBadRequest, gin.H{"message": "Server running"})
}
