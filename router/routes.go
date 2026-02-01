package router

import (
	h "db200/handlers"
	"db200/internal/app"

	"github.com/gin-gonic/gin"
)

func registerRoutes(router *gin.Engine, app *app.App) {

	linkHandler := h.NewLinkHandler(app.Services.Links)

	router.GET("/ping", h.PingHandler)
	router.POST("/api/links", linkHandler.Create)
	router.GET("/api/links", linkHandler.GetAllLinks)

}
