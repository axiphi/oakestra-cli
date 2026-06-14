package download

import (
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/oakestra/oak-go-cli/internal/iotools"
)

func GetHttpBodyToFile(url string, dstPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer iotools.CloseOrWarn(resp.Body, "HTTP response from "+url)

	outFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer iotools.CloseOrWarn(outFile, dstPath)

	_, err = io.Copy(outFile, resp.Body)
	return err
}

func GetHttpBody(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer iotools.CloseOrWarn(resp.Body, "HTTP response from "+url)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}
