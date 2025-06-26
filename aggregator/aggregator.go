package aggregator

import (
	"avs/types/proto/socket"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

const taskQueueSize = 100

func NewAggregator(aggregatorConfig AggregatorConfig, logger *zap.Logger) (*Aggregator, error) {
	agg := Aggregator{
		logger: logger,

		AggregatorConfig: aggregatorConfig,
		TaskQueue:        make(chan TaskInfo, taskQueueSize),
		PendingTasks:     make(map[uint64]TaskInfo),
		taskSequence:     0,
		FinishTasks:      make(chan TaskInfo, 100),
	}
	return &agg, nil
}

func (agg *Aggregator) Start(ctx context.Context) error {
	agg.logger.Info("Starting aggregator...")

	ctx, cancel := context.WithCancel(ctx)

	defer cancel()
	go func() {
		err := agg.ServeOperators()
		if err != nil {
			agg.logger.Fatal("Error starting RPC server", zap.Any("err", err))
		}
	}()

	// start api server
	go func() {
		agg.logger.Info("api server process started...")

		router := agg.SetupRoutes()

		log.Println("Server running on http://localhost:8080")
		err := router.Run(":8080")
		if err != nil {
			agg.logger.Fatal("Error when starting api server", zap.Any("err", err))
		}
	}()

	// handle responds from nodes through the following steps:
	//
	// - emit event includes the repsonds to verifiers
	go func() {
		for {
			select {
			case resp := <-agg.respondsChan:
				for addr, stream := range agg.VoteClients {
					if addr == resp.NodeAddress {
						continue
					}

					// sending resps to all verifiers
					stream.Send(resp)
				}
			}

		}
	}()

	// handle votes
	// update task
	go func() {
		for {
			select {
			case vote := <-agg.voteChan:

			}

		}
	}()

	// distribute tasks to nodes through the following steps:
	//
	// - choose a propser node
	// - emit event includes the task and propose node address
	go func() {
		agg.logger.Info("Distribute tasks process started...")
		for {
			select {
			case msg := <-agg.TaskQueue:
				stream := agg.TaskClients[msg.Proposer]
				if err := stream.Send(&socket.TaskMessage{
					NodeAddress: msg.Proposer,
					Prompt:      msg.Task.Prompt,
					Model:       msg.Task.Model,
					TaskId:      msg.Id,
				}); err != nil {
					log.Println("send error:", err)
				}
			default:
				log.Println("no pending task right now")
			}
		}

	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a signal to shutdown
	sig := <-sigChan
	agg.logger.Info("Received signal, shutting down...", zap.Any("signal", sig))

	cancel()

	return nil
}

// pseudo proposer picking
func (agg *Aggregator) PickProposer(taskId uint64) string {
	var addr string
	isEven := false

	if taskId%2 == 0 {
		isEven = true
	}

	for _, op := range agg.CurrentOperators {
		if isEven {
			addr = op.Pubkey
			break
		}

		addr = op.Pubkey
	}

	return addr
}
