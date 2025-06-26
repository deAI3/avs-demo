package operator

import (
	"avs/types/proto/socket"
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	}()

	// respond tasks
	go func() {

		op.logger.Info("Respond tasks process started...")
		for {
			select {
			case task := <-op.TaskQueue:
				// send responses to server
				op.RespondTask(task)
			default:
				op.logger.Info("waiting for task")
			}

		}
	}()

	// fetch responses
	go func() {
		op.logger.Info("fetch responses process started...")
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
	}()

	// vote
	go func() {
		op.logger.Info("verify responses process started...")
		for {
			select {
			case resp := <-op.ResponseQueue:
				// send responses to server
				op.VerifyResponse(resp)
			default:
				op.logger.Info("waiting for response")
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

func (op *Operator) RespondTask(task Task) error {
	// TODO: handle task here

	// then send the response chunks here
	return op.TaskStream.Send()
}

func (op *Operator) VerifyResponse(resp *socket.TaskResponseMessage) error {
	// TODO: handle verify here

	// then send the vote here
	return op.VoteStream.Send()
}
