package router

import (
	"db200/internal/app"
	"fmt"
	"os"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Инициализация Роутера
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

	// Если Render не за Cloudflare — заголовка нет, Gin продолжит брать X-Forwarded-For.
	router.TrustedPlatform = gin.PlatformCloudflare

	// CORS (до объявления маршрутов)
	config := cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Range"},
		AllowCredentials: true,
	}

	router.Use(cors.New(config))

	// Middleware Sentry (только если DSN установлен)
	if sentryDSN != "" {
		router.Use(sentrygin.New(sentrygin.Options{
			Repanic: true,
		}))
	}

	router.Use(func(c *gin.Context) {
		c.Next()

		// Логируем заголовки ответа
		fmt.Println("Response headers:")
		for key, values := range c.Writer.Header() {
			fmt.Printf("  %s: %v\n", key, values)
		}
	})

	// Регистрация роутов
	registerRoutes(router, app)

	return router
}
