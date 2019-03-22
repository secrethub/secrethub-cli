package consumption

import (
	"github.com/secrethub/secrethub-go/internals/assert"
	"os"
	"testing"

	"io/ioutil"

	"path/filepath"

	"github.com/keylockerbv/secrethub-cli/pkg/validation"
	"github.com/secrethub/secrethub-go/internals/api"
)

func TestNewEnvVar(t *testing.T) {
	// Arrange
	cases := []struct {
		name     string
		source   string
		target   string
		expected *envar
		err      error
	}{
		{
			name:   "valid",
			source: "namespace/repo/dir/secret:latest",
			target: "SECRET",
			expected: &envar{
				source: api.SecretPath("namespace/repo/dir/secret:latest"),
				target: "SECRET",
			},
			err: nil,
		},
		{
			name:     "empty_source",
			source:   "",
			target:   "SECRET",
			expected: nil,
			err:      ErrInvalidSourcePath(api.ValidateSecretPath("")),
		},
		{
			name:     "invalid_source",
			source:   "/repo/secret:latest",
			target:   "SECRET",
			expected: nil,
			err:      ErrInvalidSourcePath(api.ValidateSecretPath("/repo/secret:latest")),
		},
		{
			name:     "empty_target",
			source:   "namespace/repo/dir/secret:latest",
			target:   "",
			expected: nil,
			err:      validation.ErrInvalidEnvarName(""),
		},
		{
			name:     "invalid_target",
			source:   "namespace/repo/dir/secret:latest",
			target:   "SECRET=",
			expected: nil,
			err:      validation.ErrInvalidEnvarName("SECRET="),
		},
		{
			name:   "source_with_whitespace",
			source: " namespace/repo/dir/secret:latest ",
			target: "SECRET",
			expected: &envar{
				source: api.SecretPath("namespace/repo/dir/secret:latest"),
				target: "SECRET",
			},
			err: nil,
		},
		{
			name:   "target_with_whitespace",
			source: "namespace/repo/dir/secret:latest",
			target: " SECRET ",
			expected: &envar{
				source: api.SecretPath("namespace/repo/dir/secret:latest"),
				target: "SECRET",
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			actual, err := newEnvar(tc.source, tc.target)

			// Assert
			assert.Equal(t, err, tc.err)

			if tc.err == nil {
				assert.Equal(t, actual, tc.expected)
			}

		})
	}
}

func TestNewEnv(t *testing.T) {
	v1 := &envar{
		source: "namespace/repo/dir/secret1:latest",
		target: "SECRET_1",
	}

	v2 := &envar{
		source: "namespace/repo/dir/secret2:latest",
		target: "SECRET_2",
	}

	v3 := &envar{
		source: "namespace/repo/dir/secret3:latest",
		target: "SECRET_3",
	}

	absPath, err := filepath.Abs("abs_env_path")
	assert.OK(t, err)

	relPathWithSecretEnv := filepath.Join("rel", SecretEnvPath)

	vars := []*envar{v1, v2, v3}

	// Arrange
	cases := []struct {
		name     string
		env      string
		rootPath string
		vars     []*envar
		expected *env
	}{
		{
			name:     "valid",
			env:      "test_env",
			rootPath: relPathWithSecretEnv,
			vars:     vars,
			expected: &env{
				name:    "test_env",
				dirPath: filepath.Join(relPathWithSecretEnv, "test_env"),
				vars:    vars,
			},
		},
		{
			name:     "sorted_vars",
			env:      "test_env",
			rootPath: relPathWithSecretEnv,
			vars:     []*envar{v3, v1, v2},
			expected: &env{
				name:    "test_env",
				dirPath: filepath.Join(relPathWithSecretEnv, "test_env"),
				vars:    vars,
			},
		},
		{
			name:     "empty_root",
			env:      "test_env",
			rootPath: "",
			vars:     vars,
			expected: &env{
				name:    "test_env",
				dirPath: filepath.Join(SecretEnvPath, "test_env"),
				vars:    vars,
			},
		},
		{
			name:     "empty_vars",
			env:      "test_env",
			rootPath: relPathWithSecretEnv,
			vars:     []*envar{},
			expected: &env{
				name:    "test_env",
				dirPath: filepath.Join(relPathWithSecretEnv, "test_env"),
				vars:    []*envar{},
			},
		},
		{
			name:     "empty_env",
			env:      "",
			rootPath: relPathWithSecretEnv,
			vars:     vars,
			expected: &env{
				name:    defaultEnvName,
				dirPath: filepath.Join(relPathWithSecretEnv, defaultEnvName),
				vars:    vars,
			},
		},
		{
			name:     "no_secret_env_in_root",
			env:      "test_env",
			rootPath: "rel",
			vars:     vars,
			expected: &env{
				name:    "test_env",
				dirPath: filepath.Join(relPathWithSecretEnv, "test_env"),
				vars:    vars,
			},
		},
		{
			name:     "absolute_root_path",
			env:      "test_env",
			rootPath: absPath,
			vars:     vars,
			expected: &env{
				name:    "test_env",
				dirPath: filepath.Join(absPath, SecretEnvPath, "test_env"),
				vars:    vars,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			actual := newEnv(tc.env, tc.rootPath, tc.vars...)

			// Assert
			assert.Equal(t, actual, tc.expected)
		})
	}
}

func TestEnvParse(t *testing.T) {
	// Arrange
	source1 := "namespace/repo/dir/secret1:latest"
	target1 := "SECRET1"
	v1, err := newEnvar(source1, target1)
	assert.OK(t, err)

	source2 := "namespace/repo/dir/secret2:latest"
	target2 := "SECRET2"
	v2, err := newEnvar(source2, target2)
	assert.OK(t, err)

	cases := []struct {
		name     string
		config   map[string]interface{}
		expected *env
		err      error
	}{
		{
			name: "all_fields",
			config: map[string]interface{}{
				fieldName: "test_env",
				fieldVars: map[interface{}]interface{}{
					target1: source1,
					target2: source2,
				},
			},
			expected: &env{
				name:    "test_env",
				dirPath: filepath.Join(SecretEnvPath, "test_env"),
				vars: []*envar{
					v1,
					v2,
				},
			},
			err: nil,
		},
		{
			name: "empty_env_name",
			config: map[string]interface{}{
				fieldName: "",
				fieldVars: map[interface{}]interface{}{
					target1: source1,
					target2: source2,
				},
			},
			expected: &env{
				name:    defaultEnvName,
				dirPath: filepath.Join(SecretEnvPath, defaultEnvName),
				vars: []*envar{
					v1,
					v2,
				},
			},
			err: nil,
		},
		{
			name: "env_name_not_set",
			config: map[string]interface{}{
				fieldVars: map[interface{}]interface{}{
					target1: source1,
					target2: source2,
				},
			},
			expected: &env{
				name:    defaultEnvName,
				dirPath: filepath.Join(SecretEnvPath, defaultEnvName),
				vars: []*envar{
					v1,
					v2,
				},
			},
			err: nil,
		},
		{
			name: "empty_vars",
			config: map[string]interface{}{
				fieldName: "test_env",
				fieldVars: map[interface{}]interface{}{},
			},
			expected: &env{
				name:    "test_env",
				dirPath: filepath.Join(SecretEnvPath, "test_env"),
				vars:    []*envar{},
			},
			err: nil,
		},
		{
			name: "vars_not_set",
			config: map[string]interface{}{
				fieldName: "test_env",
			},
			expected: nil,
			err:      ErrFieldNotSet(fieldVars, make(map[interface{}]interface{})),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parser := EnvParser{}

			// Act
			actual, err := parser.Parse("", true, tc.config)

			// Assert
			assert.Equal(t, err, tc.err)

			if tc.err == nil {
				assert.Equal(t, actual, tc.expected)
			}
		})
	}
}

func TestEnvSetAndClear(t *testing.T) {
	// Arrange
	source1 := "user/repo/dir/secret1:latest"
	target1 := "SECRET1"
	secret1 := api.SecretVersion{
		Data: []byte("secret 1 data"),
	}
	v1, err := newEnvar(source1, target1)
	assert.OK(t, err)

	source2 := "user/repo/dir/secret2:latest"
	target2 := "SECRET2"
	secret2 := api.SecretVersion{
		Data: []byte("secret 2 data\n"),
	}
	v2, err := newEnvar(source2, target2)
	assert.OK(t, err)

	vNonExisting, err := newEnvar("user/repo/non_existing_secret", "NONEXISTING")
	assert.OK(t, err)

	secrets := map[api.SecretPath]api.SecretVersion{
		api.SecretPath(source1): secret1,
		api.SecretPath(source2): secret2,
	}

	cases := []struct {
		name     string
		vars     []*envar
		err      error
		expected map[string][]byte
	}{
		{
			name: "valid",
			vars: []*envar{
				v1,
				v2,
			},
			err: nil,
			expected: map[string][]byte{
				target1: secret1.Data,
				target2: secret2.Data,
			},
		},
		{
			name: "non_existing_secret",
			vars: []*envar{
				v1,
				v2,
				vNonExisting,
			},
			err:      ErrSecretNotFound(vNonExisting.source),
			expected: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newEnv("", "", tc.vars...)

			// Act
			err := env.Set(secrets)

			// Assert
			assert.Equal(t, err, tc.err)

			if tc.err == nil {
				for target, expected := range tc.expected {
					actual, err := ioutil.ReadFile(filepath.Join(env.dirPath, target))
					if err != nil {
						t.Errorf("cannot read file: %v", err)
					} else {
						assert.Equal(t, actual, expected)
					}
				}

				err = env.Clear()
				assert.OK(t, err)

				_, err = os.Stat(env.dirPath)
				if !os.IsNotExist(err) {
					t.Fatalf("file was not cleared: %v", err)
				}
			}
		})
	}
}
