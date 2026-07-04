package iotools

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/spf13/afero"
)

type TarDestination struct {
	DstPath string
	DstPerm os.FileMode
}

type TarTarget interface {
	Match(cleanName string) *TarDestination
}

type ExactTarTarget struct {
	TarName string
	DstPath string
	DstPerm os.FileMode
}

func (e ExactTarTarget) Match(cleanName string) *TarDestination {
	if e.TarName == cleanName {
		return &TarDestination{DstPath: e.DstPath, DstPerm: e.DstPerm}
	}
	return nil
}

type RegexpTarTarget struct {
	TarNameRegex    *regexp.Regexp
	DstPathTemplate string
	DstPerm         os.FileMode
}

func (r RegexpTarTarget) Match(cleanName string) *TarDestination {
	if r.TarNameRegex == nil {
		return nil
	}
	if r.TarNameRegex.MatchString(cleanName) {
		dstPath := r.TarNameRegex.ReplaceAllString(cleanName, r.DstPathTemplate)
		return &TarDestination{DstPath: dstPath, DstPerm: r.DstPerm}
	}
	return nil
}

func ExtractTarGzip(
	tarGzipPath string,
	targets ...TarTarget,
) error {
	return ExtractTarGzipInFs(afero.NewOsFs(), tarGzipPath, targets...)
}

func ExtractTarGzipInFs(
	fs afero.Fs,
	tarGzipPath string,
	targets ...TarTarget,
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
	targets ...TarTarget,
) error {
	return ExtractTarInFs(afero.NewOsFs(), tarPath, targets...)
}

// ExtractTarInFs extracts specific files from a tar archive based on the provided targets.
func ExtractTarInFs(
	fs afero.Fs,
	tarPath string,
	targets ...TarTarget,
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
	targets ...TarTarget,
) error {
	tarReader := tar.NewReader(reader)

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

		dst := findMatchingTarget(cleanName, targets)
		if dst == nil {
			continue
		}

		if err := extractTarFile(fs, tarReader, dst.DstPath, dst.DstPerm); err != nil {
			return err
		}
	}

	return nil
}

func findMatchingTarget(cleanName string, targets []TarTarget) *TarDestination {
	for _, target := range targets {
		if dest := target.Match(cleanName); dest != nil {
			return dest
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
