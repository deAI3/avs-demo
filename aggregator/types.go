package aggregator

import (
	"sync"
	"time"

	// aptos "github.com/aptos-labs/aptos-go-sdk"
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
	// AvsAddress          string
	// AccountConfig AccountConfig
}

// type AccountConfig struct {
// 	AccountPath string // store path
// 	Profile     string
// }

type Aggregator struct {
	logger *zap.Logger
	// AvsAddress        string
	// AggregatorAccount aptos.Account
	AggregatorConfig AggregatorConfig
	TaskQueue        chan TaskInfo
	PendingTasks     []TaskInfo
	CurrentTaskId    uint64
	CurrentOperators []Operator
	TaskMutex        sync.Mutex
	OperatorMutex    sync.Mutex
	// Network           aptos.NetworkConfig
}

type TaskInfo struct {
	Id              uint64
	TaskConfig      TaskConfig
	Task            TaskPayload
	TaskCreatedTime time.Time
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

// type U128Struct struct {
// 	Value *big.Int `json:"value"`
// }

// func (u *U128Struct) MarshalBCS(ser *bcs.Serializer) {
// 	ser.U128(*u.Value)
// }

// type BytesStruct struct {
// 	Value []byte
// }

// func (b *BytesStruct) MarshalBCS(ser *bcs.Serializer) {
// 	ser.WriteBytes(b.Value)
// }

// type U8Struct struct {
// 	Value uint8
// }

// func (u *U8Struct) MarshalBCS(ser *bcs.Serializer) {
// 	ser.U8(u.Value)
// }

// type VecAddr struct {
// 	Value []aptos.AccountAddress
// }

// func (v *VecAddr) MarshalBCS(ser *bcs.Serializer) {
// 	bcs.SerializeSequence(v.Value, ser)
// }

// type Addr struct {
// 	Value aptos.AccountAddress
// }

// func (v *Addr) MarshalBCS(ser *bcs.Serializer) {
// 	v.Value.MarshalBCS(ser)
// }
