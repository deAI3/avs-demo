package operator

import (
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk/crypto"
	"go.uber.org/zap"
)

func NewOperator(logger *zap.Logger, config OperatorConfig, blsPriv []byte) (*Operator, error) {
	// quorumCount := QuorumCount(client)
	// if quorumCount == 0 {
	// 	panic("No quorum found, please initialize quorum first ")
	// }

	// quorumNumbers := quorumCount
	// fmt.Println("quorumNumbers:", quorumNumbers)

	// Get OperatorId
	var privKey crypto.BlsPrivateKey
	privKey.FromBytes(config.BlsPrivateKey)

	aggClient, err := NewAggregatorRpcClient(config.AggregatorIpPortAddr)
	if err != nil {
		return nil, fmt.Errorf("can not create new aggregator Rpc Client: %v", err)
	}

	// return Operator
	operator := Operator{
		logger:        logger,
		BlsPrivateKey: blsPriv,
		AggRpcClient:  *aggClient,
		TaskQueue:     make(chan Task, 100),
	}
	return &operator, nil
}
