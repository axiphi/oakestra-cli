package install

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/oakestra/oak-go-cli/internal/cliout"
)

func DoInstallWorker(version string, autoConfirm bool) error {
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

	if err := configureWorker(); err != nil {
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
			Title(fmt.Sprintf("Start the worker node now?")).
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
