package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zbiljic/sicc/store"
)

// putCmd represents the 'put' command
var putCmd = &cobra.Command{
	Use:   "put <path> [--] <value|->",
	Short: "Put configuration to the system",
	Args:  cobra.ExactArgs(2), //nolint:gomnd
	RunE:  runPut,
}

var putParameters struct {
	Secret     bool
	Singleline bool
}

//nolint:lll
func init() {
	putCmd.Flags().BoolVar(&putParameters.Secret, "secret", false, "Add configuration as secret value")
	putCmd.Flags().BoolVarP(&putParameters.Singleline, "singleline", "s", false, "Insert single line parameter (end with \\n)")
	// add 'put' command to root command
	rootCmd.AddCommand(putCmd)
}

func runPut(cmd *cobra.Command, args []string) error {
	configPathName := path.Join(pathSeparator, args[0])

	if err := validateConfigPathName(configPathName); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	value := args[1]
	if value == "-" {
		// Read value from standard input
		if putParameters.Singleline {
			buf := bufio.NewReader(os.Stdin)

			v, err := buf.ReadString('\n')
			if err != nil {
				return err
			}

			value = strings.TrimSuffix(v, "\n")
		} else {
			v, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				return err
			}

			value = string(v)
		}
	}

	val := store.Value{
		Value: &value,
		Meta: store.Metadata{
			Secure: putParameters.Secret,
		},
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

	// Skip writing configuration if value is unchanged
	currentConfig, err := configStore.Get(parameterName, -1)
	if err == nil && value == *currentConfig.Value && currentConfig.Meta.Secure == val.Meta.Secure {
		return nil
	}

	return configStore.Put(parameterName, val)
}
