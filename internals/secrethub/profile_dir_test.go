package secrethub

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/secrethub/secrethub-go/internals/assert"
)

func TestIsOldConfiguration(t *testing.T) {
	// Arrange
	cases := map[string]struct {
		files    []string
		expected bool
	}{
		"none": {
			files:    []string{},
			expected: false,
		},
		"credential": {
			files:    []string{defaultCredentialFilename},
			expected: false,
		},
		"credential_and_old_config": {
			files:    []string{defaultCredentialFilename, oldConfigFilename},
			expected: false,
		},
		"old_config": {
			files:    []string{oldConfigFilename},
			expected: true,
		},
		"old_config_and_key": {
			files:    []string{oldConfigFilename, "key"},
			expected: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Arrange
			dir, cleanup := testdata.tempDir(t)
			defer cleanup()

			for _, file := range tc.files {
				err := os.WriteFile(filepath.Join(dir, file), []byte("test data"), 0770)
				assert.OK(t, err)
			}

			profileDir := ProfileDir(dir)

			// Act
			actual := profileDir.IsOldConfiguration()

			// Assert
			assert.Equal(t, actual, tc.expected)
		})
	}
}
