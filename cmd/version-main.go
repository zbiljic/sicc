package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// versionCmd represents the 'version' command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE:  printVersion,
}

func init() {
	// add 'version' command to root command
	rootCmd.AddCommand(versionCmd)
}

func printVersion(cmd *cobra.Command, args []string) error {
	fmt.Fprintf(os.Stdout, "%s version %s\n", AppName, Version)
	return nil
}
