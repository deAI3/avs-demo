package operator

import (
	"avs/types/proto/socket"
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk/crypto"
	"go.uber.org/zap"
)

func NewOperator(logger *zap.Logger, config OperatorConfig, blsPriv []byte) (*Operator, error) {
	// Get OperatorId
	var privKey crypto.BlsPrivateKey
	privKey.FromBytes(config.BlsPrivateKey)

	aggClient, err := NewAggregatorRpcClient(config.AggregatorIpPortAddr)
	if err != nil {
		return nil, fmt.Errorf("can not create new aggregator Rpc Client: %v", err)
	}

	// TODO:register operator if not exist

	// return Operator
	operator := Operator{
		logger:        logger,
		BlsPrivateKey: blsPriv,
		AggRpcClient:  *aggClient,
		ModelApi:      config.ModelApi,
		TaskQueue:     make(chan Task, 100),
		ResponseQueue: make(chan *socket.TaskResponseMessage, 100),
	}

	return &operator, nil
}
