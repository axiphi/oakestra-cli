package install

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/oakestra/oak-go-cli/internal/cliout"
	"github.com/oakestra/oak-go-cli/internal/cmd"
	"github.com/oakestra/oak-go-cli/internal/download"
	"github.com/oakestra/oak-go-cli/internal/enact"
	"github.com/oakestra/oak-go-cli/internal/iotools"
)

//go:embed cni-default.conflist
var cniDefaultConfig string

//go:embed crictl-default.yaml
var crictlDefaultConfig string

func installContainerd(_ OSFamily, autoConfirm bool) (bool, error) {
	if err := cmd.RunSilent("systemctl", "is-active", "--quiet", "containerd"); err == nil {
		cliout.Infof("Existing containerd installation is already running.")
		return true, nil
	}

	if err := cmd.RunSilent("systemctl", "list-unit-files", "containerd.service"); err == nil {
		plan := &enact.Plan{
			Info:  "Existing containerd installation is not running",
			Goal:  "start and enable it",
			Steps: getContainerdEnableSteps(),
		}
		return plan.Execute(autoConfirm)
	}

	tempDir, err := iotools.CreateLargeTempDir("containerd-install")
	if err != nil {
		return false, err
	}
	defer iotools.RemoveAllOrWarn(tempDir)

	plan := &enact.Plan{
		Info:  "No existing installation of containerd detected",
		Goal:  "download and install containerd stack",
		Steps: getCustomContainerdInstallSteps(tempDir),
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

func getCustomContainerdInstallSteps(tempDir string) []enact.Step {
	return []enact.Step{
		enact.FuncStep{
			Name: "Download and install containerd to /usr/local/bin/ and /etc/containerd/config.toml",
			Fn: func() error {
				tarPath := filepath.Join(tempDir, "containerd.tar.gz")
				url := fmt.Sprintf("https://github.com/containerd/containerd/releases/download/v2.3.2/containerd-2.3.2-linux-%s.tar.gz", runtime.GOARCH)
				if err := download.GetHttpBodyToFile(url, tarPath, 0o644); err != nil {
					return err
				}

				if err := iotools.ExtractTarGzip(
					tarPath,
					iotools.RegexpTarTarget{
						TarNameRegex:    regexp.MustCompile(`^bin/(.+)`),
						DstPathTemplate: "/usr/local/bin/$1",
						DstPerm:         0o755,
					},
				); err != nil {
					return err
				}

				if err := os.MkdirAll("/usr/local/lib/systemd/system", 0o755); err != nil {
					return err
				}
				if err := download.GetHttpBodyToFile("https://raw.githubusercontent.com/containerd/containerd/refs/tags/v2.3.2/containerd.service", "/usr/local/lib/systemd/system/containerd.service", 0o644); err != nil {
					return err
				}

				out, err := cmd.RunCaptured("/usr/local/bin/containerd", "config", "default")
				if err != nil {
					return fmt.Errorf("failed to generate default containerd config: %w", err)
				}

				if err := os.MkdirAll("/etc/containerd", 0o755); err != nil {
					return err
				}
				return os.WriteFile("/etc/containerd/config.toml", out, 0o644)
			},
		},
		enact.FuncStep{
			Name: "Download and install runc to /usr/local/sbin/",
			Fn: func() error {
				if err := os.MkdirAll("/usr/local/sbin", 0o755); err != nil {
					return err
				}
				url := fmt.Sprintf("https://github.com/opencontainers/runc/releases/download/v1.5.0/runc.%s", runtime.GOARCH)
				return download.GetHttpBodyToFile(url, "/usr/local/sbin/runc", 0o755)
			},
		},
		enact.FuncStep{
			Name: "Download and install CNI plugins to /opt/cni/bin/ and /etc/cni/net.d/10-containerd-net.conflist",
			Fn: func() error {
				tarPath := filepath.Join(tempDir, "cni-plugins.tgz")
				url := fmt.Sprintf("https://github.com/containernetworking/plugins/releases/download/v1.9.1/cni-plugins-linux-%s-v1.9.1.tgz", runtime.GOARCH)
				if err := download.GetHttpBodyToFile(url, tarPath, 0o644); err != nil {
					return err
				}
				if err := os.MkdirAll("/opt/cni/bin", 0o755); err != nil {
					return err
				}

				if err := iotools.ExtractTarGzip(
					tarPath,
					iotools.RegexpTarTarget{
						TarNameRegex:    regexp.MustCompile(`^([^/]+)$`),
						DstPathTemplate: "/opt/cni/bin/$1",
						DstPerm:         0o755,
					},
				); err != nil {
					return err
				}

				if err := os.MkdirAll("/etc/cni/net.d", 0o755); err != nil {
					return err
				}
				return os.WriteFile("/etc/cni/net.d/10-containerd-net.conflist", []byte(cniDefaultConfig), 0o644)
			},
		},
		enact.FuncStep{
			Name: "Download and install crictl to /usr/local/bin/ and /etc/crictl.yaml",
			Fn: func() error {
				tarPath := filepath.Join(tempDir, "crictl.tar.gz")
				url := fmt.Sprintf("https://github.com/kubernetes-sigs/cri-tools/releases/download/v1.36.0/crictl-v1.36.0-linux-%s.tar.gz", runtime.GOARCH)
				if err := download.GetHttpBodyToFile(url, tarPath, 0o644); err != nil {
					return err
				}

				if err := iotools.ExtractTarGzip(
					tarPath,
					iotools.RegexpTarTarget{
						TarNameRegex:    regexp.MustCompile(`^([^/]+)$`),
						DstPathTemplate: "/usr/local/bin/$1",
						DstPerm:         0o755,
					},
				); err != nil {
					return err
				}

				return os.WriteFile("/etc/crictl.yaml", []byte(crictlDefaultConfig), 0o644)
			},
		},
		enact.CommandStep{
			Command: "systemctl",
			Args:    []string{"daemon-reload"},
		},
		enact.CommandStep{
			Command: "systemctl",
			Args:    []string{"enable", "--now", "containerd"},
		},
	}
}
