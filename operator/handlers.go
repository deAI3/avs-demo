package operator

import (
	"avs/aggregator"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

func (op *Operator) ChatCompletions(c *gin.Context) {
	var payload aggregator.ChatCompletionRequest

	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, aggregator.ErrorResponse{
			Message: fmt.Sprintf("Failed to parse request body: %s", err.Error()),
		})
		return
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize payload"})
		return
	}

	// Make the POST request to the external Chat API
	req, err := http.NewRequest("POST", op.ModelApi+"/v1/chat/completions", bytes.NewBuffer(payloadBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("Authorization", "Bearer token-abc123")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to contact Chat API"})
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response body"})
		return
	}
	var parsed map[string]interface{}
	json.Unmarshal(body, &parsed)

	content := ""
	if choices, ok := parsed["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				content, _ = msg["content"].(string)
			}
		}
	}

	c.JSON(http.StatusOK, aggregator.ChatCompletionResponse{
		ID:      0,
		Object:  "chat.completion",
		Created: time.Now(),
		Model:   payload.Model,
		Content: content,
	})

}
