package router

import (
	"db200/internal/app"
	"fmt"
	"os"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

func NewRouter(app *app.App) *gin.Engine {
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
	registerRoutes(router, app)

	return router
}
