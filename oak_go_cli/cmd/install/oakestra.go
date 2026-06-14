package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/huh"
	"github.com/oakestra/oak-go-cli/internal/cliout"
	"github.com/oakestra/oak-go-cli/internal/download"
	"github.com/oakestra/oak-go-cli/internal/iotools"
)

func installFirstParty(version string, autoConfirm bool) error {
	var (
		arch     = runtime.GOARCH
		source   = "remote"
		nodePath string
		netPath  string
		confirm  bool
	)

	resolvedVersion, err := resolveOakestraVersion(version)
	if err != nil {
		return fmt.Errorf("failed to resolve Oakestra version: %w", err)
	}

	if !autoConfirm {
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Installation Source").
					Options(
						huh.NewOption(fmt.Sprintf("Download GitHub release %s", resolvedVersion), "remote"),
						huh.NewOption("Provide Local Archives", "local"),
					).
					Value(&source),
			),
			huh.NewGroup(
				huh.NewInput().
					Title("NodeEngine Archive Path").
					Placeholder("/path/to/NodeEngine.tar.gz").
					Value(&nodePath),
				huh.NewInput().
					Title("NetManager Archive Path").
					Placeholder("/path/to/NetManager.tar.gz").
					Value(&netPath),
			).WithHideFunc(func() bool { return source != "local" }),
			huh.NewGroup(
				huh.NewConfirm().
					Title("Proceed with installation?").
					Value(&confirm),
			),
		).Run()

		if err != nil {
			return fmt.Errorf("configuration aborted: %w", err)
		}
		if !confirm {
			cliout.Infof("Aborted worker installation.")
			return nil
		}
	}

	if err := setupDirectories(); err != nil {
		return err
	}

	cliout.Infof("Installing Oakestra Worker Node...")
	if source == "remote" {
		return installFromRemote(resolvedVersion, arch)
	} else {
		return installFromLocal(nodePath, netPath)
	}
}

func resolveOakestraVersion(version string) (string, error) {
	if version == "" {
		return download.GetHttpBody("https://raw.githubusercontent.com/oakestra/oakestra/main/version.txt")
	}
	if version == "alpha" {
		v, err := download.GetHttpBody("https://raw.githubusercontent.com/oakestra/oakestra/develop/version.txt")
		if err != nil {
			return "", err
		}
		return "alpha-" + v, nil
	}
	return version, nil
}

func setupDirectories() error {
	if err := os.MkdirAll("/var/log/oakestra", 0o755); err != nil {
		return fmt.Errorf("failed to create worker log directory (/var/log/oakestra): %w", err)
	}
	if err := os.MkdirAll("/etc/netmanager", 0o755); err != nil {
		return fmt.Errorf("failed to create NetManager config directory (/etc/netmanager): %w", err)
	}
	return nil
}

func installFromRemote(version string, arch string) error {
	tmpDir, err := iotools.CreateLargeTempDir("cli-bin")
	if err != nil {
		return err
	}
	defer iotools.RemoveAllOrWarn(tmpDir)

	nodeArchivePath := filepath.Join(tmpDir, "NodeEngine.tar.gz")
	netArchivePath := filepath.Join(tmpDir, "NetManager.tar.gz")

	nodeArchiveURL := fmt.Sprintf("https://github.com/oakestra/oakestra/releases/download/%s/NodeEngine_%s.tar.gz", version, arch)
	cliout.Infof("Downloading NodeEngine from %s...", nodeArchiveURL)
	if err := download.GetHttpBodyToFile(nodeArchiveURL, nodeArchivePath); err != nil {
		return fmt.Errorf("failed to download NodeEngine: %w", err)
	}

	netArchiveURL := fmt.Sprintf("https://github.com/oakestra/oakestra-net/releases/download/%s/NetManager_%s.tar.gz", version, arch)
	cliout.Infof("Downloading NetManager from %s...", netArchiveURL)
	if err := download.GetHttpBodyToFile(netArchivePath, netArchiveURL); err != nil {
		return fmt.Errorf("failed to download NetManager: %w", err)
	}

	return installFromLocal(nodeArchivePath, netArchivePath)
}

func installFromLocal(nodeArchivePath string, netArchivePath string) error {
	cliout.Infof("Setting up binaries and systemd services...")

	stopSystemdUnits()

	if err := installNodeArchive(nodeArchivePath); err != nil {
		return err
	}
	if err := installNetArchive(netArchivePath); err != nil {
		return err
	}

	if err := reloadSystemdUnits(); err != nil {
		return err
	}

	return nil
}

func installNodeArchive(nodeArchivePath string) error {
	return iotools.ExtractTar(
		nodeArchivePath,
		iotools.ExtractTarTarget{
			TarName: "./NodeEngine",
			DstPath: "/usr/local/bin/NodeEngine",
			DstPerm: 0o755,
		},
		iotools.ExtractTarTarget{
			TarName: "./nodeengined",
			DstPath: "/usr/local/bin/nodeengined",
			DstPerm: 0o755,
		},
		iotools.ExtractTarTarget{
			TarName: "./nodeengine.service",
			DstPath: "/usr/local/lib/systemd/system/nodeengine.service",
			DstPerm: 0o644,
		},
	)
}

func installNetArchive(netArchivePath string) error {
	installTargets := []iotools.ExtractTarTarget{
		{
			TarName: "./NetManager",
			DstPath: "/usr/local/bin/NetManager",
			DstPerm: 0o755,
		},
		{
			TarName: "./netmanager.service",
			DstPath: "/usr/local/lib/systemd/system/netmanager.service",
			DstPerm: 0o644,
		},
	}

	if _, err := os.Stat("/etc/netmanager/tuncfg.json"); errors.Is(err, os.ErrNotExist) {
		installTargets = append(installTargets, iotools.ExtractTarTarget{
			TarName: "./tuncfg.json",
			DstPath: "/etc/netmanager/tuncfg.json",
			DstPerm: 0o644,
		})
	}
	if _, err := os.Stat("/etc/netmanager/netcfg.json"); errors.Is(err, os.ErrNotExist) {
		installTargets = append(installTargets, iotools.ExtractTarTarget{
			TarName: "./netcfg.json",
			DstPath: "/etc/netmanager/netcfg.json",
			DstPerm: 0o644,
		})
	}

	return iotools.ExtractTar(
		netArchivePath,
		installTargets...,
	)
}
