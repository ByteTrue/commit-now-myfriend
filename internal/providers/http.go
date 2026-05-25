package providers

import (
	"bytes"
	"fmt"
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

// maxResponseBodyBytes caps provider response reads to protect against runaway responses.
const maxResponseBodyBytes int64 = 10 * 1024 * 1024 // 10 MB

func readResponseBody(response *http.Response) ([]byte, error) {
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxResponseBodyBytes {
		return nil, fmt.Errorf("response body exceeds %d bytes", maxResponseBodyBytes)
	}
	return body, nil
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
