package aggregator

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow any origin (for demo)
	},
}

func (agg *Aggregator) ChatCompletions(c *gin.Context) {
	var payload ChatCompletionRequest

	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: fmt.Sprintf("Failed to parse request body: %s", err.Error()),
		})
		return
	}

	// update task sequence
	agg.SeqMutex.Lock()
	seq := agg.taskSequence
	agg.taskSequence += 1
	agg.SeqMutex.Unlock()

	createdTime := time.Now()
	expiredTime := createdTime.Add(time.Minute * 2)

	// insert task to queue
	agg.TaskMutex.Lock()

	taskInfo := TaskInfo{
		Id:         seq,
		Proposer:   agg.PickProposer(seq),
		TaskConfig: TaskConfig{},
		Task: TaskPayload{
			State:         nil,
			Prompt:        payload.Messages[0].Content,
			Model:         payload.Model,
			CurrentTokens: []string{},
			TempResult:    &TempResultData{},
			FinalResult:   &FinalResultData{},
		},
		TaskCreatedTime: createdTime,
		ExpiredTime:     expiredTime,
	}

	agg.TaskQueue <- taskInfo
	agg.TaskMutex.Unlock()

	// wait for the result
	for {
		// if timeout add to pending and retry later on
		if time.Now().After(expiredTime) {
			agg.PendingTasks[seq] = taskInfo
			c.JSON(http.StatusRequestTimeout, ErrorResponse{
				Message: "request expired, our operators might need more time for this request",
			})
			return
		}

		select {
		case resp := <-agg.FinishTasks:
			if resp.Id == seq {
				c.JSON(http.StatusOK, ChatCompletionResponse{
					ID:      seq,
					Object:  "chat.completion",
					Created: time.Now(),
					Model:   payload.Model,
					Choices: []struct {
						Index        int         `json:"index"`
						Message      ChatMessage `json:"message"`
						FinishReason string      `json:"finish_reason"`
					}{},
				})
			}
			return
		default:
			agg.logger.Info("Waiting for result...")
		}
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
