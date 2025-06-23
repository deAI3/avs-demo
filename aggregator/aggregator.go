package aggregator

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/bcs"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const taskQueueSize = 100

func NewAggregator(aggregatorConfig AggregatorConfig, logger *zap.Logger, network aptos.NetworkConfig) (*Aggregator, error) {
	aggegator_account, err := SignerFromConfig(aggregatorConfig.AccountConfig.AccountPath, aggregatorConfig.AccountConfig.Profile)
	if err != nil {
		return &Aggregator{}, errors.Wrap(err, "Failed to create aggregator account")
	}

	agg := Aggregator{
		logger:            logger,
		AvsAddress:        aggregatorConfig.AvsAddress,
		AggregatorAccount: *aggegator_account,
		AggregatorConfig:  aggregatorConfig,
		TaskQueue:         make(chan Task, taskQueueSize),
		PendingTasks:      make(map[uint64]TaskInfo),

		Network: network,
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

	go func() {
		agg.logger.Info("Fetching tasks process started...")
		err := agg.FetchTasks(ctx)
		if err != nil {
			agg.logger.Fatal("Error listening for tasks", zap.Any("err", err))
		}
	}()

	go func() {
		agg.logger.Info("Chore process started...")
		err := agg.DoChore(ctx)
		if err != nil {
			agg.logger.Fatal("Error do chore", zap.Any("err", err))
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

func (agg *Aggregator) FetchTasks(ctx context.Context) error {
	client, err := aptos.NewClient(agg.Network)
	if err != nil {
		return fmt.Errorf("failed to create aptos client: %v", err)
	}

	avs := aptos.AccountAddress{}
	err = avs.ParseStringRelaxed(agg.AvsAddress)
	if err != nil {
		return fmt.Errorf("error parsing avs address: %v", err)
	}

	var taskCount uint64
	// looping
	for {
		previousTaskCount := taskCount
		newTaskCount, err := LatestTaskCount(client, avs)
		if err != nil {
			agg.logger.Warn("Failed to subscribe to new tasks", zap.Any("err", err))
			time.Sleep(RetryInterval)
			continue
		}
		taskCount = newTaskCount

		if taskCount > previousTaskCount {
			err := agg.QueueTask(ctx, avs, client, previousTaskCount, taskCount)
			if err != nil {
				return fmt.Errorf("error queuing task: %v", err)
			}
		}
		time.Sleep(30 * time.Second)
	}
}

func (agg *Aggregator) QueueTask(ctx context.Context, avs aptos.AccountAddress, client *aptos.Client, start uint64, end uint64) error {
	for i := start + 1; i <= end; i++ {
		task, err := LoadTaskById(client, avs, i)
		if err != nil {
			return fmt.Errorf("error loading task: %v", err)
		}
		responded := task["responded"].(bool)
		if responded {
			continue
		}
		agg.logger.Info("Loaded new task with id: %d", zap.Any("task id", i))
		fmt.Println("task :", task)
		agg.TaskQueue <- Task{
			Id:   i,
			Task: task,
		}
		agg.TaskMutex.Lock()
		if _, exists := agg.PendingTasks[i]; !exists {
			agg.PendingTasks[i] = TaskInfo{
				State:     task,
				Responses: make([]SignedTaskResponse, 0),
			}
		}
		agg.TaskMutex.Unlock()
		agg.logger.Info("Queued new task with id: %d", zap.Any("task id", i))
	}

	return nil
}

func LoadTaskById(client *aptos.Client, contract aptos.AccountAddress, taskId uint64) (map[string]interface{}, error) {
	taskIdBcs, err := bcs.SerializeU64(taskId)
	if err != nil {
		return nil, fmt.Errorf("can not SerializeU64: %v", err)
	}
	payload := &aptos.ViewPayload{
		Module: aptos.ModuleId{
			Address: contract,
			Name:    "service_manager",
		},
		Function: "task_by_id",
		ArgTypes: []aptos.TypeTag{},
		Args: [][]byte{
			taskIdBcs,
		},
	}
	vals, err := client.View(payload)
	if err != nil {
		return nil, fmt.Errorf("can not get task count: %v", err)
	}
	task := vals[0].(map[string]interface{})
	return task, nil
}

func LatestTaskCount(client *aptos.Client, contract aptos.AccountAddress) (uint64, error) {
	payload := &aptos.ViewPayload{
		Module: aptos.ModuleId{
			Address: contract,
			Name:    "service_manager",
		},
		Function: "task_count",
		ArgTypes: []aptos.TypeTag{},
		Args:     [][]byte{},
	}

	vals, err := client.View(payload)
	if err != nil {
		return 0, fmt.Errorf("can not get task count: %v", err)
	}
	countStr := vals[0].(string)

	count, err := strconv.ParseUint(countStr, 10, 64) // base 10 and 64-bit size
	if err != nil {
		return 0, fmt.Errorf("error parsing task count: %s", err)
	}
	return uint64(count), nil
}
