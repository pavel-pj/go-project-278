package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func linkCreateHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "CreateLink",
	})
}
