package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// ====== YOUR FIRST ROUTE ======
	// GET request to "/"
	router.GET("/", func(c *gin.Context) {
		// Send JSON response
		c.JSON(http.StatusOK, gin.H{
			"message": "Hello, Gin!",
			"status":  "success",
		})
	})

	// Run server (blocks until stopped)
	err := router.Run(":8080")
	if err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
