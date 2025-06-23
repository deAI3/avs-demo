package operator

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	aptos "github.com/aptos-labs/aptos-go-sdk"
)

func loadConfig(filename string) (*Config, error) {
	// Open the config file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %v", err)
	}
	defer file.Close()

	// Read the file contents
	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	// Unmarshal the JSON data into the Config struct
	var config Config
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	return &config, nil
}

func loadOperatorConfig(filename string) (*OperatorConfig, error) {
	// Open the config file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %v", err)
	}
	defer file.Close()

	// Read the file contents
	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	// Unmarshal the JSON data into the Config struct
	var config OperatorConfig
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	return &config, nil
}

func extractNetwork(network string) (aptos.NetworkConfig, error) {
	switch network {
	case "devnet":
		return aptos.DevnetConfig, nil
	case "localnet":
		return aptos.LocalnetConfig, nil
	case "testnet":
		return aptos.TestnetConfig, nil
	case "mainnet":
		return aptos.MainnetConfig, nil
	default:
		return aptos.NetworkConfig{}, fmt.Errorf("Choose one of: mainnet, testnet, devnet, localnet")
	}

}
