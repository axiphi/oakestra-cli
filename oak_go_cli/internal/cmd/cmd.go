package cmd

import (
	"os"
	"os/exec"
)

func RunInteractive(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func RunForwarded(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func RunSilent(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func RunCaptured(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	return cmd.Output()
}
