package cmd

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

// exportCmd represents the 'export' command
var exportCmd = &cobra.Command{
	Use:   "export <prefix...>",
	Short: "Exports parameters in the specified format",
	Args:  cobra.MinimumNArgs(1), //nolint:gomnd
	RunE:  runExport,
}

var exportParameters struct {
	Format string
	Output string
}

//nolint:lll
func init() {
	exportCmd.Flags().StringVarP(&exportParameters.Format, "format", "f", "json", "Output format (json, yaml, csv, tsv, dotenv, tfvars, tfenvvars)")
	exportCmd.Flags().StringVarP(&exportParameters.Output, "output-file", "o", "", "Output file (default is standard output)")
	// add 'export' command to root command
	rootCmd.AddCommand(exportCmd)
}

//nolint:funlen
func runExport(cmd *cobra.Command, args []string) error {
	var err error

	configStore, err := getConfigurationStore()
	if err != nil {
		return fmt.Errorf("failed to get configuration store: %w", err)
	}

	params := make(map[string]string)

	for _, arg := range args {
		prefixPath := path.Join("/", arg)

		if err := validateConfigPathName(prefixPath); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		rawValues, err := configStore.ListRaw(prefixPath)
		if err != nil {
			return fmt.Errorf("failed to list store contents (%s): %w", prefixPath, err)
		}

		for _, rawValue := range rawValues {
			k := stripPrefix(rawValue.Key, prefixPath)
			if _, ok := params[k]; ok {
				fmt.Fprintf(os.Stderr, "warning: parameter %s specified more than once (overridden by prefix %s)\n", k, prefixPath)
			}

			params[k] = rawValue.Value
		}
	}

	file := os.Stdout

	if exportParameters.Output != "" {
		if file, err = os.OpenFile(exportParameters.Output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
			return fmt.Errorf("to open output file for writing (%s): %w", exportParameters.Output, err)
		}
		defer file.Close()

		err = file.Sync()
		if err != nil {
			return fmt.Errorf("failed to save file changes: %w", err)
		}
	}

	w := bufio.NewWriter(file)
	defer w.Flush()

	switch strings.ToLower(exportParameters.Format) {
	case "json":
		err = exportAsJSON(params, w)
	case "yaml":
		err = exportAsYaml(params, w)
	case "csv":
		err = exportAsCsv(params, w)
	case "tsv":
		err = exportAsTsv(params, w)
	case "dotenv":
		err = exportAsEnvFile(params, w)
	case "tfvars":
		err = exportAsTfvars(params, w)
	case "tfenvvars":
		err = exportAsTfEnvVars(params, w)
	default:
		err = fmt.Errorf("unsupported export format: %s", exportParameters.Format)
	}

	if err != nil {
		return fmt.Errorf("unable to export parameters: %w", err)
	}

	return nil
}

func exportAsJSON(params map[string]string, w io.Writer) error {
	// JSON like:
	// {"root":{"param1": "value1","param2": "value2"}}
	jsonObj := gabs.New()

	for k, v := range params {
		hierarchy := strings.Split(k, pathSeparator)

		if _, err := jsonObj.Set(v, hierarchy...); err != nil {
			return fmt.Errorf("failed to set key %s to JSON: %w", k, err)
		}
	}

	fmt.Fprintln(w, jsonObj.String())

	return nil
}

func exportAsYaml(params map[string]string, w io.Writer) error {
	// YAML like:
	// root:
	//   param1: "value1"
	//   param2: "value2"
	jsonObj := gabs.New()

	for k, v := range params {
		hierarchy := strings.Split(k, pathSeparator)

		if _, err := jsonObj.Set(v, hierarchy...); err != nil {
			return fmt.Errorf("failed to set key %s to JSON: %w", k, err)
		}
	}

	d, err := yaml.Marshal(jsonObj.Data())
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	_, err = w.Write(d)
	if err != nil {
		return fmt.Errorf("failed to write bytes to Writer: %w", err)
	}

	return nil
}

func exportAsCsv(params map[string]string, w io.Writer) error {
	// CSV (Comma Separated Values) like:
	// param1,value1
	// param2,value2
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	for _, k := range sortedKeys(params) {
		if err := csvWriter.Write([]string{k, params[k]}); err != nil {
			return fmt.Errorf("failed to write param %s to CSV: %w", k, err)
		}
	}

	return nil
}

func exportAsTsv(params map[string]string, w io.Writer) error {
	// TSV (Tab Separated Values) like:
	for _, k := range sortedKeys(params) {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", k, params[k]); err != nil {
			return fmt.Errorf("failed to write param %s to TSV: %w", k, err)
		}
	}

	return nil
}

func exportAsEnvFile(params map[string]string, w io.Writer) error {
	// Env like:
	// KEY=val
	// OTHER=otherval
	for _, k := range sortedKeys(params) {
		key := strings.ToUpper(k)
		key = strings.ReplaceAll(key, "/", "_")
		key = strings.ReplaceAll(key, "-", "_")
		key = strings.ReplaceAll(key, ".", "_")

		_, err := w.Write([]byte(fmt.Sprintf(`%s="%s"`+"\n", key, doubleQuoteEscape(params[k]))))
		if err != nil {
			return fmt.Errorf("failed to write param %s: %w", k, err)
		}
	}

	return nil
}

func exportAsTfvars(params map[string]string, w io.Writer) error {
	// Terraform Variables is like dotenv, but keeps case
	for _, k := range sortedKeys(params) {
		key := strings.ReplaceAll(k, "/", "_")

		_, err := w.Write([]byte(fmt.Sprintf(`%s = "%s"`+"\n", key, doubleQuoteEscape(params[k]))))
		if err != nil {
			return fmt.Errorf("failed to write param %s: %w", k, err)
		}
	}

	return nil
}

func exportAsTfEnvVars(params map[string]string, w io.Writer) error {
	// Terraform Variables is like dotenv, but keeps the TF_VAR and keeps case
	for _, k := range sortedKeys(params) {
		key := "TF_VAR_" + k
		key = strings.ReplaceAll(key, "/", "_")
		key = strings.ReplaceAll(key, "-", "_")
		key = strings.ReplaceAll(key, ".", "_")

		_, err := w.Write([]byte(fmt.Sprintf(`%s="%s"`+"\n", key, doubleQuoteEscape(params[k]))))
		if err != nil {
			return fmt.Errorf("failed to write param %s: %w", k, err)
		}
	}

	return nil
}
