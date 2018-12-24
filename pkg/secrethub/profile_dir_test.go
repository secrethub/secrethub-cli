package secrethub

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/keylockerbv/secrethub/testutil"
)

func TestIsOldConfiguration(t *testing.T) {
	testutil.Unit(t)

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
			dir, cleanup := testdata.TempDir(t)
			defer cleanup()

			for _, file := range tc.files {
				err := ioutil.WriteFile(filepath.Join(dir, file), []byte("test data"), 0770)
				testutil.OK(t, err)
			}

			profileDir := ProfileDir(dir)

			// Act
			actual := profileDir.IsOldConfiguration()

			// Assert
			testutil.Compare(t, actual, tc.expected)
		})
	}
}
