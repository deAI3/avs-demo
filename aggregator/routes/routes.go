package routes

import (
	handlers "avs/aggregator/routes/v1"

	"github.com/gin-gonic/gin"
)

func SetupRoutes() *gin.Engine {
	r := gin.Default()

	r.POST("/api/v1/chat/completions", handlers.ChatCompletions)
	r.GET("/api/v1/models", handlers.GetModels)

	return r
}
