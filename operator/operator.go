package operator

import (
	"avs/aggregator"
	"avs/types/proto/socket"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
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

	// start api server
	go func() {
		op.logger.Info("api server process started...")

		router := op.SetupRoutes()

		log.Println("Server running on http://localhost:8080")
		err := router.Run(":8080")
		if err != nil {
			op.logger.Fatal("Error when starting api server", zap.Any("err", err))
		}
	}()

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
				Id: task.TaskId,
				Task: map[string]interface{}{
					"prompt": task.Prompt,
					"model":  task.Model,
				},
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
	// TODO: route task to the right model
	model := task.Task["model"].(string)
	prompt := task.Task["prompt"].(string)
	payload := aggregator.ChatCompletionRequest{
		Model:  model,
		Stream: false,
		Messages: []aggregator.ChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		op.logger.Error("Failed to serialize payload", zap.Error(err))
		return err
	}
	req, err := http.NewRequest("POST", op.ModelApi+"/v1/chat/completions", bytes.NewBuffer(payloadBytes))
	if err != nil {
		op.logger.Error("Failed to create request", zap.Error(err))
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		op.logger.Error("Failed to contact Chat API", zap.Error(err))
		return err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		op.logger.Error("Failed to read response body", zap.Error(err))
		return err
	}
	req.Header.Set("Authorization", "Bearer token-abc123")
	req.Header.Set("Content-Type", "application/json")

	var parsed map[string]interface{}
	json.Unmarshal(body, &parsed)
	content := ""
	if choices, ok := parsed["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				content, _ = msg["content"].(string)
			}
		}
	}

	// then send the response chunks here
	return op.TaskStream.Send(&socket.TaskResponseMessage{
		TaskId:      task.Id,
		NodeAddress: string(op.BlsPrivateKey),
		Model:       model,
		Resps:       []string{content},
	})
}

func (op *Operator) VerifyResponse(resp *socket.TaskResponseMessage) error {
	// TODO: handle verify here
	// then send the vote here
	// return op.VoteStream.Send()
	return nil
}

// func (op *Operator) FetchTasks() error {
// 	// var taskCount uint64
// 	// // looping
// 	// for {
// 	// 	previousTaskCount := taskCount
// 	// 	newTaskCount, err := LatestTaskCount(client, op.avsAddress)
// 	// 	if err != nil {
// 	// 		op.logger.Warn("Failed to subscribe to new tasks", zap.Any("err", err))
// 	// 		time.Sleep(RetryInterval)
// 	// 		continue
// 	// 	}
// 	// 	taskCount = newTaskCount

// 	// 	if taskCount > previousTaskCount {
// 	// 		err := op.QueueTask(ctx, client, previousTaskCount, taskCount)
// 	// 		if err != nil {
// 	// 			return fmt.Errorf("error queuing task: %v", err)
// 	// 		}
// 	// 	}
// 	// }
// 	price := big.NewInt(int64(getCMCPrice(upperDenom) * 1000000))

// 	msghHash, err := GetMsgHash(client, op.avsAddress, taskId, *price)
// 	if err != nil {
// 		return fmt.Errorf("failed to GetMsgHash: %v", err)
// 	}

// 	trimmedMsgHash := strings.TrimPrefix(msghHash, "0x")
// 	bytesMsgHash, err := hex.DecodeString(trimmedMsgHash)
// 	if err != nil {
// 		return fmt.Errorf("failed to decode hex to string: %v", err)
// 	}

// 	var priv crypto.BlsPrivateKey
// 	err = priv.FromBytes(op.BlsPrivateKey)
// 	if err != nil {
// 		panic("Failed to create bls priv key" + err.Error())
// 	}
// 	signature, err := priv.Sign(bytesMsgHash)
// 	if err != nil {
// 		panic("Failed to create signature" + err.Error())
// 	}

// 	// pubKey, err := priv.GeneratePubkey()
// 	// if err != nil {
// 	// 	panic("Failed to generate pubkey from privkey" + err.Error())
// 	// }

// 	op.AggRpcClient.SendSignedTaskResponseToAggregator(aggregator.SignedTaskResponse{
// 		TaskId:    taskId,
// 		Pubkey:    signature.Auth.PublicKey().Bytes(),
// 		Signature: signature.Auth.Signature().Bytes(),
// 		Response:  price,
// 	})

// 	return nil
// }
