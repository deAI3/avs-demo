package operator

import (
	"github.com/gin-gonic/gin"
)

func (op *Operator) SetupRoutes() *gin.Engine {
	r := gin.Default()

	r.POST("/api/v1/chat/completions", op.ChatCompletions)
	// r.GET("/api/v1/models", GetModels)

	return r
}
