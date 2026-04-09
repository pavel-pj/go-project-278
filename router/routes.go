package router

import (
	h "db200/handlers"
	"db200/internal/app"

	"github.com/gin-gonic/gin"
)

func registerRoutes(router *gin.Engine, app *app.App) {

	linkHandler := h.NewLinkHandler(app.Queries, app.Services.Links)
	//visitHandler := h.NewVisitHandler(app.Queries)

	router.GET("/ping", h.PingHandler)
	//router.POST("/api/links", linkHandler.Create)
	router.GET("/api/links", linkHandler.GetAllLinks)
	//router.GET("/api/links/:id", linkHandler.GetLink)
	//router.DELETE("/api/links/:id", linkHandler.DeleteLink)
	//router.PUT("/api/links/:id", linkHandler.UpdateLink)

	//router.GET("/r/:code", visitHandler.Redirect)
	//router.GET("/api/link_visits", visitHandler.GetVisits)

}
