package aggregator

import (
	"sync"
	"time"

	"go.uber.org/zap"

	pb "avs/types/proto/aggregator"
	"avs/types/proto/socket"
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
	CurrentTaskId    uint64
	CurrentOperators map[string]*pb.Operator
	TaskMutex        sync.Mutex
	OperatorMutex    sync.Mutex
	FinishTasks      chan TaskInfo

	// stream
	TaskClients map[string]socket.TaskService_TaskStreamServer // map node address to stream connection
	VoteClients map[string]socket.TaskService_VoteStreamServer // map node address to stream connection

	// general channel
	respondsChan chan *socket.TaskResponseMessage
	voteChan     chan *socket.ResponseVoteMessage

	SeqMutex     sync.Mutex
	taskSequence uint64

	// default handler
	pb.UnimplementedOperatorServiceServer
	socket.UnimplementedTaskServiceServer
}

type TaskInfo struct {
	Id              uint64
	Proposer        string
	TaskConfig      TaskConfig
	Task            TaskPayload
	TaskCreatedTime time.Time
	ExpiredTime     time.Time
}

type TaskConfig struct {
	NumGenerators               uint64
	TokenThreshold              uint64
	InferenceStepExpirationTime time.Duration
}

type TaskPayload struct {
	State            TaskState
	Prompt           string
	Model            string
	CurrentTokens    []string
	AppliedOperators [][]byte // Public keys of operators who applied to the task]
	TempResult       *TempResultData
	FinalResult      *FinalResultData
}

type TempResultData struct {
	InferenceStep InferenceStep
}

type FinalResultData struct {
	Output           string
	TotalSteps       uint64
	WinningGenerator []byte // Public key of the winning generator
	CompletionTime   time.Time
}

type InferenceStep struct {
	// Tokens            []string
	// Finalized         bool
	Generators        [][]byte          // List of generators' public keys
	Validators        [][]byte          // List of validators' public keys
	GeneratorResults  map[uint64]string // Map of generator index to their results
	VerificationVotes []VerificationVote
	// ChoseGenerator    []byte // Public key of the chosen generator
}

type VerificationVote struct {
	Tokens         string
	OperatorPubkey []byte
	Signature      []byte
	TimeStamp      time.Time
}

type SignedTaskResponse struct {
	TaskId         uint64
	ApplyTask      *ApplyTaskResponse
	IsGenerator    bool
	GenerateAnswer *GenerateAnswerResponse
	VerifyAnswer   *VerifyAnswerResponse
	Pubkey         []byte
}

type ApplyTaskResponse struct {
	Signature []byte
}

type GenerateAnswerResponse struct {
	Signature []byte
	Response  string
}

type VerifyAnswerResponse struct {
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
	ID      uint64    `json:"id"`
	Object  string    `json:"object"`
	Created time.Time `json:"created"`
	Model   string    `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

type Model struct {
	NodeAddress string   `json:"node_address"`
	Models      []string `json:"models"`
}

type ModelsResponse struct {
	Status string  `json:"status"`
	Models []Model `json:"models"`
}
