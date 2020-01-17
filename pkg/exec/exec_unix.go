// +build linux darwin

package exec

import (
	osexec "os/exec"
	"syscall"
)

func Exec(command string, args []string, env []string) error {
	argv0, err := osexec.LookPath(command)
	if err != nil {
		return err
	}

	argv := make([]string, 0, 1+len(args)) //nolint:gomnd
	argv = append(argv, command)
	argv = append(argv, args...)

	// Only return if the execution fails.
	return syscall.Exec(argv0, argv, env)
}
