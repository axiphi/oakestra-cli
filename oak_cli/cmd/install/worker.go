package install

import (
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/charmbracelet/huh"
	"github.com/oakestra/oak-go-cli/internal/cliout"
	"github.com/oakestra/oak-go-cli/internal/elevate"
	"github.com/oakestra/oak-go-cli/internal/enact"
)

func DoInstallWorker(ctx context.Context, version string, autoConfirm bool) error {
	if !elevate.HasRootPrivileges() {
		elevationCmd, err := elevate.CurrentCommandWithRootElevation()
		if err != nil {
			return err
		}

		plan := &enact.Plan{
			Info: "This command requires root privileges",
			Goal: "re-execute it with root privileges",
			Steps: []enact.Step{
				enact.FuncStep{
					Name: fmt.Sprintf("Execute %q",
						strings.Join(elevationCmd.Args, " ")),
					Fn: func() error {
						return syscall.Exec(elevationCmd.Path, elevationCmd.Args, os.Environ())
					},
				},
			},
		}

		reExecuted, err := plan.Execute(autoConfirm)
		if err != nil {
			return err
		}
		if !reExecuted {
			return fmt.Errorf("root privileges required. Please run with sudo or as root")
		}

		// unreachable
		return nil
	}

	if err := checkPrerequisites(); err != nil {
		return err
	}

	osFamily := detectOSFamily()
	if osFamily == Unknown {
		cliout.Warnf("Unsupported OS detected. We cannot assist with third-party dependency installation.")
	}

	containerdReady, err := installContainerd(osFamily, autoConfirm)
	if err != nil {
		return err
	}
	if !containerdReady {
		cliout.Warnf("Containerd is not installed or enabled, disabling the containerd runtime of the Oakestra worker.")
		cliout.Warnf("Make sure to install and enable containerd on this machine yourself if you want to use it.")
	}

	if err := installNvidiaCTK(osFamily, autoConfirm); err != nil {
		return err
	}

	workerReady, err := installFirstParty(version, autoConfirm)
	if err != nil {
		return err
	}
	if !workerReady {
		cliout.Warnf("Oakestra worker was not installed.")
		return nil
	}

	if err := configureWorker(ctx); err != nil {
		cliout.Warnf("Failed to configure cluster for worker: %v", err)
	}

	if err := startWorker(autoConfirm); err != nil {
		cliout.Warnf("Failed to start worker: %v", err)
	}

	// completion hint
	cliout.Infof("%s Use %s to manage the worker node.", cliout.Green("✓ Worker node installed."), cliout.Bold("oak worker"))
	return nil
}

func startWorker(autoConfirm bool) error {
	if !autoConfirm {
		var confirmed = true
		err := huh.NewConfirm().
			Title("Start the worker node now?").
			Value(&confirmed).
			Run()

		if err != nil {
			return err
		}

		if !confirmed {
			return nil
		}
	}

	return startSystemdUnits()
}
