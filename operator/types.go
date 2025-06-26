package operator

import (
	pb "avs/types/proto/aggregator"
	"avs/types/proto/socket"

	aptos "github.com/aptos-labs/aptos-go-sdk"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/Layr-Labs/eigensdk-go/crypto/bls"
)

type Config struct {
	CmcApi string `json:"cmc_api"`
}

type Operator struct {
	logger *zap.Logger

	BlsPrivateKey         []byte
	TaskStream            socket.TaskService_TaskStreamClient
	VoteStream            socket.TaskService_VoteStreamClient
	OperatorServiceClient pb.OperatorServiceClient
	AggRpcClient          AggregatorRpcClient
	// network      aptos.NetworkConfig
	TaskQueue     chan Task
	ResponseQueue chan *socket.TaskResponseMessage
}

type Task struct {
	Id   uint64
	Task map[string]interface{}
}

type AggregatorRpcClient struct {
	rpcClient            *grpc.ClientConn
	aggregatorIpPortAddr string
}

type OperatorConfig struct {
	BlsPrivateKey []byte
	// AvsAddress           string
	AggregatorIpPortAddr string
	// OperatorId           eigentypes.OperatorId
}

type AVSTask struct {
	// TODO
	task_created_timestamp uint64
	responded              bool
	respond_fee_token      uint64
	respond_fee_limit      uint64
}

type BlsConfig struct {
	KeyPair *bls.KeyPair
}

type MetadataStr struct {
	Inner string
}

type Metadata struct {
	Inner aptos.AccountAddress
}
