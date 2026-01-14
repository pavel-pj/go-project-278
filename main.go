// main go
package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := setupRouter()
	router = pingHandler(router)

	// Run server (blocks until stopped)
	err := router.Run(":8080")
	if err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func setupRouter() *gin.Engine {
	router := gin.Default()
	return router
}

func pingHandler(router *gin.Engine) *gin.Engine {
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	return router
}
