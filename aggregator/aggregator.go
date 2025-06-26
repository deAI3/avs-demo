package aggregator

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

const taskQueueSize = 100

func NewAggregator(aggregatorConfig AggregatorConfig, logger *zap.Logger) (*Aggregator, error) {
	// aggegator_account, err := SignerFromConfig(aggregatorConfig.AccountConfig.AccountPath, aggregatorConfig.AccountConfig.Profile)
	// if err != nil {
	// 	return &Aggregator{}, errors.Wrap(err, "Failed to create aggregator account")
	// }

	agg := Aggregator{
		logger: logger,

		AggregatorConfig: aggregatorConfig,
		TaskQueue:        make(chan TaskInfo, taskQueueSize),
		PendingTasks:     []TaskInfo{},
		CurrentTaskId:    0,
		// Network: network,
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

	// start api server and websocket
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
	// - wait for verifier results to update task state
	// - when task reach finalized state, store results on state
	go func() {
		for {
			for addr, conn := range agg.clients {

			}
			msg := <-agg.responsesRecv

			// TODO: emit responses to verifier node

		}
	}()

	// distribute tasks to nodes through the following steps:
	//
	// - choose a propser node
	// - emit event includes the task and propose node address
	go func() {
		agg.logger.Info("Fetching tasks process started...")
		// err := agg.FetchTasks(ctx)
		// if err != nil {
		// 	agg.logger.Fatal("Error listening for tasks", zap.Any("err", err))
		// }
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a signal to shutdown
	sig := <-sigChan
	agg.logger.Info("Received signal, shutting down...", zap.Any("signal", sig))

	cancel()

	return nil
}
