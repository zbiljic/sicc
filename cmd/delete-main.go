package cmd

import (
	"errors"
	"fmt"
	"path"

	"github.com/spf13/cobra"

	"github.com/zbiljic/sicc/store"
)

// deleteCmd represents the 'delete' command
var deleteCmd = &cobra.Command{
	Use:   "delete <path>",
	Short: "Delete a configuration(s) from the system, including all versions",
	Args:  cobra.ExactArgs(1), //nolint:gomnd
	RunE:  runDelete,
}

var deleteParameters struct {
	Recursive bool
	Force     bool
	DryRun    bool
}

//nolint:lll
func init() {
	deleteCmd.Flags().BoolVar(&deleteParameters.Recursive, "recursive", false, "Delete recursively")
	deleteCmd.Flags().BoolVar(&deleteParameters.Force, "force", false, "Allow a recursive delete operation")
	deleteCmd.Flags().BoolVar(&deleteParameters.DryRun, "dryrun", false, "Display result of delete operation without actually removing configurations")
	// add 'delete' command to root command
	rootCmd.AddCommand(deleteCmd)
}

//nolint:funlen
func runDelete(cmd *cobra.Command, args []string) error {
	configPathName := path.Join(pathSeparator, args[0])

	if err := validateConfigPathName(configPathName); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	configStore, err := getConfigurationStore()
	if err != nil {
		return fmt.Errorf("failed to get configuration store: %w", err)
	}

	// check whether to delete single configuration

	_, err = getFromStore(configStore, configPathName)
	if err != nil {
		if !errors.Is(err, store.ErrConfigNotFound) {
			return fmt.Errorf("failed to fetch configuration: %w", err)
		}
	} else {
		fmt.Printf("Removing `%s`\n", configPathName)

		if !deleteParameters.DryRun {
			err = deleteFromStore(configStore, configPathName)
			if err != nil {
				return fmt.Errorf("failed to delete configuration `%s`: %w", configPathName, err)
			}
		}
	}

	// check to delete configurations recursively

	if !deleteParameters.Recursive {
		return nil
	}

	//nolint
	if !deleteParameters.Force {
		const ErrRecursiveDeleteOp = "Removal requires --force flag. This operation is *IRREVERSIBLE*. Please review carefully before performing this *DANGEROUS* operation."
		return errors.New(ErrRecursiveDeleteOp)
	}

	configs, err := configStore.List(configPathName, false)
	if err != nil {
		return fmt.Errorf("failed to list store contents (%s): %w", configPathName, err)
	}

	for _, config := range configs {
		key := config.Meta.Key

		fmt.Printf("Removing `%s`\n", key)

		if !deleteParameters.DryRun {
			// actually delete configuration
			err = deleteFromStore(configStore, key)
			if err != nil {
				return fmt.Errorf("failed to delete configuration `%s`: %w", key, err)
			}
		}
	}

	return nil
}

func parameterNameFromPath(configPath string) store.ParameterName {
	parameterPath, name := path.Split(configPath)

	return store.ParameterName{
		ParameterPath: parameterPath,
		Name:          name,
	}
}

func getFromStore(configStore store.Store, configPath string) (store.Value, error) {
	parameterName := parameterNameFromPath(configPath)
	return configStore.Get(parameterName, -1)
}

func deleteFromStore(configStore store.Store, configPath string) error {
	parameterName := parameterNameFromPath(configPath)
	return configStore.Delete(parameterName)
}
