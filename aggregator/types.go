package aggregator

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type TaskState uint8

const (
	WaitingForApply TaskState = 1
	TempResult      TaskState = 2
	Finalize        TaskState = 3
	Unknown         TaskState = 0
)

type AggregatorConfig struct {
	ServerIpPortAddress string
	StorePath           string
}

type TaskContext struct {
	TaskMutex sync.Mutex
	Task      TaskInfo
}

type Aggregator struct {
	logger *zap.Logger

	AggregatorConfig AggregatorConfig
	TaskQueue        chan TaskInfo
	PendingTasks     map[uint64]TaskInfo
	CurrentOperators map[string]Operator
	TaskMutex        sync.Mutex
	OperatorMutex    sync.Mutex

	// socket
	TaskClients map[string]*websocket.Conn // map node address to socket connection
	VoteClients map[string]*websocket.Conn // map node address to socket connection

	broadcastChan chan TaskInfo
	responsesRecv chan TaskInfo

	taskSequence uint64
}

type TaskInfo struct {
	Id              uint64
	TaskConfig      TaskConfig
	Task            TaskPayload
	TaskCreatedTime time.Time
}

type Operator struct {
	Pubkey []byte
	Stake  uint64
	Models []string
}

type TaskConfig struct {
	NumGenerators               uint64
	TokenThreshold              uint64
	InferenceStepExpirationTime time.Duration
}

type TaskPayload struct {
	State         TaskState
	Prompt        string
	CurrentTokens []string
	TempResult    *TempResultData
	FinalResult   *FinalResultData
}

type TempResultData struct {
	CurrentStep            uint64
	Steps                  []InferenceStep
	NextStepExpirationTime time.Time
}

type FinalResultData struct {
	Output           string
	TotalSteps       uint64
	WinningGenerator []byte // Public key of the winning generator
	CompletionTime   time.Time
}

type InferenceStep struct {
	Step              uint64
	Tokens            []string
	Finalized         bool
	Generators        [][]byte // List of generators' public keys
	Validators        [][]byte // List of validators' public keys
	GeneratorResults  [][]string
	VerificationVotes []VerificationVote
	ChoseGenerator    []byte // Public key of the chosen generator
}

type VerificationVote struct {
	Tokens         []string
	OperatorPubkey []byte
	Signature      []byte
	TimeStamp      time.Time
}

// chat completions api
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
