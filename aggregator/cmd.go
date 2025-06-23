package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	flagAptosNetwork          = "aptos-network"
	flagAggregatorConfig      = "aggregator-config"
	flagAggregatorAccountPath = "aggregator-account"
)

func AggregatorCommand(zLogger *zap.Logger) *cobra.Command {
	aggregatorCmd := &cobra.Command{
		Use:   "aggregator",
		Short: "aggregator command for avs",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// // Add operator-specific subcommands here
	aggregatorCmd.AddCommand(
		Start(zLogger), // Example: 'operator start'
		CreateAggregatorConfig(zLogger),
	)

	return aggregatorCmd
}

func Start(logger *zap.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "start",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get all the flags
			network, err := cmd.Flags().GetString(flagAptosNetwork)
			if err != nil {
				return errors.Wrap(err, flagAptosNetwork)
			}
			aggregatorConfigPath, err := cmd.Flags().GetString(flagAggregatorConfig)
			if err != nil {
				return errors.Wrap(err, flagAggregatorConfig)
			}

			aggregatorConfig, err := loadAggregatorConfig(aggregatorConfigPath)
			if err != nil {
				return fmt.Errorf("can not load aggregator config: %s", err)
			}

			networkConfig, err := extractNetwork(network)
			if err != nil {
				return fmt.Errorf("wrong config: %s", err)
			}

			aggregator, err := NewAggregator(*aggregatorConfig, logger, networkConfig)
			if err != nil {
				logger.Error("Cannot create aggregator", zap.Any("err", err))
				return err
			}

			// // Listen for new task created in the ServiceManager contract in a separate goroutine
			// go func() {
			// 	listenErr := aggregator.SubscribeToNewTasks(networkConfig)
			// 	if listenErr != nil {
			// 		aggregator.logger.Fatal("Error subscribing for new tasks", zap.Any("err", listenErr))
			// 	}
			// }()

			err = aggregator.Start(context.Background())
			if err != nil {
				logger.Error("Cannot start aggregator", zap.Any("err", err))
				return err
			}
			return nil
			// client.SubmitTransaction()
		},
	}
	cmd.Flags().String(flagAggregatorConfig, "config/aggregator-config.json", "see the example at config/aggregator-example.json")
	cmd.Flags().String(flagAptosNetwork, "devnet", "choose network to connect to: mainnet, testnet, devnet, localnet")
	return cmd
}

func CreateAggregatorConfig(logger *zap.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "config",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			aggregatorAccountPath, err := cmd.Flags().GetString(flagAggregatorAccountPath)
			if err != nil {
				return errors.Wrap(err, flagAggregatorConfig)
			}

			aggregatorConfigPath, err := cmd.Flags().GetString(flagAggregatorConfig)
			if err != nil {
				return errors.Wrap(err, flagAggregatorConfig)
			}

			avsAddress := aptos.AccountAddress{}
			if err := avsAddress.ParseStringRelaxed(args[0]); err != nil {
				return fmt.Errorf("failed to parse avs address: %s", err)
			}

			portAddr := args[1]
			aggregatorConfig := AggregatorConfig{
				ServerIpPortAddress: portAddr,
				AvsAddress:          avsAddress.String(),
				AccountConfig: AccountConfig{
					AccountPath: aggregatorAccountPath,
					Profile:     "aggregator",
				},
			}
			bz, err := json.Marshal(aggregatorConfig)
			if err != nil {
				return fmt.Errorf("failed to marshal operator config: %s", err)
			}

			f, err := os.Create(aggregatorConfigPath)
			if err != nil {
				return fmt.Errorf("failed to create file at %s: %s", aggregatorConfigPath, err)
			}
			_, err = f.WriteString(string(bz))
			if err != nil {
				return fmt.Errorf("failed to write to file at %s: %s", aggregatorConfigPath, err)
			}

			return nil
		},
	}
	cmd.Flags().String(flagAggregatorAccountPath, ".aptos/config.yaml", "default for the account derivation")
	cmd.Flags().String(flagAggregatorConfig, "config/aggregator-config.json", "path for the config file of aggregator")
	return cmd
}

func loadAggregatorConfig(filename string) (*AggregatorConfig, error) {
	// Open the config file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %v", err)
	}
	defer file.Close()

	// Read the file contents
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	// Unmarshal the JSON data into the Config struct
	var config AggregatorConfig
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
		return aptos.NetworkConfig{}, fmt.Errorf("choose one of: mainnet, testnet, devnet, localnet")
	}
}
