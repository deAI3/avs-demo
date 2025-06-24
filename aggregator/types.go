package aggregator

import (
	"log"
	"sync"
)

type AggregatorConfig struct {
	ServerIpPortAddress string
	StorePath           string
}

type Aggregator struct {
	logger *log.Logger

	AggregatorConfig AggregatorConfig
	TaskQueue        chan Task
	PendingTasks     map[uint64]TaskInfo
	CurrentOperators []Operator
	TaskMutex        sync.Mutex
	OperatorMutex    sync.Mutex
}

type TaskInfo struct {
	State     map[string]interface{}
	Responses map[uint64][]SignedTaskResponse
}

type Task struct {
	Id   uint64
	Task map[string]interface{}
}

type SignedTaskResponse struct {
	TaskId    uint64
	Pubkey    []byte
	Signature []byte
	Response  []string
}

type Operator struct {
	Pubkey []byte
	Stake  uint64
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
}
