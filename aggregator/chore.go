package aggregator

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/bcs"
)

const ChoreInterval = 1 * time.Minute

func (agg *Aggregator) DoChore(ctx context.Context) error {
	client, err := aptos.NewClient(agg.Network)
	if err != nil {
		return fmt.Errorf("failed to create aptos client: %v", err)
	}

	avsAddress := aptos.AccountAddress{}
	err = avsAddress.ParseStringRelaxed(agg.AvsAddress)
	if err != nil {
		return fmt.Errorf("failed to parse avs address: %v", err)
	}

	for {
		// Get quorum count
		quorumCount, err := QuorumCount(client, avsAddress)
		if err != nil {
			return fmt.Errorf("failed to get quorum count: %v", err)
		}

		var operatorsIdPerQuorum [][]interface{}

		if quorumCount != 0 {
			for i := 1; i <= int(quorumCount); i++ {
				operatorList, err := GetOperatorListAtTimestamp(client, avsAddress, uint8(i), uint64(time.Now().Unix()))
				if err != nil {
					return fmt.Errorf("can not get operator list %v", err)
				}
				operatorsIdPerQuorum = append(operatorsIdPerQuorum, operatorList)
			}
		}

		var operatorAddresses = make([][]aptos.AccountAddress, quorumCount)
		// Get list operator for each quorum
		for i, operatorsIds := range operatorsIdPerQuorum {
			operatorAddresses[i] = make([]aptos.AccountAddress, len(operatorsIds))
			for j := 0; j < len(operatorsIds); j++ {
				opId, ok := operatorsIds[j].(string)
				if !ok {
					return fmt.Errorf("can not convert operator id to sting")
				}
				strimmedId := strings.TrimPrefix(opId, "0x")
				bytes, err := hex.DecodeString(strimmedId)
				if err != nil {
					return err
				}
				operatorAddr, err := GetOperatorAddress(client, avsAddress, bytes)
				if err != nil {
					return fmt.Errorf("can not get operator address %v", err)
				}
				addr := aptos.AccountAddress{}
				err = addr.ParseStringRelaxed(operatorAddr)
				if err != nil {
					return fmt.Errorf("can not ParseStringRelaxed: %v", err)
				}
				operatorAddresses[i][j] = addr
			}
		}

		// Update quorums
		err = UpdateOperatorsForQuorum(client, &agg.AggregatorAccount, agg.AvsAddress, quorumCount, operatorAddresses)
		if err != nil {
			return fmt.Errorf("can not update operators for quorum: %v", err)
		}
		agg.logger.Info("Done UpdateOperatorsForQuorum. Next update after 1 min")
		time.Sleep(ChoreInterval)
	}
}

func QuorumCount(client *aptos.Client, contract aptos.AccountAddress) (uint8, error) {
	payload := &aptos.ViewPayload{
		Module: aptos.ModuleId{
			Address: contract,
			Name:    "registry_coordinator",
		},
		Function: "quorum_count",
		ArgTypes: []aptos.TypeTag{},
		Args:     [][]byte{},
	}

	vals, err := client.View(payload)
	if err != nil {
		return 0, fmt.Errorf("no quorum found")
	}
	count := vals[0].(float64)
	return uint8(count), nil
}

func GetOperatorListAtTimestamp(client *aptos.Client, contract aptos.AccountAddress, quorum uint8, timestamp uint64) ([]interface{}, error) {
	quorumBcs, err := bcs.SerializeU8(quorum)
	if err != nil {
		return nil, err
	}
	timestampBcs, err := bcs.SerializeU64(timestamp)
	if err != nil {
		return nil, err
	}
	payload := &aptos.ViewPayload{
		Module: aptos.ModuleId{
			Address: contract,
			Name:    "index_registry",
		},
		Function: "get_operator_list_at_timestamp",
		ArgTypes: []aptos.TypeTag{},
		Args: [][]byte{
			quorumBcs, timestampBcs,
		},
	}

	vals, err := client.View(payload)
	if err != nil {
		return nil, err
	}
	operatorList := vals[0].([]interface{})
	return operatorList, nil
}

func GetOperatorAddress(client *aptos.Client, contract aptos.AccountAddress, operatorId []byte) (string, error) {
	operatorIdBcs, err := bcs.SerializeBytes(operatorId)
	if err != nil {
		return "", err
	}
	payload := &aptos.ViewPayload{
		Module: aptos.ModuleId{
			Address: contract,
			Name:    "registry_coordinator",
		},
		Function: "get_operator_address",
		ArgTypes: []aptos.TypeTag{},
		Args: [][]byte{
			operatorIdBcs,
		},
	}

	vals, err := client.View(payload)
	if err != nil {
		return "", err
	}
	operatorList := vals[0].(string)
	return operatorList, nil
}

func UpdateOperatorsForQuorum(
	client *aptos.Client,
	aggregatorAccount *aptos.Account,
	contractAddr string,
	quorumNumbers uint8,
	addresses [][]aptos.AccountAddress,
) error {
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

	vecAddrs := []VecAddr{}
	for _, vecAddress := range addresses {
		vecAddrs = append(vecAddrs, VecAddr{
			Value: vecAddress,
		})
	}
	addressesSerializer := &bcs.Serializer{}
	bcs.SerializeSequence(vecAddrs, addressesSerializer)

	if err != nil {
		panic("Failed to serialize addresses:" + err.Error())
	}

	payload := aptos.EntryFunction{
		Module: aptos.ModuleId{
			Address: contract,
			Name:    "registry_coordinator",
		},
		Function: "update_operators_for_quorum",
		ArgTypes: []aptos.TypeTag{},
		Args: [][]byte{
			quorumSerializer.ToBytes(), addressesSerializer.ToBytes(),
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
		panic("Failed to update operators for quorum")
	}
	return nil
}
