package armor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCRC24KnownVectors(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected uint32
	}{
		{
			name:     "empty",
			input:    []byte{},
			expected: 0xB704CE,
		},
		{
			name:     "openpgp example",
			input:    []byte("123456789"),
			expected: 0x21CF02,
		},
	}

	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := crc24(c.input)
			require.Equal(t, c.expected, got)
		})
	}
}

func TestParseRejectsBadChecksum(t *testing.T) {
	path := filepath.Join("..", "test_data", "key.gpg.asc")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	broken := bytes.Replace(raw, []byte("=0YYh"), []byte("=0000"), 1)
	_, err = Parse(bytes.NewReader(broken))
	require.ErrorIs(t, err, ErrBadChecksum)
}

func TestParseSuccess(t *testing.T) {
	inputs := []string{
		"alice_cert.asc",
		"alice_private_key.asc",
		"alice_rev_cert.asc",
		"bob_cert.asc",
		"bob_private_key.asc",
		"bob_rev_cert.asc",
		"key.gpg.asc",
	}

	for _, f := range inputs {
		t.Run(f, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join("..", "test_data", f))
			require.NoError(t, err)
			_, err = Parse(bytes.NewReader(content))
			require.NoError(t, err)
		})
	}
}

func TestBodyDecode(t *testing.T) {
	expected, err := os.ReadFile("../test_data/key.gpg")
	require.NoError(t, err)

	armored, err := os.ReadFile("../test_data/key.gpg.asc")
	require.NoError(t, err)
	actual, err := Parse(bytes.NewReader(armored))
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
