package iotools

import (
	"io"

	"github.com/oakestra/oak-go-cli/internal/cliout"
)

func CloseOrWarn(closer io.Closer, name string) {
	if err := closer.Close(); err != nil {
		cliout.Warnf("failed to close %q: %v", name, err)
	}
}
