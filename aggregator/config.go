package aggregator

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/crypto"
	"gopkg.in/yaml.v2"
)

type AptosConfig struct {
	Profiles map[string]Profile `yaml:"profiles"`
}

type Profile struct {
	Network    string `yaml:"devnet"`
	PrivateKey string `yaml:"private_key"`
	PublicKey  string `yaml:"public_key"`
	Account    string `yaml:"account"`
	RestURL    string `yaml:"rest_url"`
	FaucetURL  string `yaml:"faucet_url"`
}

func SignerFromConfig(path string, profile string) (*aptos.Account, error) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	fmt.Println("yamlFile :", string(yamlFile))
	var aptosConfig AptosConfig
	err = yaml.Unmarshal(yamlFile, &aptosConfig)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	trimmedPriv := strings.TrimPrefix(aptosConfig.Profiles[profile].PrivateKey, "0x")
	trimmedPub := strings.TrimPrefix(aptosConfig.Profiles[profile].PublicKey, "0x")

	privateKeyBytes, err := hex.DecodeString(trimmedPriv)
	if err != nil {
		panic("Could not get aggregator priv key:" + err.Error())
	}
	publicKeyBytes, err := hex.DecodeString(trimmedPub)
	if err != nil {
		panic("Could not get aggregator pub key:" + err.Error())
	}

	privateKey := make([]byte, 64)
	copy(privateKey[:32], privateKeyBytes)
	copy(privateKey[32:], publicKeyBytes)
	priv := crypto.Ed25519PrivateKey{
		Inner: privateKey,
	}
	// Create the sender from the key locally
	sender, err := aptos.NewAccountFromSigner(&priv)
	if err != nil {
		panic("Could not get account from signer:" + err.Error())
	}
	return sender, nil
}
