package cmd

import (
	"fmt"
	"os"
	"path"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/zbiljic/sicc/store"
)

// getCmd represents the 'get' command
var getCmd = &cobra.Command{
	Use:   "get <path>",
	Short: "Get configuration from the system",
	Args:  cobra.ExactArgs(1), //nolint:gomnd
	RunE:  runGet,
}

var getParameters struct {
	Version int
	Quiet   bool
}

//nolint:lll
func init() {
	getCmd.Flags().IntVarP(&getParameters.Version, "version", "v", -1, "The version number of the secret. Defaults to latest.")
	getCmd.Flags().BoolVarP(&getParameters.Quiet, "quiet", "q", false, "Only print the secret")
	// add 'get' command to root command
	rootCmd.AddCommand(getCmd)
}

func runGet(cmd *cobra.Command, args []string) error {
	configPathName := path.Join(pathSeparator, args[0])

	if err := validateConfigPathName(configPathName); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	configStore, err := getConfigurationStore()
	if err != nil {
		return fmt.Errorf("failed to get configuration store: %w", err)
	}

	path, name := path.Split(configPathName)

	parameterName := store.ParameterName{
		ParameterPath: path,
		Name:          name,
	}

	config, err := configStore.Get(parameterName, getParameters.Version)
	if err != nil {
		return fmt.Errorf("failed to fetch configuration: %w", err)
	}

	if getParameters.Quiet {
		fmt.Fprintf(os.Stdout, "%s\n", *config.Value)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)

	fmt.Fprintln(w, "Key\tValue\tVersion\tSecure\tLastModified\tUser")
	fmt.Fprintf(w, "%s\t%s\t%d\t%t\t%s\t%s\n",
		config.Meta.Key,
		*config.Value,
		config.Meta.Version,
		config.Meta.Secure,
		config.Meta.LastModifiedDate.Local().Format(shortTimeFormat),
		config.Meta.LastModifiedUser,
	)

	w.Flush()

	return nil
}
