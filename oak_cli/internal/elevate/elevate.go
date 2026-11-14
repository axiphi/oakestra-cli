package elevate

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func HasRootPrivileges() bool {
	return os.Geteuid() == 0
}

func CurrentCommandWithRootElevation() (*exec.Cmd, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable: %w", err)
	}

	for _, tool := range []string{"run0", "pkexec", "sudo", "doas"} {
		path, err := exec.LookPath(tool)
		if err != nil {
			continue
		}

		cmd := exec.Command(
			path,
			append([]string{exe}, os.Args[1:]...)...,
		)

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd, nil
	}

	return nil, errors.New(
		"root privileges required, but none of run0, pkexec, sudo or doas is available",
	)
}
