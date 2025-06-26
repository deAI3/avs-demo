package aggregator

import (
	"avs/types/proto/socket"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/rpc"
	"os"
	"time"

	"path/filepath"

	"go.uber.org/zap"
)

const (
	THRESHOLD_DENOMINATOR       uint64 = 100
	QUORUM_THRESHOLD_PERCENTAGE uint64 = 67
)

func (agg *Aggregator) ServeOperators() error {
	// Registers a new RPC server
	err := rpc.Register(agg)
	if err != nil {
		return err
	}

	// Registers an HTTP handler for RPC messages
	rpc.HandleHTTP()

	agg.logger.Info("Starting RPC server on address:", zap.String("address", agg.AggregatorConfig.ServerIpPortAddress))

	err = http.ListenAndServe(agg.AggregatorConfig.ServerIpPortAddress, nil)
	if err != nil {
		return err
	}

	return nil
}

func (agg *Aggregator) HandleOperatorRegister(operator Operator, reply *uint8) error {
	agg.logger.Info("Received operators request")

	if err := agg.processOperatorRegisterRequest(operator); err != nil {
		agg.logger.Error("Failed to process operator register request", zap.Error(err))
		return fmt.Errorf("failed to process operator register request: %v", err)
	}
	// establish a stream channel
	agg.TaskClients[string(operator.Pubkey)] = make(chan socket.TaskMessage, 10)
	agg.VoteClients[string(operator.Pubkey)] = make(chan socket.TaskResponseMessage, 10)

	// Set reply to indicate success (e.g., 0 = success)
	*reply = 0

	agg.logger.Info("Successfully processed operator register request", zap.Any("operator", operator))
	return nil
}

func (agg *Aggregator) HandleOperatorDeregister(pubkey []byte, reply *uint8) error {
	agg.logger.Info("Received operator deregister request", zap.ByteString("pubkey", pubkey))

	if err := agg.processOperatorDeregisterRequest(pubkey); err != nil {
		agg.logger.Error("Failed to process operator deregister request", zap.Error(err))
		return fmt.Errorf("failed to process operator deregister request: %v", err)
	}

	// remove stream channel
	delete(agg.TaskClients, string(pubkey))
	delete(agg.VoteClients, string(pubkey))

	// Set reply to indicate success (e.g., 0 = success)
	*reply = 0
	agg.logger.Info("Successfully processed operator deregister request", zap.ByteString("pubkey", pubkey))
	return nil
}

// Define the RespondTask method for handling incoming RPC calls
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

func (agg *Aggregator) processOperatorRegisterRequest(operator Operator) error {
	agg.OperatorMutex.Lock()

	var operators map[string]Operator
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
	if _, has := operators[string(operator.Pubkey)]; has {
		agg.OperatorMutex.Unlock()
		return fmt.Errorf("operator already exists", operator.Pubkey)
	}

	// Add the new operator to the list
	operators[string(operator.Pubkey)] = operator
	currentOperators := make([]Operator, 0, len(operators))
	for _, op := range operators {
		currentOperators = append(currentOperators, op)
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
	var operators map[string]Operator
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
		currentOperators := make([]Operator, 0, len(operators))
		for _, op := range operators {
			currentOperators = append(currentOperators, op)
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
			// (signed_stake * THRESHOLD_DENOMINATOR) >= (total_stake * QUORUM_THRESHOLD_PERCENTAGE)
			// if signedStake*THRESHOLD_DENOMINATOR >= totalStake*QUORUM_THRESHOLD_PERCENTAGE {
			// 	agg.logger.Info("Quorum for task has reached. Responding...", zap.Any("task_id", signedTaskResponse.TaskId))

			// 	err = RespondToAvs(client, &agg.AggregatorAccount, agg.AvsAddress, signedTaskResponse.TaskId,
			// 		sigs,
			// 		pks,
			// 		resps,
			// 	)

			// 	if err != nil {
			// 		return fmt.Errorf("failed to respond task: %v", err)
			// }
			// } else {
			// 	agg.logger.Info("Quorum for task has not reached. Waiting for other operators", zap.Any("task_id", signedTaskResponse.TaskId), zap.Any("Consensus", float64(signedStake*THRESHOLD_DENOMINATOR)/float64(totalStake)))
			// }
		}
	} else {
		return fmt.Errorf("task %d is not in valid state", signedTaskResponse.TaskId)
	}

	agg.PendingTasks[signedTaskResponse.TaskId] = taskInfo
	agg.TaskMutex.Unlock()
	return nil
}

// // aggregator: &signer,
// // task_id: u64,
// // responses: vector<u128>,
// // signer_pubkeys: vector<vector<u8>>,
// // signer_sigs: vector<vector<u8>>,
// // TODO update here
// func RespondToAvs(
// 	client *aptos.Client,
// 	aggregatorAccount *aptos.Account,
// 	contractAddr string,
// 	taskId uint64,
// 	signature []BytesStruct,
// 	pubkey []BytesStruct,
// 	responses []U128Struct,
// ) error {
// 	contract := aptos.AccountAddress{}
// 	err := contract.ParseStringRelaxed(contractAddr)
// 	if err != nil {
// 		panic("Failed to parse address:" + err.Error())
// 	}
// 	taskIdBcs, err := bcs.SerializeU64(taskId)
// 	if err != nil {
// 		panic("Failed to bcs serialize task id:" + err.Error())
// 	}

// 	sigSerializer := bcs.Serializer{}
// 	bcs.SerializeSequence(signature, &sigSerializer)

// 	pubkeySerializer := bcs.Serializer{}
// 	bcs.SerializeSequence(pubkey, &pubkeySerializer)

// 	responseSerializer := bcs.Serializer{}
// 	bcs.SerializeSequence(responses, &responseSerializer)
// 	payload := aptos.EntryFunction{
// 		Module: aptos.ModuleId{
// 			Address: contract,
// 			Name:    "service_manager",
// 		},
// 		Function: "respond_to_task",
// 		ArgTypes: []aptos.TypeTag{},
// 		Args: [][]byte{
// 			taskIdBcs, responseSerializer.ToBytes(), pubkeySerializer.ToBytes(), sigSerializer.ToBytes(),
// 		},
// 	}

// 	// Build transaction
// 	rawTxn, err := client.BuildTransaction(aggregatorAccount.AccountAddress(),
// 		aptos.TransactionPayload{Payload: &payload})
// 	if err != nil {
// 		panic("Failed to build transaction:" + err.Error())
// 	}

// 	// Sign transaction
// 	signedTxn, err := rawTxn.SignedTransaction(aggregatorAccount)
// 	if err != nil {
// 		panic("Failed to sign transaction:" + err.Error())
// 	}
// 	fmt.Printf("Submit register operator for %s\n", aggregatorAccount.AccountAddress())

// 	// Submit and wait for it to complete
// 	submitResult, err := client.SubmitTransaction(signedTxn)
// 	if err != nil {
// 		panic("Failed to submit transaction:" + err.Error())
// 	}
// 	txnHash := submitResult.Hash

// 	// Wait for the transaction
// 	fmt.Printf("And we wait for the transaction %s to complete...\n", txnHash)
// 	userTxn, err := client.WaitForTransaction(txnHash)
// 	if err != nil {
// 		panic("Failed to wait for transaction:" + err.Error())
// 	}
// 	fmt.Printf("The transaction completed with hash: %s and version %d\n", userTxn.Hash, userTxn.Version)
// 	if !userTxn.Success {
// 		// TODO: log something more
// 		panic("Failed to respond to avs")
// 	}
// 	return nil
// }

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
