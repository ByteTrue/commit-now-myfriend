package providers

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

type HTTPDoer interface {
	Do(request *http.Request) (*http.Response, error)
}

func defaultHTTPClient() HTTPDoer {
	return &http.Client{Timeout: 60 * time.Second}
}

func readResponseBody(response *http.Response) ([]byte, error) {
	defer response.Body.Close()
	return io.ReadAll(response.Body)
}

func newJSONRequest(method, url string, body []byte) (*http.Request, error) {
	request, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	return request, nil
}

func ioNopCloserFromBytes(body []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(body))
}
