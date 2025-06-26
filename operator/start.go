package operator

import (
	"fmt"
	"log"

	aptos "github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/bcs"
	"github.com/aptos-labs/aptos-go-sdk/crypto"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
)

type AptosAccountConfig struct {
	configPath string
	profile    string
}

func AptosClient(networkConfig aptos.NetworkConfig) *aptos.Client {
	// Create a client for Aptos
	client, err := aptos.NewClient(networkConfig)
	if err != nil {
		panic("Failed to create client:" + err.Error())
	}
	return client
}

func NewOperator(logger *zap.Logger, config OperatorConfig, accountConfig AptosAccountConfig, blsPriv []byte) (*Operator, error) {
	operatorAccount, err := SignerFromConfig(accountConfig.configPath, accountConfig.profile)
	if err != nil {
		panic("Failed to create operator account:" + err.Error())
	}

	// registered := IsOperatorRegistered(client, avsAddress, operatorAccount.Address.String())

	// if !registered {
	log.Println("Operator is not registered with A2D avs AVS, registering...")

	// Register Operator
	// ignore error here because panic all the time
	var priv crypto.BlsPrivateKey
	msg := []byte("PubkeyRegistration")
	bcsOperatorAccount, err := bcs.Serialize(&operatorAccount.Address)
	if err != nil {
		panic("Failed to bsc serialize account" + err.Error())
	}

	msg = append(msg, bcsOperatorAccount...)
	keccakMsg := ethcrypto.Keccak256(msg)
	err = priv.FromBytes(config.BlsPrivateKey)
	if err != nil {
		panic("Failed to create bls priv key" + err.Error())
	}

	signature, err := priv.Sign(keccakMsg)
	if err != nil {
		panic("Failed to create signature" + err.Error())
	}
	pop, err := priv.GenerateBlsPop()
	if err != nil {
		panic("Failed to generate bls proof of possession" + err.Error())
	}
	_ = RegisterOperator(
		operatorAccount,
		avsAddress.String(),
		quorumNumbers,
		signature.Auth.Signature().Bytes(),
		signature.PubKey().Bytes(),
		pop.Bytes(),
	)

	// Get OperatorId
	var privKey crypto.BlsPrivateKey
	privKey.FromBytes(config.BlsPrivateKey)

	aggClient, err := NewAggregatorRpcClient(config.AggregatorIpPortAddr)
	if err != nil {
		return nil, fmt.Errorf("can not create new aggregator Rpc Client: %v", err)
	}

	// return Operator
	operator := Operator{
		logger:        logger,
		account:       operatorAccount,
		BlsPrivateKey: blsPriv,
		AggRpcClient:  *aggClient,
		TaskQueue:     make(chan Task, 100),
	}
	return &operator, nil
}

func IsOperatorRegistered(client *aptos.Client, contract aptos.AccountAddress, operator_addr string) bool {
	account := aptos.AccountAddress{}
	err := account.ParseStringRelaxed(operator_addr)
	if err != nil {
		panic("Could not ParseStringRelaxed:" + err.Error())
	}
	operator, err := bcs.Serialize(&account)
	if err != nil {
		panic("Could not serialize operator address:" + err.Error())
	}
	payload := &aptos.ViewPayload{
		Module: aptos.ModuleId{
			Address: contract,
			Name:    "registry_coordinator",
		},
		Function: "get_operator_status",
		ArgTypes: []aptos.TypeTag{},
		Args: [][]byte{
			operator,
		},
	}

	vals, err := client.View(payload)
	if err != nil {
		panic("Could not get operator status:" + err.Error())
	}
	status := vals[0].(float64)
	return status != 0
}

func RegisterOperator(
	operatorAccount *aptos.Account,
	contractAddr string,
	quorumNumbers uint8,
	signature []byte,
	pubkey []byte,
	proofPossession []byte,
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

	sig, err := bcs.SerializeBytes(signature)
	if err != nil {
		panic("Failed to bcs serialize signature:" + err.Error())
	}
	pk, err := bcs.SerializeBytes(pubkey)
	if err != nil {
		panic("Failed to bcs serialize pubkey:" + err.Error())
	}
	pop, err := bcs.SerializeBytes(proofPossession)
	if err != nil {
		panic("Failed to bcs serialize proof of possession:" + err.Error())
	}
	payload := aptos.EntryFunction{
		Module: aptos.ModuleId{
			Address: contract,
			Name:    "registry_coordinator",
		},
		Function: "registor_operator",
		ArgTypes: []aptos.TypeTag{},
		Args: [][]byte{
			quorumSerializer.ToBytes(), sig, pk, pop,
		},
	}
	// Build transaction
	rawTxn, err := client.BuildTransaction(operatorAccount.AccountAddress(),
		aptos.TransactionPayload{Payload: &payload})
	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}

	// Sign transaction
	signedTxn, err := rawTxn.SignedTransaction(operatorAccount)
	if err != nil {
		panic("Failed to sign transaction:" + err.Error())
	}
	fmt.Printf("Submit register operator for %s\n", operatorAccount.AccountAddress())

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
		panic("Failed to register operator")
	}
	return nil
}
