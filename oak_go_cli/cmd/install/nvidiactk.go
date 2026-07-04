package install

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/oakestra/oak-go-cli/internal/cliout"
	"github.com/oakestra/oak-go-cli/internal/download"
	"github.com/oakestra/oak-go-cli/internal/enact"
	"github.com/oakestra/oak-go-cli/internal/iotools"
)

func installNvidiaCTK(osFamily OSFamily, autoConfirm bool) error {
	// nvidia-smi for standard GPUs, tegrastats for Jetson GPUs
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		if _, err := exec.LookPath("tegrastats"); err != nil {
			cliout.Infof("No NVIDIA GPU detected, skipping NVIDIA Container Toolkit installation.")
			return nil
		}
	}

	if _, err := exec.LookPath("nvidia-ctk"); err == nil {
		plan := &enact.Plan{
			Info:  "Detected existing installation of NVIDIA Container Toolkit",
			Goal:  "configure it",
			Steps: getNvidiaCTKConfigureCommands(),
		}

		_, err := plan.Execute(autoConfirm)
		return err
	}

	if ready, err := ensureNvidiaCTKPrerequisites(osFamily, autoConfirm); err != nil || !ready {
		return err
	}

	installSteps := getNvidiaCTKInstallCommands(osFamily)
	if installSteps == nil {
		cliout.Infof("Automatic installation of NVIDIA Container Toolkit not supported for the current OS, skipping.")
		return nil
	}

	plan := &enact.Plan{
		Info:  "No existing installation of NVIDIA Container Toolkit was detected",
		Goal:  "install it",
		Steps: installSteps,
	}

	_, err := plan.Execute(autoConfirm)
	return err
}

func ensureNvidiaCTKPrerequisites(osFamily OSFamily, autoConfirm bool) (bool, error) {
	if osFamily != Apt {
		return true, nil
	}

	if _, err := exec.LookPath("gpg"); err == nil {
		return true, nil
	}

	plan := &enact.Plan{
		Info: "Missing gpg command",
		Goal: "install it",
		Steps: []enact.Step{
			enact.CommandStep{
				Command: "apt-get",
				Args:    []string{"update"},
			},
			enact.CommandStep{
				Command: "apt-get",
				Args:    []string{"install", "-y", "gnupg"},
			},
		},
	}

	return plan.Execute(autoConfirm)
}

func getNvidiaCTKConfigureCommands() []enact.Step {
	return []enact.Step{
		enact.CommandStep{
			Command: "nvidia-ctk",
			Args:    []string{"runtime", "configure", "--runtime=containerd"},
		},
		enact.CommandStep{
			Command: "systemctl",
			Args:    []string{"restart", "containerd"},
		},
	}
}

func getNvidiaCTKInstallCommands(osFamily OSFamily) []enact.Step {
	switch osFamily {
	case Apt:
		return []enact.Step{
			enact.FuncStep{
				Name: "Download NVIDIA GPG Key",
				Fn: func() error {
					return download.GetHttpBodyToFile("https://nvidia.github.io/libnvidia-container/gpgkey", "/tmp/nvidia.pub", 0o644)
				},
			},
			enact.CommandStep{
				Command: "gpg",
				Args:    []string{"--dearmor", "-o", "/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg", "/tmp/nvidia.pub"},
			},
			enact.FuncStep{
				Name: "Configure NVIDIA APT Repository List",
				Fn: func() error {
					err := download.GetHttpBodyToFile(
						"https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list",
						"/etc/apt/sources.list.d/nvidia-container-toolkit.list",
						0o644,
					)
					if err != nil {
						return err
					}

					file, err := os.OpenFile("/etc/apt/sources.list.d/nvidia-container-toolkit.list", os.O_RDWR, 0o644)
					if err != nil {
						return err
					}
					defer iotools.CloseOrWarn(file, "/etc/apt/sources.list.d/nvidia-container-toolkit.list")

					content, err := io.ReadAll(file)
					if err != nil {
						return err
					}

					modifiedContent := strings.ReplaceAll(
						string(content),
						"deb https://",
						"deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://",
					)

					if err := file.Truncate(0); err != nil {
						return err
					}
					if _, err := file.Seek(0, io.SeekStart); err != nil {
						return err
					}

					_, err = io.Copy(file, strings.NewReader(modifiedContent))
					return err
				},
			},
			enact.CommandStep{
				Command: "apt-get",
				Args:    []string{"update"},
			},
			enact.CommandStep{
				Command: "apt-get",
				Args:    []string{"install", "-y", "nvidia-container-toolkit"},
			},
			enact.CommandStep{
				Command: "nvidia-ctk",
				Args:    []string{"runtime", "configure", "--runtime=containerd"},
			},
			enact.CommandStep{
				Command: "systemctl",
				Args:    []string{"restart", "containerd"},
			},
		}
	case Dnf:
		return []enact.Step{
			enact.FuncStep{
				Name: "Download NVIDIA Toolkit Repository",
				Fn: func() error {
					return download.GetHttpBodyToFile(
						"https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo",
						"/etc/yum.repos.d/nvidia-container-toolkit.repo",
						0o644,
					)
				},
			},
			enact.CommandStep{
				Command: "dnf",
				Args:    []string{"install", "-y", "nvidia-container-toolkit"},
			},
			enact.CommandStep{
				Command: "nvidia-ctk",
				Args:    []string{"runtime", "configure", "--runtime=containerd"},
			},
			enact.CommandStep{
				Command: "systemctl",
				Args:    []string{"restart", "containerd"},
			},
		}
	}
	return nil
}
