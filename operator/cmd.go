package operator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/aptos-labs/aptos-go-sdk/crypto"
)

const (
	flagAptosNetwork      = "aptos-network"
	flagAptosConfigPath   = "aptos-config"
	flagAvsOperatorConfig = "avs-operator-config"
	flagAccountProfile    = "account-profile"
)

func OperatorCommand(zLogger *zap.Logger) *cobra.Command {
	operatorCmd := &cobra.Command{
		Use:   "operator",
		Short: "operator command for avs",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Add operator-specific subcommands here
	operatorCmd.AddCommand(
		Start(zLogger),                // Example: 'operator start'
		CreateOperatorConfig(zLogger), // Example: 'operator create-key'
		Deregister(zLogger),           // Example: 'operator deregister'
		QueryPrice(zLogger),           // Example: 'operator price')
	)

	return operatorCmd
}

func QueryPrice(logger *zap.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "price",
		Short: "price",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			symbol := args[0]
			getCMCPrice(symbol)
			return nil
		},
	}
	cmd.Flags().String(flagAvsOperatorConfig, "config/operator-config.json", "see the example at config/example.json")
	return cmd
}

func CreateOperatorConfig(logger *zap.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "config",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			operatorConfigPath, err := cmd.Flags().GetString(flagAvsOperatorConfig)
			if err != nil {
				return errors.Wrap(err, flagAvsOperatorConfig)
			}

			privKey, err := crypto.GenerateBlsPrivateKey()
			if err != nil {
				return fmt.Errorf("unable to generate bls keys: %s", err)
			}

			portAddr := args[0]
			operatorConfig := OperatorConfig{
				BlsPrivateKey:        privKey.Inner.Marshal(),
				AggregatorIpPortAddr: portAddr,
			}
			bz, err := json.Marshal(operatorConfig)
			if err != nil {
				return fmt.Errorf("failed to marshal operator config: %s", err)
			}

			f, err := os.Create(operatorConfigPath)
			if err != nil {
				return fmt.Errorf("failed to create file at %s: %s", operatorConfigPath, err)
			}
			_, err = f.WriteString(string(bz))
			if err != nil {
				return fmt.Errorf("failed to write to file at %s: %s", operatorConfigPath, err)
			}

			return nil
		},
	}
	cmd.Flags().String(flagAvsOperatorConfig, "config/operator-config.json", "see the example at config/example.json")
	return cmd
}

// func InitializeQuorum(logger *zap.Logger) *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "initialize-quorum",
// 		Short: "initialize-quorum",
// 		Args:  cobra.ExactArgs(2),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			aptosPath, err := cmd.Flags().GetString(flagAptosConfigPath)
// 			if err != nil {
// 				return errors.Wrap(err, flagAptosConfigPath)
// 			}
// 			accountProfile, err := cmd.Flags().GetString(flagAccountProfile)
// 			if err != nil {
// 				return errors.Wrap(err, flagAccountProfile)
// 			}
// 			network, err := cmd.Flags().GetString(flagAptosNetwork)
// 			if err != nil {
// 				return errors.Wrap(err, flagAptosNetwork)
// 			}
// 			operatorConfigPath, err := cmd.Flags().GetString(flagAvsOperatorConfig)
// 			if err != nil {
// 				return errors.Wrap(err, flagAvsOperatorConfig)
// 			}

// 			networkConfig, err := extractNetwork(network)
// 			if err != nil {
// 				return fmt.Errorf("wrong config: %s", err)
// 			}
// 			operatorConfig, err := loadOperatorConfig(operatorConfigPath)
// 			if err != nil {
// 				return fmt.Errorf("can not load operator config: %s", err)
// 			}

// 			maxOperatorCount, err := strconv.ParseUint(args[0], 10, 32)
// 			if err != nil {
// 				return fmt.Errorf("can not parse max operator count: %s", err)
// 			}

// 			minimumStake := new(big.Int)
// 			_, success := minimumStake.SetString(args[1], 10)
// 			if !success {
// 				return fmt.Errorf("can not parse minimum stake: %s", err)
// 			}

// 			err = InitQuorum(
// 				networkConfig,
// 				*operatorConfig,
// 				AptosAccountConfig{
// 					configPath: aptosPath,
// 					profile:    accountProfile,
// 				},
// 				uint32(maxOperatorCount),
// 				*minimumStake,
// 			)
// 			return err
// 		},
// 	}
// 	cmd.Flags().String(flagAptosConfigPath, ".aptos/config.yaml", "the path to your operator priv and pub key")
// 	cmd.Flags().String(flagAccountProfile, "default", "the account profile to use")
// 	cmd.Flags().String(flagAptosNetwork, "devnet", "choose network to connect to: mainnet, testnet, devnet, localnet")
// 	cmd.Flags().String(flagAvsOperatorConfig, "config/operator-config.json", "see the example at config/example.json")
// 	return cmd
// }

func Deregister(logger *zap.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deregister",
		Short: "deregister",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			aptosPath, err := cmd.Flags().GetString(flagAptosConfigPath)
			if err != nil {
				return errors.Wrap(err, flagAptosConfigPath)
			}
			accountProfile, err := cmd.Flags().GetString(flagAccountProfile)
			if err != nil {
				return errors.Wrap(err, flagAccountProfile)
			}
			operatorConfigPath, err := cmd.Flags().GetString(flagAvsOperatorConfig)
			if err != nil {
				return errors.Wrap(err, flagAvsOperatorConfig)
			}

			operatorConfig, err := loadOperatorConfig(operatorConfigPath)
			if err != nil {
				return fmt.Errorf("can not load operator config: %s", err)
			}

			operatorAccount, err := SignerFromConfig(aptosPath, accountProfile)
			if err != nil {
				panic("Failed to create operator account:" + err.Error())
			}

			aggClient, err := NewAggregatorRpcClient(operatorConfig.AggregatorIpPortAddr)
			if err != nil {
				return fmt.Errorf("can not create new aggregator Rpc Client: %v", err)
			}

			aggClient.SendDeregisterOperatorRequest(operatorAccount.PubKey().Bytes())
			return nil
		},
	}
	cmd.Flags().String(flagAptosConfigPath, ".aptos/config.yaml", "the path to your operator priv and pub key")
	cmd.Flags().String(flagAccountProfile, "default", "the account profile to use")
	cmd.Flags().String(flagAvsOperatorConfig, "config/operator-config.json", "see the example at config/example.json")
	return cmd
}

func Start(logger *zap.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "start",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get all the flags
			aptosPath, err := cmd.Flags().GetString(flagAptosConfigPath)
			if err != nil {
				return errors.Wrap(err, flagAptosConfigPath)
			}
			accountProfile, err := cmd.Flags().GetString(flagAccountProfile)
			if err != nil {
				return errors.Wrap(err, flagAccountProfile)
			}
			operatorConfigPath, err := cmd.Flags().GetString(flagAvsOperatorConfig)
			if err != nil {
				return errors.Wrap(err, flagAvsOperatorConfig)
			}

			operatorConfig, err := loadOperatorConfig(operatorConfigPath)
			if err != nil {
				return fmt.Errorf("can not load operator config: %s", err)
			}

			operator, err := NewOperator(
				logger,
				*operatorConfig,
				AptosAccountConfig{
					configPath: aptosPath,
					profile:    accountProfile,
				},
				operatorConfig.BlsPrivateKey,
			)
			if err != nil {
				return fmt.Errorf("can not create new operator: %s", err)
			}

			operator.Start(context.Background())

			return nil
			// client.SubmitTransaction()
		},
	}
	cmd.Flags().String(flagAptosConfigPath, ".aptos/config.yaml", "the path to your operator priv and pub key")
	cmd.Flags().String(flagAccountProfile, "default", "the account profile to use")
	cmd.Flags().String(flagAvsOperatorConfig, "config/operator-config.json", "see the example at config/example.json")
	return cmd
}
