package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zbiljic/sicc/store"
)

// AppName - the name of the application.
const AppName = "sicc"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:               AppName,
	Short:             "CLI for managing configurations",
	SilenceUsage:      true,
	PersistentPreRunE: registerBefore,
}

//nolint:lll
func init() {
	rootCmd.PersistentFlags().BoolVarP(&globalVerbose, "verbose", "", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&globalBackend, "backend", "b", "ssm", `Backend to use
	null: no-op
	ssm: SSM Parameter Store
`)
	rootCmd.PersistentFlags().IntVarP(&globalNumRetries, "retries", "r", defaultNumRetries,
		"For SSM, the number of retries to make before giving up")
}

func registerBefore(cmd *cobra.Command, args []string) error {
	// Update global flags (if anything changed from other sources).
	updateGlobals()

	return nil
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if cmd, err := rootCmd.ExecuteC(); err != nil {
		if strings.Contains(err.Error(), "arg(s)") || strings.Contains(err.Error(), "usage") {
			cmd.Usage() //nolint:errcheck
		}

		os.Exit(globalErrorExitStatus)
	}
}

func getConfigurationStore() (store.Store, error) {
	backend := strings.ToLower(globalBackend)

	var (
		s   store.Store
		err error
	)

	switch backend {
	case "null":
		s = store.NewNullStore()
	case "ssm":
		s, err = store.NewSSMStore(globalNumRetries)
	default:
		return nil, fmt.Errorf("invalid backend `%s`", backend)
	}

	return s, err
}
