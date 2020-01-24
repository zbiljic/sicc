package cmd

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/jeremywohl/flatten"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	utilyaml "github.com/zbiljic/sicc/pkg/util/yaml"
	"github.com/zbiljic/sicc/store"
)

// importCmd represents the 'import' command
var importCmd = &cobra.Command{
	Use:   "import <path> <file|->",
	Short: "Import configurations from file",
	Args:  cobra.ExactArgs(2), //nolint:gomnd
	RunE:  runImport,
}

var importParameters struct {
	Secret bool
}

func init() {
	importCmd.Flags().BoolVar(&importParameters.Secret, "secret", false, "Add configurations as secrets")
	// add 'import' command to root command
	rootCmd.AddCommand(importCmd)
}

//nolint:funlen
func runImport(cmd *cobra.Command, args []string) error {
	configPathName := path.Join(pathSeparator, args[0])

	if err := validateConfigPathName(configPathName); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	var in io.Reader

	file := args[1]
	if file == "-" {
		in = os.Stdin
	} else {
		var err error
		in, err = os.Open(file)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
	}

	var data map[string]interface{}

	if err := utilyaml.NewYAMLOrJSONDecoder(in, 16).Decode(&data); err != nil {
		return fmt.Errorf("failed to decode input as JSON or YAML: %w", err)
	}

	toBeImported, err := flatten.Flatten(data, "", flatten.PathStyle)
	if err != nil {
		return fmt.Errorf("failed to flatten input: %w", err)
	}

	configStore, err := getConfigurationStore()
	if err != nil {
		return fmt.Errorf("failed to get configuration store: %w", err)
	}

	importedCount := 0

	for key, value := range toBeImported {
		parameterName := store.ParameterName{
			ParameterPath: configPathName,
			Name:          key,
		}

		v := cast.ToString(value)

		if v != "" {
			val := store.Value{
				Value: &v,
				Meta: store.Metadata{
					Secure: importParameters.Secret,
				},
			}

			// Skip writing configuration if value is unchanged
			currentConfig, err := configStore.Get(parameterName, -1)
			if err == nil && value == *currentConfig.Value && currentConfig.Meta.Secure == val.Meta.Secure {
				continue
			}

			fmt.Printf("Importing `%s`\n", path.Join(configPathName, key))

			if err := configStore.Put(parameterName, val); err != nil {
				return fmt.Errorf("failed to write configuration `%s`: %w", v, err)
			}

			importedCount++
		}
	}

	fmt.Fprintf(os.Stdout, "Successfully imported %d/%d configurations\n", importedCount, len(toBeImported))

	return nil
}
