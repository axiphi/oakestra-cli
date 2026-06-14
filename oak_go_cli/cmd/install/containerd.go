package install

import (
	"os/exec"

	"github.com/oakestra/oak-go-cli/internal/cliout"
	"github.com/oakestra/oak-go-cli/internal/enact"
)

func installContainerd(osFamily OSFamily, autoConfirm bool) (bool, error) {
	if err := exec.Command("systemctl", "is-active", "--quiet", "containerd").Run(); err == nil {
		cliout.Infof("Existing containerd installation is already running.")
		return true, nil
	}

	// Check if containerd is installed but inactive
	if err := exec.Command("systemctl", "list-unit-files", "containerd.service").Run(); err == nil {
		plan := &enact.Plan{
			Info:  "Existing containerd installation is not running",
			Goal:  "start and enable it",
			Steps: getContainerdEnableSteps(),
		}

		return plan.Execute(autoConfirm)
	}

	installSteps := getContainerdInstallSteps(osFamily)
	if installSteps == nil {
		cliout.Infof("Automatic installation of containerd is not supported for the current OS, skipping.")
		return false, nil
	}

	plan := &enact.Plan{
		Info:  "No existing installation of containerd detected",
		Goal:  "install it",
		Steps: installSteps,
	}

	return plan.Execute(autoConfirm)
}

func getContainerdEnableSteps() []enact.Step {
	return []enact.Step{
		enact.CommandStep{
			Command: "systemctl",
			Args:    []string{"enable", "--now", "containerd"},
		},
	}
}

func getContainerdInstallSteps(osFamily OSFamily) []enact.Step {
	switch osFamily {
	case Apt:
		return []enact.Step{
			enact.CommandStep{
				Command: "apt-get",
				Args:    []string{"update"},
			},
			enact.CommandStep{
				Command: "apt-get",
				Args:    []string{"install", "-y", "containerd"},
			},
			enact.CommandStep{
				Command: "systemctl",
				Args:    []string{"enable", "--now", "containerd"},
			},
		}
	case Dnf:
		return []enact.Step{
			enact.CommandStep{
				Command: "dnf",
				Args:    []string{"install", "-y", "containerd"},
			},
			enact.CommandStep{
				Command: "systemctl",
				Args:    []string{"enable", "--now", "containerd"},
			},
		}
	}
	return nil
}
