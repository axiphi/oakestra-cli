package iotools

import (
	"errors"
	"syscall"

	"github.com/spf13/afero"
)

func MoveFile(srcPath string, dstPath string) error {
	return MoveFileInFs(afero.NewOsFs(), srcPath, srcPath)
}

func MoveFileInFs(fs afero.Fs, srcPath string, dstPath string) error {
	err := fs.Rename(srcPath, dstPath)
	if err == nil {
		return nil
	}

	// errno 18/EXDEV: "invalid cross-device link" means os.Rename failed,
	// because we tried to move across filesystems.
	// We can fall back to copying it in that case.
	if !errors.Is(err, syscall.EXDEV) {
		return err
	}

	srcInfo, err := fs.Stat(srcPath)
	if err != nil {
		return err
	}

	if err := CopyFileInFs(fs, srcPath, dstPath, srcInfo.Mode()); err != nil {
		return err
	}
	RemoveOrWarn(srcPath)

	return nil
}

func MoveDir(fs afero.Fs, srcPath string, dstPath string) error {
	return MoveDirInFs(afero.NewOsFs(), srcPath, srcPath)
}

func MoveDirInFs(fs afero.Fs, srcPath string, dstPath string) error {
	err := fs.Rename(srcPath, dstPath)
	if err == nil {
		return nil
	}

	// errno 18/EXDEV: "invalid cross-device link" means os.Rename failed,
	// because we tried to move across filesystems.
	// We can fall back to copying it in that case.
	if !errors.Is(err, syscall.EXDEV) {
		return err
	}

	if err := CopyDirInFs(fs, srcPath, dstPath); err != nil {
		return err
	}
	RemoveAllOrWarn(srcPath)

	return nil
}
