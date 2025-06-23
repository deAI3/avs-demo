package operator

import (
	"avs/aggregator"
	"fmt"
	"net/rpc"
	"time"

	"github.com/pkg/errors"
)

func NewAggregatorRpcClient(aggregatorIpPortAddr string) (*AggregatorRpcClient, error) {
	client, err := rpc.DialHTTP("tcp", aggregatorIpPortAddr)
	if err != nil {
		return nil, err
	}

	return &AggregatorRpcClient{
		rpcClient:            client,
		aggregatorIpPortAddr: aggregatorIpPortAddr,
	}, nil
}

func (c *AggregatorRpcClient) SendSignedTaskResponseToAggregator(signedTaskResponse aggregator.SignedTaskResponse) {
	var reply uint8
	for retries := 0; retries < MaxRetries; retries++ {
		err := c.rpcClient.Call("Aggregator.RespondTask", signedTaskResponse, &reply)
		if err != nil {
			fmt.Println("Received error from aggregator", "err :", err)
			if errors.Is(err, rpc.ErrShutdown) {
				fmt.Println("Aggregator is shutdown. Reconnecting...")
				client, err := rpc.DialHTTP("tcp", c.aggregatorIpPortAddr)
				if err != nil {
					fmt.Println("Could not reconnect to aggregator", "err", err)
					time.Sleep(RetryInterval)
				} else {
					c.rpcClient = client
					fmt.Println("Reconnected to aggregator")
				}
			} else {
				fmt.Println("Received error from aggregator:", err, ". Retrying RespondTask RPC call...")
				time.Sleep(RetryInterval)
			}
		} else {
			fmt.Println("Signed task response header accepted by aggregator.", "reply", reply)
			return
		}
	}
}
