package install

import (
	"fmt"

	"github.com/oakestra/oak-go-cli/internal/cmd"
)

func stopSystemdUnits() {
	// we don't care if this fails because oakestra wasn't installed yet
	_ = cmd.RunSilent("systemctl", "stop", "nodeengine")
	_ = cmd.RunSilent("systemctl", "stop", "netmanager")
}

func reloadSystemdUnits() error {
	if err := cmd.RunSilent("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}
	return nil
}

func startSystemdUnits() error {
	if err := cmd.RunSilent("systemctl", "start", "nodeengine"); err != nil {
		return fmt.Errorf("failed to start nodeengine systemd service: %w", err)
	}
	if err := cmd.RunSilent("systemctl", "start", "netmanager"); err != nil {
		return fmt.Errorf("failed to start nodeengine systemd service: %w", err)
	}
	return nil
}
