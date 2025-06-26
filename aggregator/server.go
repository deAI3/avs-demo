package aggregator

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"path/filepath"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	pb "avs/types/proto/aggregator"
	"avs/types/proto/socket"
)

const (
	THRESHOLD_DENOMINATOR       uint64 = 100
	QUORUM_THRESHOLD_PERCENTAGE uint64 = 67
)

func (agg *Aggregator) ServeOperators() error {
	grpcAddr := ":50051"
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatal("failed to listen:", err)
	}

	server := grpc.NewServer()

	pb.RegisterOperatorServiceServer(server, agg)
	socket.RegisterTaskServiceServer(server, agg)

	log.Println("gRPC server started on", grpcAddr)
	if err := server.Serve(lis); err != nil {
		log.Fatal("gRPC server error:", err)
	}

	return nil
}

func (agg *Aggregator) RegisterOperator(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterRespond, error) {
	agg.logger.Info("Received operators request")

	if err := agg.processOperatorRegisterRequest(req.Operator); err != nil {
		agg.logger.Error("Failed to process operator register request", zap.Error(err))
		return &pb.RegisterRespond{
			Respond: "failed",
		}, fmt.Errorf("failed to process operator register request: %v", err)
	}

	agg.logger.Info("Successfully processed operator register request", zap.Any("operator", req.Operator))
	return &pb.RegisterRespond{
		Respond: "success",
	}, nil
}

func (agg *Aggregator) DeregisterOperator(ctx context.Context, req *pb.DeregisterRequest) (*pb.DeregisterRespond, error) {
	agg.logger.Info("Received operator deregister request", zap.String("pubkey", req.Pubkey))

	if err := agg.processOperatorDeregisterRequest([]byte(req.Pubkey)); err != nil {
		agg.logger.Error("Failed to process operator deregister request", zap.Error(err))
		return &pb.DeregisterRespond{
			Respond: "failed",
		}, fmt.Errorf("failed to process operator deregister request: %v", err)
	}

	// remove stream channel
	delete(agg.TaskClients, req.Pubkey)
	delete(agg.VoteClients, req.Pubkey)

	agg.logger.Info("Successfully processed operator deregister request", zap.String("pubkey", req.Pubkey))
	return &pb.DeregisterRespond{
		Respond: "success",
	}, nil
}

// Define the RespondTask method for handling incoming RPC calls
// TODO update here
func (agg *Aggregator) RespondTask(signedTaskResponse SignedTaskResponse, reply *uint8) error {
	agg.logger.Info("Received signed task response", zap.Any("response", signedTaskResponse))

	// Process the signed task response
	if err := agg.processTaskResponse(signedTaskResponse); err != nil {
		agg.logger.Error("Failed to process signed task response", zap.Error(err))
		return fmt.Errorf("failed to process task response: %v", err)
	}

	// Set reply to indicate success (e.g., 0 = success)
	*reply = 0
	agg.logger.Info("Successfully processed signed task response")
	return nil
}

func (agg *Aggregator) processOperatorRegisterRequest(operator *pb.Operator) error {
	agg.OperatorMutex.Lock()

	var operators map[string]*pb.Operator
	storePath := filepath.Join(agg.AggregatorConfig.StorePath, "operators.json")
	if _, err := os.Stat(storePath); err == nil {
		data, err := os.ReadFile(storePath)
		if err != nil {
			agg.OperatorMutex.Unlock()
			return fmt.Errorf("error reading operators file: %v", err)
		}

		err = json.Unmarshal(data, &operators)
		if err != nil {
			agg.OperatorMutex.Unlock()
			return fmt.Errorf("error unmarshalling operators data: %v", err)
		}
	} else if !os.IsNotExist(err) {
		agg.OperatorMutex.Unlock()
		return fmt.Errorf("error checking operators file: %v", err)
	}

	// Check if the operator already exists
	if _, has := operators[operator.Pubkey]; has {
		agg.OperatorMutex.Unlock()
		return fmt.Errorf("operator already exists", operator.Pubkey)
	}

	// Add the new operator to the list
	operators[operator.Pubkey] = operator
	currentOperators := make(map[string]*pb.Operator, len(operators))
	for _, op := range operators {
		currentOperators[op.Pubkey] = op
	}
	agg.CurrentOperators = currentOperators
	agg.OperatorMutex.Unlock()
	bz, err := json.Marshal(operators)
	if err != nil {
		return fmt.Errorf("error marshalling operators data: %v", err)
	}
	err = os.WriteFile(storePath, bz, 0644)
	if err != nil {
		return fmt.Errorf("error writing operators file: %v", err)
	}
	agg.logger.Info("Operator registered successfully", zap.Any("operator", operator))
	return nil
}

func (agg *Aggregator) processOperatorDeregisterRequest(pubkey []byte) error {
	agg.OperatorMutex.Lock()

	storePath := filepath.Join(agg.AggregatorConfig.StorePath, "operators.json")
	var operators map[string]*pb.Operator
	if _, err := os.Stat(storePath); err == nil {
		data, err := os.ReadFile(storePath)
		if err != nil {
			agg.OperatorMutex.Unlock()
			return fmt.Errorf("error reading operators file: %v", err)
		}

		err = json.Unmarshal(data, &operators)
		if err != nil {
			agg.OperatorMutex.Unlock()
			return fmt.Errorf("error unmarshalling operators data: %v", err)
		}
	} else {
		agg.OperatorMutex.Unlock()
		return fmt.Errorf("error checking operators file: %v", err)
	}

	// Find and remove the operator with the given pubkey
	if _, has := operators[string(pubkey)]; has {
		delete(operators, string(pubkey))
		currentOperators := make(map[string]*pb.Operator, len(operators))
		for _, op := range operators {
			currentOperators[op.Pubkey] = op
		}

		agg.CurrentOperators = currentOperators
		bz, err := json.Marshal(operators)
		if err != nil {
			agg.OperatorMutex.Unlock()
			return fmt.Errorf("error marshalling operators data: %v", err)
		}
		err = os.WriteFile(storePath, bz, 0644)
		if err != nil {
			agg.OperatorMutex.Unlock()
			return fmt.Errorf("error writing operators file: %v", err)
		}
		agg.logger.Info("Operator deregistered successfully", zap.ByteString("pubkey", pubkey))
		agg.OperatorMutex.Unlock()
		return nil
	}

	agg.OperatorMutex.Unlock()
	return fmt.Errorf("operator with pubkey %s not found", pubkey)
}

// TODO update here
func (agg *Aggregator) processTaskResponse(signedTaskResponse SignedTaskResponse) error {
	var err error
	agg.TaskMutex.Lock()
	taskInfo := agg.PendingTasks[signedTaskResponse.TaskId]

	if err != nil {
		return fmt.Errorf("can't check signature: %v", err)
	}

	currTime := time.Now()
	if taskInfo.TaskCreatedTime.Add(taskInfo.TaskConfig.InferenceStepExpirationTime).Before(currTime) {
		return fmt.Errorf("task %d is expired", signedTaskResponse.TaskId)
	}

	if taskInfo.Task.State == WaitingForApply {
		if signedTaskResponse.ApplyTask == nil {
			return fmt.Errorf("apply task is nil for task %d", signedTaskResponse.TaskId)
		}
		taskInfo.Task.AppliedOperators = append(taskInfo.Task.AppliedOperators, signedTaskResponse.Pubkey)
		buf := new(bytes.Buffer)
		if err := binary.Write(buf, binary.LittleEndian, signedTaskResponse.TaskId); err != nil {
			return err
		}
		msgHash := sha256.Sum256(buf.Bytes())

		err = CheckSignatures(msgHash[:],
			signedTaskResponse.Pubkey,
			signedTaskResponse.ApplyTask.Signature)
		if err != nil {
			agg.logger.Error("Failed to check signatures", zap.Error(err))
			return fmt.Errorf("failed to check signatures: %v", err)
		}
	} else if taskInfo.Task.State == TempResult {
		if signedTaskResponse.IsGenerator {

			if signedTaskResponse.GenerateAnswer == nil {
				return fmt.Errorf("generate answer is nil for task %d", signedTaskResponse.TaskId)
			}
			// Get operator index
			operatorIndex := -1
			for i, op := range taskInfo.Task.AppliedOperators {
				if bytes.Equal(op, signedTaskResponse.Pubkey) {
					operatorIndex = i
					break
				}
			}
			if operatorIndex == -1 {
				return fmt.Errorf("operator with pubkey %s is not applied to task %d", signedTaskResponse.Pubkey, signedTaskResponse.TaskId)
			}
			buf := new(bytes.Buffer)
			if err := binary.Write(buf, binary.LittleEndian, signedTaskResponse.TaskId); err != nil {
				return err
			}

			buf.Write([]byte(signedTaskResponse.GenerateAnswer.Response))
			msgHash := sha256.Sum256(buf.Bytes())
			err = CheckSignatures(msgHash[:],
				signedTaskResponse.Pubkey,
				signedTaskResponse.GenerateAnswer.Signature)
			if err != nil {
				agg.logger.Error("Failed to check signatures", zap.Error(err))
				return fmt.Errorf("failed to check signatures: %v", err)
			}
			taskInfo.Task.TempResult.InferenceStep.GeneratorResults[uint64(operatorIndex)] = signedTaskResponse.GenerateAnswer.Response
		} else {
			// TODO: verify and check if finalized

		}
	} else {
		return fmt.Errorf("task %d is not in valid state", signedTaskResponse.TaskId)
	}

	agg.PendingTasks[signedTaskResponse.TaskId] = taskInfo
	agg.TaskMutex.Unlock()
	return nil
}

// quorum_numbers: vector<u8>,
// reference_timestamp: u64,
// msg_hashes: vector<vector<u8>>,
// signer_pubkeys: vector<vector<u8>>,
// signer_sigs: vector<vector<u8>>,
// TODO update here
func CheckSignatures(
	msgHashes []byte,
	pubkey []byte,
	signature []byte,
) error {

	return nil
}

// establish stream for node operator and handle receive message flow
func (agg *Aggregator) TaskStream(stream socket.TaskService_TaskStreamServer) error {
	// first Msg
	firstMsg, err := stream.Recv()
	if err != nil {
		log.Println("client recv error:", err)
		return err
	}

	if _, has := agg.TaskClients[firstMsg.NodeAddress]; !has {
		// establish a stream channel
		agg.TaskClients[firstMsg.NodeAddress] = stream
	}

	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Println("client recv error:", err)
			return err
		}

		// add respond to chan
		agg.respondsChan <- msg
	}
}

// establish stream for node operator and handle receive message flow
func (agg *Aggregator) VoteStream(stream socket.TaskService_VoteStreamServer) (err error) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Println("client recv error:", err)
			return err
		}

		if _, has := agg.TaskClients[msg.NodeAddress]; !has {
			// establish a stream channel
			agg.VoteClients[msg.NodeAddress] = stream
		}

		// add respond to chan
		agg.voteChan <- msg
	}
}
