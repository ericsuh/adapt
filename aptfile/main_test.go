package aptfile

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected any
		wantErr  bool
	}{
		{
			name:     "package directive",
			line:     "package curl",
			expected: PackageDirective{Name: "curl"},
		},
		{
			name:     "package directive with version",
			line:     `package "curl=5.3"`,
			expected: PackageDirective{Name: "curl", Version: "5.3"},
		},
		{
			name:     "package directive with version (long form)",
			line:     `package "curl", version: "5.3"`,
			expected: PackageDirective{Name: "curl", Version: "5.3"},
		},
		{
			name:     "package directive with release",
			line:     `package "curl/multiverse"`,
			expected: PackageDirective{Name: "curl", Release: "multiverse"},
		},
		{
			name:     "package directive with release (long form)",
			line:     `package curl, release: "multiverse"`,
			expected: PackageDirective{Name: "curl", Release: "multiverse"},
		},
		{
			name:     "ppa directive",
			line:     "ppa deadsnakes/ppa",
			expected: PpaDirective{Name: "deadsnakes/ppa"},
		},
		{
			name: "repo directive",
			line: `repo "https://example.com/ubuntu" jammy main`,
			expected: RepoDirective{
				URL:       "https://example.com/ubuntu",
				Suite:     "jammy",
				Component: "main",
			},
		},
		{
			name: "repo directive with arch and key",
			line: `repo "https://example.com/ubuntu" "jammy" "main", arch: "amd64", signed-by: "https://example.com/key/thing.gpg"`,
			expected: RepoDirective{
				URL:       "https://example.com/ubuntu",
				Suite:     "jammy",
				Component: "main",
				Arch:      "amd64",
				SignedBy:  "https://example.com/key/thing.gpg",
			},
		},
		{
			name: "repo-src directive",
			line: `repo-src "https://example.com/ubuntu" jammy main`,
			expected: RepoDirective{
				IsSrc:     true,
				URL:       "https://example.com/ubuntu",
				Suite:     "jammy",
				Component: "main",
			},
		},
		{
			name:     "deb directive",
			line:     `deb "https://example.com/tool.deb"`,
			expected: DebFileDirective{Path: "https://example.com/tool.deb"},
		},
		{
			name:     "hold directive",
			line:     "hold curl",
			expected: HoldDirective{PackageName: "curl"},
		},
		{
			name:    "invalid syntax",
			line:    "package foo: bar",
			wantErr: true,
		},
		{
			name:    "unknown directive",
			line:    "foo bar",
			wantErr: true,
		},
		{
			name:    "package missing argument",
			line:    "package",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseLine(0, tc.line)
			fmt.Printf("%s: %v\n", tc.name, got)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err, "Got error", got)
			require.Equal(t, tc.expected, got)
		})
	}
}
