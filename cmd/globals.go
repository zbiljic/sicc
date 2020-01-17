package cmd

import (
	"os"
	"strconv"
)

const (
	globalErrorExitStatus = 1 // Global error exit status.
)

const (
	// pathSeparator is separeator used for parameter name
	pathSeparator = "/"

	// shortTimeFormat is a short format for printing timestamps
	shortTimeFormat = "2006-01-02 15:04:05"

	// defaultNumRetries is the default for the number of retries we'll use for our AWS client
	defaultNumRetries = 10
)

var (
	globalVerbose    = false             // Verbose flag set via command line
	globalBackend    = "ssm"             // Backend flag set via command line
	globalNumRetries = defaultNumRetries // Retries flag set via command line
	// WHEN YOU ADD NEXT GLOBAL FLAG, MAKE SURE TO ALSO UPDATE PERSISTENT FLAGS, FLAG CONSTANTS AND UPDATE FUNC.
)

const (
	verboseEnvVar = "SICC_VERBOSE"
	backendEnvVar = "SICC_BACKEND"
	retriesEnvVar = "SICC_RETRIES"
)

func updateGlobals() {
	if verbose, ok := os.LookupEnv(verboseEnvVar); ok {
		globalVerbose, _ = strconv.ParseBool(verbose)
	}

	if backend, ok := os.LookupEnv(backendEnvVar); ok {
		globalBackend = backend
	}

	if retries, ok := os.LookupEnv(retriesEnvVar); ok {
		globalNumRetries, _ = strconv.Atoi(retries)
	}
}
