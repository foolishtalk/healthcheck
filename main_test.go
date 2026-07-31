package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestCheckDownloadHeadersUsesHEAD(t *testing.T) {
	var method string
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		method = request.Method
		return &http.Response{
			Status:     "200 OK",
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    request,
		}, nil
	})}

	if err := checkDownloadHeadersWithClient("https://example.com/large-file.zip", time.Second, client); err != nil {
		t.Fatalf("checkDownloadHeaders returned an error: %v", err)
	}
	if method != http.MethodHead {
		t.Fatalf("request method = %q, want HEAD", method)
	}
}

func TestCheckDownloadHeadersRejectsFailure(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			Status:     "404 Not Found",
			StatusCode: http.StatusNotFound,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    request,
		}, nil
	})}

	if err := checkDownloadHeadersWithClient("https://example.com/missing", time.Second, client); err == nil {
		t.Fatal("checkDownloadHeaders returned nil for HTTP 404")
	}
}

func TestLoadConfigAndDeriveHostname(t *testing.T) {
	path := filepath.Join(t.TempDir(), "services.json")
	contents := []byte(`{
  "services": [{
    "name": "example",
    "download_url": "https://downloads.example.com/file.zip"
  }]
}`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	config, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig returned an error: %v", err)
	}
	host, err := config.Services[0].hostname()
	if err != nil {
		t.Fatal(err)
	}
	if host != "downloads.example.com" {
		t.Fatalf("hostname = %q, want downloads.example.com", host)
	}
	if config.timeout() != defaultTimeoutSeconds*time.Second {
		t.Fatalf("default timeout = %s", config.timeout())
	}
}
