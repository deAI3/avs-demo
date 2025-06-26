package operator

import (
	"avs/aggregator"
	"avs/types/proto/socket"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aptos-labs/aptos-go-sdk/crypto"
	"go.uber.org/zap"
)

const (
	MaxRetries                        = 10
	RetryInterval                     = 2 * time.Second
	BlockInterval              uint64 = 1000
	PollLatestBatchInterval           = 5 * time.Second
	RemoveBatchFromSetInterval        = 5 * time.Minute
)

func (op *Operator) Start(ctx context.Context) error {
	op.logger.Info("Starting operator...")

	ctx, cancel := context.WithCancel(ctx)

	defer cancel()
	// Fetching tasks
	go func() {
		op.logger.Info("Fetching tasks process started...")
		op.FetchTasks()
		// get tasks from channel
	}()

	go func() {
		op.logger.Info("Respond tasks process started...")
		for {
			select {
			case task := <-op.TaskQueue:
				// send responds to server
				op.RespondTask(task)
			default:
				op.logger.Info("waiting for task")
			}

		}
	}()

	go func() {
		op.logger.Info("fetch responds process started...")
		op.FetchTasks()
	}()

	go func() {
		op.logger.Info("verify responds process started...")
		for {
			select {
			case resp := <-op.ResponseQueue:
				// send responds to server
				op.VerifyResponse(resp)
			default:
				op.logger.Info("waiting for task")
			}

		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a signal to shutdown
	sig := <-sigChan
	op.logger.Info("Received signal, shutting down...", zap.Any("signal", sig))

	cancel()
	return nil
}

func (op *Operator) FetchTasks() {
	// Receive tasks
	for {
		task, err := op.TaskStream.Recv()
		if err == io.EOF {
			log.Println("Stream closed by server")
		}
		if err != nil {
			log.Fatalf("Stream recv error: %v", err)

		}

		op.TaskQueue <- Task{
			Id:   task.TaskId,
			Task: make(map[string]interface{}),
		}
	}
}

// TODO: update here
func (op *Operator) RespondTask(task Task) error {
	denom := task.Task["data_request"].(string)
	upperDenom := strings.ToUpper(denom)
	taskId := task.Id

	price := big.NewInt(int64(getCMCPrice(upperDenom) * 1000000))

	msghHash, err := GetMsgHash(client, op.avsAddress, taskId, *price)
	if err != nil {
		return fmt.Errorf("failed to GetMsgHash: %v", err)
	}

	trimmedMsgHash := strings.TrimPrefix(msghHash, "0x")
	bytesMsgHash, err := hex.DecodeString(trimmedMsgHash)
	if err != nil {
		return fmt.Errorf("failed to decode hex to string: %v", err)
	}

	var priv crypto.BlsPrivateKey
	err = priv.FromBytes(op.BlsPrivateKey)
	if err != nil {
		panic("Failed to create bls priv key" + err.Error())
	}
	signature, err := priv.Sign(bytesMsgHash)
	if err != nil {
		panic("Failed to create signature" + err.Error())
	}

	// pubKey, err := priv.GeneratePubkey()
	// if err != nil {
	// 	panic("Failed to generate pubkey from privkey" + err.Error())
	// }

	op.AggRpcClient.SendSignedTaskResponseToAggregator(aggregator.SignedTaskResponse{
		TaskId:    taskId,
		Pubkey:    signature.Auth.PublicKey().Bytes(),
		Signature: signature.Auth.Signature().Bytes(),
		Response:  price,
	})
	return nil
}

func (op *Operator) FetchResponse() {
	// Receive resp
	for {
		resp, err := op.VoteStream.Recv()
		if err == io.EOF {
			log.Println("Stream closed by server")
		}
		if err != nil {
			log.Fatalf("Stream recv error: %v", err)

		}

		op.ResponseQueue <- resp
	}
}

func (op *Operator) VerifyResponse(resp *socket.TaskResponseMessage) {

}
