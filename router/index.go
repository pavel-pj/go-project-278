package router

import (
	"db200/internal/app"
	"fmt"
	"os"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

// NewRouter creates and configures a new Gin router with all necessary middleware
// and routes. It initializes Sentry for error tracking if the SENTRY_DSN environment
// variable is set. The router is pre-configured with default Gin middleware and
// registered routes for the application.
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
