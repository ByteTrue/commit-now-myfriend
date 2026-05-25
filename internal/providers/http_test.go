package providers

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestReadResponseBodyRejectsOversizedBodies(t *testing.T) {
	body := strings.Repeat("a", int(maxResponseBodyBytes)+1)
	response := &http.Response{Body: io.NopCloser(strings.NewReader(body))}

	_, err := readResponseBody(response)
	if err == nil {
		t.Fatal("expected oversized response body error")
	}
	if !strings.Contains(err.Error(), "response body exceeds") {
		t.Fatalf("unexpected error: %v", err)
	}
}
