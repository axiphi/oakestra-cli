package iotools

import (
	"github.com/oakestra/oak-go-cli/internal/cliout"
	"github.com/spf13/afero"
)

func RemoveOrWarn(name string) {
	RemoveOrWarnInFs(afero.NewOsFs(), name)
}

func RemoveOrWarnInFs(fs afero.Fs, name string) {
	if err := fs.Remove(name); err != nil {
		cliout.Warnf("Failed to remove file or directory %q: %v", name, err)
	}
}

func RemoveAllOrWarn(name string) {
	RemoveAllOrWarnInFs(afero.NewOsFs(), name)
}

func RemoveAllOrWarnInFs(fs afero.Fs, name string) {
	if err := fs.RemoveAll(name); err != nil {
		cliout.Warnf("Failed to remove file or directory %q: %v", name, err)
	}
}
