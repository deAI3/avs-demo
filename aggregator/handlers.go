package aggregator

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow any origin (for demo)
	},
}

func (agg *Aggregator) GetTask(c *gin.Context) {
	var payload GetTaskRequest

	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: fmt.Sprintf("Failed to parse request body: %s", err.Error()),
		})
		return
	}

	if taskInfo, ok := agg.PendingTasks[payload.TaskId]; ok {
		c.JSON(http.StatusOK, taskInfo)
	} else {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Message: "Task not found",
		})
	}
}

func (agg *Aggregator) GetModels(c *gin.Context) {
	models := []Model{}

	for _, op := range agg.CurrentOperators {
		models = append(models, Model{
			NodeAddress: op.Pubkey,
			Models:      op.Models,
		})
	}

	c.JSON(http.StatusOK, ModelsResponse{
		Status: "success",
		Models: models,
	})
}
