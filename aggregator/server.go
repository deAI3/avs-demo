package aggregator

import (
	"avs/types/proto/socket"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/rpc"
	"os"

	"path/filepath"
	"strconv"
	"strings"

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
	agg.CurrentOperators = operators
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
		agg.CurrentOperators = operators
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
	var timestamp uint64
	var err error
	agg.TaskMutex.Lock()
	taskInfo, exists := agg.PendingTasks[signedTaskResponse.TaskId]
	if exists {
		var timestampStr = taskInfo.State["task_created_timestamp"].(string)
		timestamp, err = strconv.ParseUint(timestampStr, 10, 64) // base 10, 64-bit size
		if err != nil {
			return fmt.Errorf("error converting string to uint64: %v", err)
		}
	} else {
		task, err := LoadTaskById(client, avs, signedTaskResponse.TaskId)
		if err != nil {
			return fmt.Errorf("error loading task: %v", err)
		}
		var timestampStr = task["task_created_timestamp"].(string)
		timestamp, err = strconv.ParseUint(timestampStr, 10, 64) // base 10, 64-bit size
		if err != nil {
			return fmt.Errorf("error converting string to uint64: %v", err)
		}
		taskInfo = TaskInfo{
			State:     task,
			Responses: make([]SignedTaskResponse, 0),
		}
		agg.PendingTasks[signedTaskResponse.TaskId] = taskInfo
	}

	taskInfo.Responses[signedTaskResponse.TaskId] = append(taskInfo.Responses[signedTaskResponse.TaskId], signedTaskResponse)
	resps := []U128Struct{}
	pks := []BytesStruct{}
	sigs := []BytesStruct{}
	msgs := []BytesStruct{}
	for _, response := range taskInfo.Responses {
		pks = append(pks, BytesStruct{
			Value: response.Pubkey,
		})
		sigs = append(sigs, BytesStruct{
			Value: response.Signature,
		})
		resps = append(resps, U128Struct{
			Value: response.Response,
		})
	}

	// GetMsgHashes
	msgHashes, err := GetMsgHashes(client, agg.AvsAddress, signedTaskResponse.TaskId,
		resps,
		pks,
	)
	if err != nil {
		return fmt.Errorf("failed to get msg hashes: %v", err)
	}
	for _, hash := range msgHashes {
		hexStr, ok := hash.(string)
		if !ok {
			return fmt.Errorf("data is not a string")
		}
		trimmedHexStr := strings.TrimPrefix(hexStr, "0x")
		bytesMsgHash, err := hex.DecodeString(trimmedHexStr)
		if err != nil {
			return fmt.Errorf("can't decode string: %v", err)
		}
		msgs = append(msgs, BytesStruct{
			Value: bytesMsgHash,
		})
	}

	signedStake, totalStake, err := CheckSignatures(client, agg.AvsAddress, 1, timestamp,
		msgs,
		pks,
		sigs,
	)
	if err != nil {
		return fmt.Errorf("can't check signature: %v", err)
	}

	fmt.Println("signedStake: ", signedStake)
	fmt.Println("totalStake: ", totalStake)
	// (signed_stake * THRESHOLD_DENOMINATOR) >= (total_stake * QUORUM_THRESHOLD_PERCENTAGE)
	if signedStake*THRESHOLD_DENOMINATOR >= totalStake*QUORUM_THRESHOLD_PERCENTAGE {
		agg.logger.Info("Quorum for task has reached. Responding...", zap.Any("task_id", signedTaskResponse.TaskId))

		err = RespondToAvs(client, &agg.AggregatorAccount, agg.AvsAddress, signedTaskResponse.TaskId,
			sigs,
			pks,
			resps,
		)

		if err != nil {
			return fmt.Errorf("failed to respond task: %v", err)
		}
	} else {
		agg.logger.Info("Quorum for task has not reached. Waiting for other operators", zap.Any("task_id", signedTaskResponse.TaskId), zap.Any("Consensus", float64(signedStake*THRESHOLD_DENOMINATOR)/float64(totalStake)))
	}

	agg.PendingTasks[signedTaskResponse.TaskId] = taskInfo
	agg.TaskMutex.Unlock()
	return nil
}

// aggregator: &signer,
// task_id: u64,
// responses: vector<u128>,
// signer_pubkeys: vector<vector<u8>>,
// signer_sigs: vector<vector<u8>>,
// TODO update here
func RespondToAvs(
	client *aptos.Client,
	aggregatorAccount *aptos.Account,
	contractAddr string,
	taskId uint64,
	signature []BytesStruct,
	pubkey []BytesStruct,
	responses []U128Struct,
) error {
	contract := aptos.AccountAddress{}
	err := contract.ParseStringRelaxed(contractAddr)
	if err != nil {
		panic("Failed to parse address:" + err.Error())
	}
	taskIdBcs, err := bcs.SerializeU64(taskId)
	if err != nil {
		panic("Failed to bcs serialize task id:" + err.Error())
	}

	sigSerializer := bcs.Serializer{}
	bcs.SerializeSequence(signature, &sigSerializer)

	pubkeySerializer := bcs.Serializer{}
	bcs.SerializeSequence(pubkey, &pubkeySerializer)

	responseSerializer := bcs.Serializer{}
	bcs.SerializeSequence(responses, &responseSerializer)
	payload := aptos.EntryFunction{
		Module: aptos.ModuleId{
			Address: contract,
			Name:    "service_manager",
		},
		Function: "respond_to_task",
		ArgTypes: []aptos.TypeTag{},
		Args: [][]byte{
			taskIdBcs, responseSerializer.ToBytes(), pubkeySerializer.ToBytes(), sigSerializer.ToBytes(),
		},
	}

	// Build transaction
	rawTxn, err := client.BuildTransaction(aggregatorAccount.AccountAddress(),
		aptos.TransactionPayload{Payload: &payload})
	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}

	// Sign transaction
	signedTxn, err := rawTxn.SignedTransaction(aggregatorAccount)
	if err != nil {
		panic("Failed to sign transaction:" + err.Error())
	}
	fmt.Printf("Submit register operator for %s\n", aggregatorAccount.AccountAddress())

	// Submit and wait for it to complete
	submitResult, err := client.SubmitTransaction(signedTxn)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := submitResult.Hash

	// Wait for the transaction
	fmt.Printf("And we wait for the transaction %s to complete...\n", txnHash)
	userTxn, err := client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	fmt.Printf("The transaction completed with hash: %s and version %d\n", userTxn.Hash, userTxn.Version)
	if !userTxn.Success {
		// TODO: log something more
		panic("Failed to respond to avs")
	}
	return nil
}

// quorum_numbers: vector<u8>,
// reference_timestamp: u64,
// msg_hashes: vector<vector<u8>>,
// signer_pubkeys: vector<vector<u8>>,
// signer_sigs: vector<vector<u8>>,
// TODO update here
func CheckSignatures(
	contractAddr string,
	quorumNumbers uint8,
	referenceTimestamp uint64,
	msgHashes []byte,
	pubkey []byte,
	signature []byte,
) (uint64, uint64, error) {
	contract := aptos.AccountAddress{}
	err := contract.ParseStringRelaxed(contractAddr)
	if err != nil {
		panic("Failed to parse address:" + err.Error())
	}

	quorumSerializer := &bcs.Serializer{}
	bcs.SerializeSequence([]U8Struct{
		{
			Value: quorumNumbers,
		},
	}, quorumSerializer)

	timestampBcs, err := bcs.SerializeU64(referenceTimestamp)
	if err != nil {
		panic("Failed to SerializeU64:" + err.Error())
	}

	sigSerializer := bcs.Serializer{}
	bcs.SerializeSequence(signature, &sigSerializer)

	pubkeySerializer := bcs.Serializer{}
	bcs.SerializeSequence(pubkey, &pubkeySerializer)

	msgHashesSerializer := bcs.Serializer{}
	bcs.SerializeSequence(msgHashes, &msgHashesSerializer)

	payload := &aptos.ViewPayload{
		Module: aptos.ModuleId{
			Address: contract,
			Name:    "bls_sig_checker",
		},
		Function: "check_signatures",
		ArgTypes: []aptos.TypeTag{},
		Args: [][]byte{
			quorumSerializer.ToBytes(),
			timestampBcs,
			msgHashesSerializer.ToBytes(),
			pubkeySerializer.ToBytes(),
			sigSerializer.ToBytes(),
		},
	}

	vals, err := client.View(payload)
	if err != nil {
		return 0, 0, err
	}
	signedStakeStr := vals[0].([]interface{})[0].(string)
	signedStake, err := strconv.ParseUint(signedStakeStr, 10, 64) // base 10, 64-bit size
	if err != nil {
		return 0, 0, fmt.Errorf("error converting string to uint64: %v", err)
	}
	totalStakeStr := vals[1].([]interface{})[0].(string)
	totalStake, err := strconv.ParseUint(totalStakeStr, 10, 64) // base 10, 64-bit size
	if err != nil {
		return 0, 0, fmt.Errorf("error converting string to uint64: %v", err)
	}
	return signedStake, totalStake, nil
}
