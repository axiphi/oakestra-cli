package install

import (
	"fmt"
	"os"
	"runtime"
)

func checkPrerequisites() error {
	if err := checkIsLinux(); err != nil {
		return err
	}
	if err := checkIsSystemdBased(); err != nil {
		return err
	}
	if err := checkHasRootPrivileges(); err != nil {
		return err
	}
	return nil
}

func checkIsLinux() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("this tool requires a Linux operating system")
	}
	return nil
}

func checkIsSystemdBased() error {
	// Official systemd check: https://www.freedesktop.org/software/systemd/man/latest/sd_booted.html
	if info, err := os.Stat("/run/systemd/system/"); err != nil || !info.IsDir() {
		return fmt.Errorf("systemd is not running on this machine")
	}
	return nil
}

func checkHasRootPrivileges() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("root privileges required. Please run with sudo or as root")
	}
	return nil
}
