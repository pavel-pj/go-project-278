// main go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

func main() {
	router := setupRouter()
	router = pingHandler(router)
	router = pingErr1(router)

	// Run server (blocks until stopped)
	err := router.Run(":8080")
	if err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func setupRouter() *gin.Engine {
	sentryDSN := os.Getenv("SENTRY_DSN")
	// "https://df657a4b3d27b8a40709ced96e15c4a8@o4510775359832064.ingest.us.sentry.io/4510775423270912",
	// To initialize Sentry's handler, you need to initialize Sentry itself beforehand
	if err := sentry.Init(sentry.ClientOptions{
		Dsn: sentryDSN,
	}); err != nil {
		fmt.Printf("Sentry initialization failed: %v\n", err)
	}

	router := gin.Default()
	router.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))

	return router
}

func pingHandler(router *gin.Engine) *gin.Engine {
	router.GET("/ping", func(c *gin.Context) {
		//panic("Тестовая паника для Sentry!")
		//c.JSON(500, gin.H{"error": "Что-то сломалось"})
		//sentry.CaptureMessage("ERROR")
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	return router
}

func pingErr1(router *gin.Engine) *gin.Engine {
	router.GET("/err1", func(c *gin.Context) {

		//c.JSON(500, gin.H{"error": "Что-то сломалось"})
		sentry.CaptureMessage("It works!")
		/*c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})*/
	})
	return router
}
