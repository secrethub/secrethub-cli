package secretspec

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
)

const source1 = "user/repo/dir/secret1:latest"

func TestNewFile(t *testing.T) {
	// Arrange
	cases := []struct {
		name     string
		source   string
		target   string
		filemode os.FileMode
		fail     bool
		expected *file
	}{
		{
			name:     "all_values",
			source:   source1,
			target:   "target",
			filemode: 0770,
			expected: &file{
				source:   source1,
				target:   "target",
				filemode: 0770,
			},
		},
		{
			name:     "empty_source",
			source:   "",
			target:   "target",
			filemode: 0770,
			fail:     true,
		},
		{
			name:     "invalid_source",
			source:   "/repo/secret",
			target:   "target",
			filemode: 0770,
			fail:     true,
		},
		{
			name:     "empty_target",
			source:   source1,
			target:   "",
			filemode: 0770,
			expected: &file{
				source:   source1,
				target:   "secret1",
				filemode: 0770,
			},
		},
		{
			name:     "empty_filemode",
			source:   source1,
			target:   "target",
			filemode: DefaultFileMode,
			expected: &file{
				source:   source1,
				target:   "target",
				filemode: DefaultFileMode,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			actual, err := newFile(tc.source, tc.target, tc.filemode)

			// Assert
			if tc.fail && err == nil {
				t.Errorf("unexpected error: %v (actual) != %v (expected)", err, nil)
			} else if !tc.fail && err != nil {
				t.Errorf("unexpected error: %v (actual) != some error (expected)", err)
			}

			if !tc.fail {
				if !reflect.DeepEqual(actual, tc.expected) {
					t.Errorf("unexpected result: %v (actual) != %v (expected)", actual, tc.expected)
				}
			}
		})
	}
}

func TestFileParse(t *testing.T) {
	// Arrange
	cases := []struct {
		name     string
		config   map[string]interface{}
		fail     bool
		expected *file
	}{
		{
			name: "all_values",
			config: map[string]interface{}{
				"source":   source1,
				"target":   "target",
				"filemode": "0770",
			},
			expected: &file{
				source:   source1,
				target:   "target",
				filemode: 0770,
			},
		},
		{
			name: "invalid_source",
			config: map[string]interface{}{
				"source":   "",
				"target":   "target",
				"filemode": "0770",
			},
			fail: true,
		},
		{
			name: "target_not_set",
			config: map[string]interface{}{
				"source":   source1,
				"filemode": "0770",
			},
			expected: &file{
				source:   source1,
				target:   "secret1",
				filemode: 0770,
			},
		},
		{
			name: "target_empty",
			config: map[string]interface{}{
				"source":   source1,
				"target":   "",
				"filemode": "0770",
			},
			expected: &file{
				source:   source1,
				target:   "secret1",
				filemode: 0770,
			},
		},
		{
			name: "filemode_not_set",
			config: map[string]interface{}{
				"source": source1,
				"target": "target",
			},
			expected: &file{
				source:   source1,
				target:   "target",
				filemode: DefaultFileMode,
			},
		},
		{
			name: "filemode_empty",
			config: map[string]interface{}{
				"source":   source1,
				"target":   "target",
				"filemode": "",
			},
			expected: &file{
				source:   source1,
				target:   "target",
				filemode: DefaultFileMode,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			parser := FileParser{}

			// Act
			actual, err := parser.Parse("", true, tc.config)

			// Assert
			if tc.fail && err == nil {
				t.Errorf("unexpected error: %v (actual) != %v (expected)", err, nil)
			} else if !tc.fail && err != nil {
				t.Errorf("unexpected error: %v (actual) != some error (expected)", err)
			}

			if !tc.fail {
				if !reflect.DeepEqual(actual, tc.expected) {
					t.Errorf("unexpected result: %v (actual) != %v (expected)", actual, tc.expected)
				}
			}
		})
	}
}

func TestFileSetAndClear(t *testing.T) {
	// Arrange
	source1 := "user/repo/dir/secret:latest"
	secret1 := api.SecretVersion{
		Data: []byte("secret 1 data"),
	}

	sourceNewLine := "user/repo/dir/secret_newline:latest"
	secretNewLine := api.SecretVersion{
		Data: []byte("secret 2 data\n"),
	}

	sourceNonExisting := "user/repo/non_existing_secret"

	absTarget, err := filepath.Abs("test_set_abs_path")
	assert.OK(t, err)

	secrets := map[string]api.SecretVersion{
		source1:       secret1,
		sourceNewLine: secretNewLine,
	}

	cases := []struct {
		name     string
		source   string
		target   string
		filemode os.FileMode
		err      error
		expected []byte
	}{
		{
			name:     "relative_target_path",
			source:   source1,
			target:   "test_set_relative_path",
			filemode: DefaultFileMode,
			err:      nil,
			expected: append(secret1.Data, '\n'),
		},
		{
			name:     "absolute_target_path",
			source:   source1,
			target:   absTarget,
			filemode: DefaultFileMode,
			err:      nil,
			expected: append(secret1.Data, '\n'),
		},
		{
			name:     "newline_secret",
			source:   sourceNewLine,
			target:   "test_set_newline_secret",
			filemode: DefaultFileMode,
			err:      nil,
			expected: secretNewLine.Data,
		},
		{
			name:     "secret_not_in_result",
			source:   sourceNonExisting,
			target:   "test_set_non_existing_secret",
			filemode: DefaultFileMode,
			err:      ErrSecretNotFound(sourceNonExisting),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			file, err := newFile(tc.source, tc.target, tc.filemode)
			if err != nil {
				t.Fatalf("cannot create new file consumable: %v", err)
			}

			// Act
			err = file.Set(secrets)

			// Assert
			if err != tc.err {
				t.Errorf("unexpected error: %v (actual) != %v (expected)", err, tc.err)
			}

			if tc.err == nil {
				actual, err := os.ReadFile(tc.target)
				if err != nil {
					t.Errorf("cannot read file: %v", err)
				} else {
					assert.Equal(t, actual, tc.expected)
				}

				err = file.Clear()
				if err != nil {
					t.Fatalf("failed to clear file: %v", err)
				}

				_, err = os.Stat(file.target)
				if !os.IsNotExist(err) {
					t.Fatalf("file was not cleared: %v", err)
				}
			}
		})
	}
}
