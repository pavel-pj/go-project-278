package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func pingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

/*
func pingHandler23(router *gin.Engine) {
	router.GET("/ping", func(c *gin.Context) {
		//panic("Тестовая паника для Sentry!")
		//c.JSON(500, gin.H{"error": "Что-то сломалось"})
		//sentry.CaptureMessage("ERROR")
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

}
*/
