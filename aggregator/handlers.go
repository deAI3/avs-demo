package aggregator

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow any origin (for demo)
	},
}

func (agg *Aggregator) ChatCompletions(c *gin.Context) {

}

func (agg *Aggregator) GetModels(c *gin.Context) {

}
