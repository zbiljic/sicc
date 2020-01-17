package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zbiljic/sicc/pkg/environ"
	"github.com/zbiljic/sicc/pkg/exec"
)

const (
	// Default value to expect in strict mode
	strictValueDefault = "changeme"
)

// execCmd represents the 'exec' command
var execCmd = &cobra.Command{
	Use:     "exec <prefix...> -- <command> [<arg...>]",
	Short:   "Executes a command with configurations loaded into the environment",
	Args:    cobra.MinimumNArgs(1), //nolint:gomnd
	PreRunE: checkExecSyntax,
	RunE:    runExec,
	//nolint:lll
	Example: `
Given a configuration store like this:

	$ echo '{"/prod/db/username": "admin", "/prod/db/password": "pass"}' | sicc import -

--strict will fail with unfilled env vars

	$ HOME=/tmp DB_USERNAME=changeme DB_PASSWORD=changeme EXTRA=changeme sicc exec --strict /prod exec -- env
	sicc: extra unfilled env var EXTRA
	exit 1

--pristine takes effect after checking for --strict values

	$ HOME=/tmp DB_USERNAME=changeme DB_PASSWORD=changeme sicc exec --strict --pristine /prod exec -- env
	DB_USERNAME=admin
	DB_PASSWORD=pass
`,
}

var execParameters struct {
	// When true, only use variables retrieved from the backend, do not inherit
	// existing environment variables
	Pristine bool

	// When true, enable strict mode, which checks that all secrets replace
	// env vars with a special sentinel value
	Strict bool

	// Value to expect in strict mode
	StrictValue string
}

//nolint:lll
func init() {
	execCmd.Flags().BoolVar(&execParameters.Pristine, "pristine", false,
		"Only use variables retrieved from the backend; do not inherit existing environment variables")
	execCmd.Flags().BoolVar(&execParameters.Strict, "strict", false,
		`Enable strict mode: only inject secrets for which there is a corresponding
env var with value <strict-value>, and fail if there are any env vars with
that value missing from secrets`)
	execCmd.Flags().StringVar(&execParameters.StrictValue, "strict-value", strictValueDefault,
		"Value to expect in --strict mode")
	// add 'exec' command to root command
	rootCmd.AddCommand(execCmd)
}

// checkExecSyntax - validate the passed arguments
func checkExecSyntax(cmd *cobra.Command, args []string) error {
	dashIx := cmd.ArgsLenAtDash()

	if dashIx == -1 {
		return errors.New("please separate prefix and command with '--'. See usage")
	}

	//nolint:gomnd
	if err := cobra.MinimumNArgs(1)(cmd, args[:dashIx]); err != nil {
		return fmt.Errorf("at least one prefix must be specified: %w. See usage", err)
	}

	//nolint:gomnd
	if err := cobra.MinimumNArgs(1)(cmd, args[dashIx:]); err != nil {
		return fmt.Errorf("must specify command to run: %w. See usage", err)
	}

	return nil
}

func runExec(cmd *cobra.Command, args []string) error {
	dashIx := cmd.ArgsLenAtDash()
	prefixPaths, command, commandArgs := args[:dashIx], args[dashIx], args[dashIx+1:]

	for i, p := range prefixPaths {
		prefixPaths[i] = path.Join("/", p)

		if err := validateConfigPathName(prefixPaths[i]); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	configStore, err := getConfigurationStore()
	if err != nil {
		return fmt.Errorf("failed to get configuration store: %w", err)
	}

	if execParameters.Pristine && globalVerbose {
		fmt.Fprintf(os.Stderr, "%s: pristine mode engaged\n", AppName)
	}

	var env environ.Environ

	if execParameters.Strict {
		if globalVerbose {
			fmt.Fprintf(os.Stderr, "%s: strict mode engaged\n", AppName)
		}

		env = environ.Environ(os.Environ())

		err := env.LoadStrict(configStore, execParameters.StrictValue, execParameters.Pristine, prefixPaths...)
		if err != nil {
			return err
		}
	} else {
		if !execParameters.Pristine {
			env = environ.Environ(os.Environ())
		}

		for _, prefixPath := range prefixPaths {
			collisions := make([]string, 0)

			err := env.Load(configStore, prefixPath, &collisions)
			if err != nil {
				return fmt.Errorf("failed to list store contents: %w", err)
			}

			for _, c := range collisions {
				fmt.Fprintf(os.Stderr, "warning: configuration %s overwriting environment variable %s\n", prefixPath, c)
			}
		}
	}

	if globalVerbose {
		fmt.Fprintf(os.Stdout, "info: With environment %s\n", strings.Join(env, ","))
	}

	return exec.Exec(command, commandArgs, env)
}
