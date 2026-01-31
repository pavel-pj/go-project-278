package router

import (
	"fmt"
	"os"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	// Инициализация Sentry
	sentryDSN := os.Getenv("SENTRY_DSN")
	if sentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn: sentryDSN,
		}); err != nil {
			fmt.Printf("Sentry initialization failed: %v\n", err)
		}
	}

	router := gin.Default()

	// Middleware Sentry (только если DSN установлен)
	if sentryDSN != "" {
		router.Use(sentrygin.New(sentrygin.Options{
			Repanic: true,
		}))
	}

	// Регистрация роутов
	registerRoutes(router)

	return router
}

func registerRoutes(router *gin.Engine) {
	router.GET("/ping", pingHandler)
	router.POST("/api/links", linkCreateHandler)

}
