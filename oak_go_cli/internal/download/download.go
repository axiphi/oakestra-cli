package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/oakestra/oak-go-cli/internal/iotools"
)

type HTTPError struct {
	StatusCode int
	URL        string
	Body       string
}

func (e *HTTPError) Error() string {
	preview := e.Body
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}
	return fmt.Sprintf("unsuccessful status code %d fetching %s: %s", e.StatusCode, e.URL, preview)
}

func checkHTTPStatus(resp *http.Response, url string) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {

		// limit error body to 10KiB.
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024))

		return &HTTPError{
			StatusCode: resp.StatusCode,
			URL:        url,
			Body:       strings.TrimSpace(string(bodyBytes)),
		}
	}
	return nil
}

func GetHttpBodyToFile(url string, dstPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer iotools.CloseOrWarn(resp.Body, "HTTP response from "+url)

	if err := checkHTTPStatus(resp, url); err != nil {
		return err
	}

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

	if err := checkHTTPStatus(resp, url); err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}
