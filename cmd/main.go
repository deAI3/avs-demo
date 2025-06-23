package main

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"avs/aggregator"
	operator "avs/operator"
)

func main() {
	zLogger, _ := zap.NewProduction(zap.AddStacktrace(zap.DPanicLevel))
	defer zLogger.Sync()

	logger := zLogger.Sugar()

	rootCmd := &cobra.Command{
		Use:   "avs [command]",
		Short: "command for avs",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.AddCommand(
		operator.OperatorCommand(zLogger),
		aggregator.AggregatorCommand(zLogger),
	)

	err := rootCmd.Execute()
	if err != nil {
		logger.Fatal(err)
	}
}
