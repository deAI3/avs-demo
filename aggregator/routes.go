package aggregator

import (
	"github.com/gin-gonic/gin"
)

func (agg *Aggregator) SetupRoutes() *gin.Engine {
	r := gin.Default()

	// r.POST("/api/v1/chat/completions", op.ChatCompletions)
	r.GET("/api/v1/models", agg.GetModels)
	r.GET("/api/v1/task", agg.GetTask)

	return r
}
