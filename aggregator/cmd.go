package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	flagStream              = "streamming"
	flagAptosNetwork        = "aptos-network"
	flagAggregatorConfig    = "aggregator-config"
	flagAggregatorStorePath = "aggregator-store-path"
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
			aggregatorConfigPath, err := cmd.Flags().GetString(flagAggregatorConfig)
			if err != nil {
				return errors.Wrap(err, flagAggregatorConfig)
			}

			aggregatorConfig, err := loadAggregatorConfig(aggregatorConfigPath)
			if err != nil {
				return fmt.Errorf("can not load aggregator config: %s", err)
			}

			aggregator, err := NewAggregator(*aggregatorConfig, logger)
			if err != nil {
				logger.Error("Cannot create aggregator", zap.Any("err", err))
				return err
			}

			err = aggregator.Start(context.Background())
			if err != nil {
				logger.Error("Cannot start aggregator", zap.Any("err", err))
				return err
			}
			return nil
		},
	}
	cmd.Flags().String(flagAggregatorConfig, "config/aggregator-config.json", "see the example at config/aggregator-example.json")
	return cmd
}

func CreateAggregatorConfig(logger *zap.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "config",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			aggregatorStorePath, err := cmd.Flags().GetString(flagAggregatorStorePath)
			if err != nil {
				return errors.Wrap(err, flagAggregatorConfig)
			}

			aggregatorConfigPath, err := cmd.Flags().GetString(flagAggregatorConfig)
			if err != nil {
				return errors.Wrap(err, flagAggregatorConfig)
			}

			portAddr := args[1]
			aggregatorConfig := AggregatorConfig{
				ServerIpPortAddress: portAddr,
				StorePath:           aggregatorStorePath,
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
	cmd.Flags().String(flagAggregatorStorePath, ".aggregator", "default for the store path")
	cmd.Flags().String(flagAggregatorConfig, "config/aggregator-config.json", "path for the config file of aggregator")
	return cmd
}

func RequestTask(logger *zap.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "request-task",
		Short: "request task from aggregator",
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Requesting task from aggregator...")
			aggregatorConfigPath, err := cmd.Flags().GetString(flagAggregatorConfig)
			if err != nil {
				return errors.Wrap(err, flagAggregatorConfig)
			}

			aggregatorConfig, err := loadAggregatorConfig(aggregatorConfigPath)
			if err != nil {
				return fmt.Errorf("can not load aggregator config: %s", err)
			}

			aggregator, err := NewAggregator(*aggregatorConfig, logger)
			if err != nil {
				logger.Error("Cannot create aggregator", zap.Any("err", err))
				return err
			}

			model := args[0]
			stream, err := cmd.Flags().GetBool(flagStream)
			if err != nil {
				return errors.Wrap(err, flagStream)
			}

			if (len(args)-1)%2 != 0 {
				return errors.New("args length after index 1 is not even")
			}

			messages := make([]ChatMessage, 0, (len(args)-1)/2)
			for i := 1; i < len(args); i += 2 {
				messages = append(messages, ChatMessage{
					Role:    args[i],
					Content: args[i+1],
				})
			}
			aggregator.NewTask(ChatCompletionRequest{
				Model:    model,
				Messages: messages,
				Stream:   stream,
			})
			return nil
		},
	}
	cmd.Flags().String(flagAggregatorConfig, "config/aggregator-config.json", "see the example at config/aggregator-example.json")
	cmd.Flags().Bool(flagStream, false, "using streaming mode to request model or not")
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
	bytes, err := io.ReadAll(file)
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
