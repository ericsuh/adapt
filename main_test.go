package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal URL",
			input:    "https://example.com/repo",
			expected: "https_example_com_repo",
		},
		{
			name:     "URL with special characters",
			input:    "https://deb.nodesource.com/node_20.x",
			expected: "https_deb_nodesource_com_node_20_x",
		},
		{
			name:     "very long URL",
			input:    "https://very-long-domain-name.example.com/with/many/path/segments/that/exceed/limit",
			expected: "https_very-long-domain-name_example_com_with_many_",
		},
		{
			name:     "URL with port",
			input:    "http://example.com:8080/repo",
			expected: "http_example_com_8080_repo",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only special characters",
			input:    "!!!@@@###$$$",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeFilename() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDownloadGPGKeyHandlesBinary(t *testing.T) {
	// Read the binary GPG key from test data
	binaryKey, err := os.ReadFile("test_data/key.gpg")
	if err != nil {
		t.Fatalf("Failed to read test key: %v", err)
	}

	// Create a test HTTP server that returns binary GPG key
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = io.Copy(w, bytes.NewReader(binaryKey))
	}))
	defer ts.Close()

	// Create a temp file for destination
	tempDir := t.TempDir()
	destPath := filepath.Join(tempDir, "test.gpg")

	// Download the key
	err = downloadGPGKey(ts.URL, destPath)
	if err != nil {
		t.Fatalf("downloadGPGKey failed: %v", err)
	}

	// Verify the downloaded file matches the original
	downloaded, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if !bytes.Equal(downloaded, binaryKey) {
		t.Errorf("Downloaded key doesn't match original")
	}
}

func TestDownloadGPGKeyHandlesAsciiArmored(t *testing.T) {
	// Read the ASCII-armored GPG key from test data
	armoredKey, err := os.ReadFile("test_data/key.gpg.asc")
	if err != nil {
		t.Fatalf("Failed to read test key: %v", err)
	}

	// Read expected binary output
	expectedBinary, err := os.ReadFile("test_data/key.gpg")
	if err != nil {
		t.Fatalf("Failed to read expected binary: %v", err)
	}

	// Create a test HTTP server that returns ASCII-armored GPG key
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.Copy(w, bytes.NewReader(armoredKey))
	}))
	defer ts.Close()

	// Create a temp file for destination
	tempDir := t.TempDir()
	destPath := filepath.Join(tempDir, "test.gpg")

	// Download the key
	err = downloadGPGKey(ts.URL, destPath)
	if err != nil {
		t.Fatalf("downloadGPGKey failed: %v", err)
	}

	// Verify the downloaded file matches the expected binary
	downloaded, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if !bytes.Equal(downloaded, expectedBinary) {
		t.Errorf("Downloaded key doesn't match expected binary")
	}
}

