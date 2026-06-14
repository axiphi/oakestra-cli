package iotools

import (
	"archive/tar"
	"fmt"
	"io"
	"os"

	"github.com/spf13/afero"
)

type ExtractTarTarget struct {
	TarName string
	DstPath string
	DstPerm os.FileMode
}

// ExtractTar performs ExtractTarInFs in the OS filesystem.
func ExtractTar(
	tarPath string,
	targets ...ExtractTarTarget,
) error {
	return ExtractTarInFs(afero.NewOsFs(), tarPath, targets...)
}

// ExtractTarInFs extracts specific files from a tar archive as specified in pathMapping.
// The target directory baseDstDir must not exist when this function is called.
// If no files in the tar matched any entry in pathMapping, targetDir is not created.
func ExtractTarInFs(
	fs afero.Fs,
	tarPath string,
	targets ...ExtractTarTarget,
) error {
	tarFile, err := fs.Open(tarPath)
	if err != nil {
		return fmt.Errorf("failed to open tar file: %w", err)
	}
	defer CloseOrWarn(tarFile, tarPath)

	tarReader := tar.NewReader(tarFile)

	targetsByTarName := make(map[string]ExtractTarTarget, len(targets))
	for _, target := range targets {
		targetsByTarName[target.TarName] = target
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar archive: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		target, exists := targetsByTarName[header.Name]
		if !exists {
			continue
		}

		if err := extractTarFile(fs, tarReader, target.DstPath, target.DstPerm); err != nil {
			return err
		}
	}

	return nil
}

func extractTarFile(fs afero.Fs, srcReader *tar.Reader, dstPath string, perm os.FileMode) error {
	outFile, err := fs.OpenFile(dstPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer CloseOrWarn(outFile, dstPath)

	if _, err := io.Copy(outFile, srcReader); err != nil {
		return fmt.Errorf("failed to write file contents: %w", err)
	}

	return nil
}
