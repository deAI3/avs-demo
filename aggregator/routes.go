package aggregator

import (
	"github.com/gin-gonic/gin"
)

func (agg *Aggregator) SetupRoutes() *gin.Engine {
	r := gin.Default()

	r.POST("/api/v1/chat/completions", agg.ChatCompletions)
	r.GET("/api/v1/models", agg.GetModels)

	return r
}
