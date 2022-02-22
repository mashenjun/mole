package main

import (
	"github.com/pingcap/log"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	"os"
)

func init() {
	if os.Getenv("DEBUG") == "1" {
		log.SetLevel(zapcore.DebugLevel)
	}
}

func main() {
	if err := command().Execute(); err != nil {
		os.Exit(-1)
	}
}

func command() *cobra.Command {
	cmd := &cobra.Command{
		Use:  `mole`,
		Long: `mole is a command-line to collect information and data masking`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage: true,
	}
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.SilenceErrors = false
	cmd.AddCommand(
		schemaCmd(),
		metricsCmd(),
		keyvizCmd(),
		convertCmd(),
		reshapeCmd(),
		splitCmd(),
		rebuildCmd(),
	)
	return cmd
}
