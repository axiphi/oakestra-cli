package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/oakestra/oak-go-cli/internal/cliout"
	"github.com/oakestra/oak-go-cli/internal/download"
	"github.com/oakestra/oak-go-cli/internal/iotools"
)

func installFirstParty(version string, autoConfirm bool) (bool, error) {
	var (
		arch     = runtime.GOARCH
		source   = "remote"
		nodePath string
		netPath  string
		confirm  bool
	)

	resolvedVersion, err := resolveOakestraVersion(version)
	if err != nil {
		return false, fmt.Errorf("failed to resolve Oakestra version: %w", err)
	}

	if !autoConfirm {
		nodeFilePicker := huh.NewFilePicker().
			Title("NodeEngine Archive Path").
			Description("Select your NodeEngine.tar.gz file").
			AllowedTypes([]string{".tar.gz", ".tgz", ".tar"}).
			Validate(huh.ValidateNotEmpty()).
			CurrentDirectory("/").
			Value(&nodePath)
		netFilePicker := huh.NewFilePicker().
			Title("NetManager Archive Path").
			Description("Select your NetManager.tar.gz file").
			AllowedTypes([]string{".tar.gz", ".tgz", ".tar"}).
			Validate(huh.ValidateNotEmpty()).
			CurrentDirectory("/").
			Value(&netPath)

		form := huh.NewForm(
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
				nodeFilePicker,
				netFilePicker,
			).WithHideFunc(func() bool { return source != "local" }),
			huh.NewGroup(
				huh.NewConfirm().
					Title("Proceed with installation?").
					Value(&confirm),
			),
		)

		if currentDir, err := os.Getwd(); err == nil {
			filePickerTraverseFromRootToTargetDir(nodeFilePicker, currentDir, false)
			filePickerTraverseFromRootToTargetDir(netFilePicker, currentDir, false)
		}

		err := form.Run()
		if err != nil {
			return false, fmt.Errorf("configuration aborted: %w", err)
		}
		if !confirm {
			return false, nil
		}
	}

	if err := setupDirectories(); err != nil {
		return false, err
	}

	cliout.Infof("Installing Oakestra Worker Node...")
	if source == "remote" {
		return true, installFromRemote(resolvedVersion, arch)
	} else {
		return true, installFromLocal(nodePath, netPath)
	}
}

func filePickerTraverseFromRootToTargetDir(filePicker *huh.FilePicker, targetDir string, showHidden bool) {
	filePicker.Picking(true)

	cleanPath := filepath.Clean(targetDir)
	parts := strings.Split(cleanPath, string(filepath.Separator))

	currentPath := string(filepath.Separator)
	for _, part := range parts {
		if part == "" {
			continue
		}

		entries, err := filePickerReadDir(currentPath, showHidden)
		if err != nil {
			// If we hit a permission error or the dir doesn't exist, halt traversal
			return
		}

		// Find the index of the target folder within the current directory
		targetIndex := -1
		for i, entry := range entries {
			if entry.Name() == part {
				targetIndex = i
				break
			}
		}

		// If the directory isn't found in the list, we can't traverse further
		if targetIndex == -1 {
			return
		}

		// Send 'Down' commands to move the cursor to the target index.
		// Note: Every time a directory is opened, huh's filepicker resets its cursor to index 0.
		for i := 0; i < targetIndex; i++ {
			filePicker.Update(tea.KeyMsg{
				Type: tea.KeyDown,
				Alt:  false,
			})
		}

		// Send the 'Open' command to enter the directory
		_, msg := filePicker.Update(tea.KeyMsg{
			Type: tea.KeyRight,
			Alt:  false,
		})
		filePicker.Update(interface{}(msg).(tea.Cmd)())

		// Update the current path for the next depth iteration
		currentPath = filepath.Join(currentPath, part)
	}

	filePicker.Picking(false)
}

func filePickerReadDir(path string, showHidden bool) ([]os.DirEntry, error) {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	sort.Slice(dirEntries, func(i, j int) bool {
		if dirEntries[i].IsDir() == dirEntries[j].IsDir() {
			return dirEntries[i].Name() < dirEntries[j].Name()
		}
		return dirEntries[i].IsDir()
	})

	if showHidden {
		return dirEntries, nil
	}

	var sanitizedDirEntries []os.DirEntry
	for _, dirEntry := range dirEntries {
		isHidden := strings.HasPrefix(dirEntry.Name(), ".")
		if isHidden {
			continue
		}
		sanitizedDirEntries = append(sanitizedDirEntries, dirEntry)
	}
	return sanitizedDirEntries, nil
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
