package operator

import (
	"context"
	"fmt"
	"net/rpc"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	pb "avs/types/proto/aggregator"
	"avs/types/proto/socket"
)

func NewAggregatorRpcClient(aggregatorIpPortAddr string) (*AggregatorRpcClient, error) {
	client, err := grpc.NewClient(aggregatorIpPortAddr)
	if err != nil {
		return nil, err
	}

	return &AggregatorRpcClient{
		rpcClient:            client,
		aggregatorIpPortAddr: aggregatorIpPortAddr,
	}, nil
}

func (op *Operator) SendRegisterOperatorRequest(request *pb.Operator) {
	for retries := 0; retries < MaxRetries; retries++ {
		ctx := context.Background()
		operatorServiceClient := pb.NewOperatorServiceClient(op.AggRpcClient.rpcClient)
		resp, err := operatorServiceClient.RegisterOperator(ctx, &pb.RegisterRequest{})
		if err != nil {
			fmt.Println("Received error from aggregator", "err :", err)
			if errors.Is(err, rpc.ErrShutdown) {
				fmt.Println("Aggregator is shutdown. Reconnecting...")
				client, err := grpc.NewClient(op.AggRpcClient.aggregatorIpPortAddr)
				if err != nil {
					fmt.Println("Could not reconnect to aggregator", "err", err)
					time.Sleep(RetryInterval)
				} else {
					op.AggRpcClient.rpcClient = client
					fmt.Println("Reconnected to aggregator")
				}
			} else {
				fmt.Println("Received error from aggregator:", err, ". Retrying RegisterOperator RPC call...")
				time.Sleep(RetryInterval)
			}
		} else {
			// if success boostrap stream channels
			taskServiceClient := socket.NewTaskServiceClient(op.AggRpcClient.rpcClient)
			taskstream, err := taskServiceClient.TaskStream(context.Background())
			if err != nil {
				op.logger.Error(fmt.Sprintf("can not create new tasks stream to aggregator server: %v", err))
			}

			votestream, err := taskServiceClient.VoteStream(context.Background())
			if err != nil {
				op.logger.Error(fmt.Sprintf("can not create new votes stream to aggregator server: %v", err))
			}

			op.TaskStream = taskstream
			op.VoteStream = votestream
			fmt.Println("Register operator request accepted by aggregator.", "reply", resp.Respond)
			return
		}
	}
}

func (op *Operator) SendDeregisterOperatorRequest(request []byte) {
	for retries := 0; retries < MaxRetries; retries++ {
		ctx := context.Background()
		operatorServiceClient := pb.NewOperatorServiceClient(op.AggRpcClient.rpcClient)
		resp, err := operatorServiceClient.DeregisterOperator(ctx, &pb.DeregisterRequest{
			Pubkey: string(request),
		})
		if err != nil {
			fmt.Println("Received error from aggregator", "err :", err)
			if errors.Is(err, rpc.ErrShutdown) {
				fmt.Println("Aggregator is shutdown. Reconnecting...")
				client, err := grpc.NewClient(op.AggRpcClient.aggregatorIpPortAddr)
				if err != nil {
					fmt.Println("Could not reconnect to aggregator", "err", err)
					time.Sleep(RetryInterval)
				} else {
					op.AggRpcClient.rpcClient = client
					fmt.Println("Reconnected to aggregator")
				}
			} else {
				fmt.Println("Received error from aggregator:", err, ". Retrying DeregisterOperator RPC call...")
				time.Sleep(RetryInterval)
			}
		} else {
			// remove stream channels
			if op.TaskStream != nil {
				op.TaskStream.CloseSend()
			}

			if op.VoteStream != nil {
				op.VoteStream.CloseSend()
			}
			fmt.Println("Deregister operator request accepted by aggregator.", "reply", resp.Respond)
			return
		}
	}
}
