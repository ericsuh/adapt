package main

import (
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
