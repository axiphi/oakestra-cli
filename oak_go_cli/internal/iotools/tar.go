package iotools

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

type ExtractTarTarget struct {
	TarName string
	DstPath string
	DstPerm os.FileMode
}

func ExtractTarGzip(
	tarGzipPath string,
	targets ...ExtractTarTarget,
) error {
	return ExtractTarGzipInFs(afero.NewOsFs(), tarGzipPath, targets...)
}

func ExtractTarGzipInFs(
	fs afero.Fs,
	tarGzipPath string,
	targets ...ExtractTarTarget,
) error {
	tarGzipFile, err := fs.Open(tarGzipPath)
	if err != nil {
		return fmt.Errorf("failed to open tar gzip file: %w", err)
	}
	defer CloseOrWarn(tarGzipFile, tarGzipPath)

	gzipReader, err := gzip.NewReader(tarGzipFile)
	if err != nil {
		return fmt.Errorf("failed to read tar gzip file: %w", err)
	}
	defer CloseOrWarn(gzipReader, tarGzipPath)

	return extractTarFromReader(fs, gzipReader, targets...)
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

	return extractTarFromReader(fs, tarFile, targets...)
}

func extractTarFromReader(
	fs afero.Fs,
	reader io.Reader,
	targets ...ExtractTarTarget,
) error {
	tarReader := tar.NewReader(reader)

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

		// Sometimes entries are "./filename.txt", other times "filename.txt" and so on.
		// Normalizing the name here makes working with the archives a lot easier.
		cleanName := filepath.Clean(header.Name)

		target, exists := targetsByTarName[cleanName]
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
