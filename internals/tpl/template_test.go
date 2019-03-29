package tpl

import (
	"fmt"
	"testing"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
)

var (
	testSecretPath   = api.SecretPath("danny/example-repo/hello")
	testSecretValue  = "hello world"
	testSecretPath2  = api.SecretPath("danny/example-repo/hello2")
	testSecretValue2 = "hello world2"

	dataJSON = fmt.Sprintf(
		`{
			"some_field" : "some value",
			"secret_field" : "${%s}"
		}`,
		testSecretPath,
	)
	expectedJSON = fmt.Sprintf(
		`{
			"some_field" : "some value",
			"secret_field" : "%s"
		}`,
		testSecretValue,
	)

	dataYAML = fmt.Sprintf(
		`config:
			some_field: "some value"
			secret_field: "${%s}"`,
		testSecretPath,
	)
	expectedYAML = fmt.Sprintf(
		`config:
			some_field: "some value"
			secret_field: "%s"`,
		testSecretValue,
	)
)

func TestParse(t *testing.T) {
	// Arrange
	cases := map[string]struct {
		raw      string
		success  bool
		expected []api.SecretPath
	}{
		"empty_string": {
			raw:      "",
			success:  true,
			expected: []api.SecretPath{},
		},
		"none": {
			raw:      "foo=bar",
			success:  true,
			expected: []api.SecretPath{},
		},
		"one": {
			raw:      fmt.Sprintf(`${%s}`, testSecretPath),
			success:  true,
			expected: []api.SecretPath{testSecretPath},
		},
		"with_space": {
			raw:      fmt.Sprintf(`${ %s }`, testSecretPath),
			success:  true,
			expected: []api.SecretPath{testSecretPath},
		},
		"two": {
			raw:      fmt.Sprintf(`${ %s }${ %s}`, testSecretPath, testSecretPath2),
			success:  true,
			expected: []api.SecretPath{testSecretPath, testSecretPath2},
		},
		"duplicates": {
			raw:      fmt.Sprintf(`${ %s }${ %s}${%s }`, testSecretPath, testSecretPath2, testSecretPath2),
			success:  true,
			expected: []api.SecretPath{testSecretPath, testSecretPath2},
		},
		"invalid_path": {
			raw:      `${ invalidpath }`,
			success:  true,
			expected: []api.SecretPath{},
		},
		"empty": {
			raw:      `${}`,
			success:  true,
			expected: []api.SecretPath{},
		},
		"empty_nested": {
			raw:      `${${}}`,
			success:  true,
			expected: []api.SecretPath{},
		},
		"path_folowed_by_delim": {
			raw:      fmt.Sprintf(`${ %s ${}}`, testSecretPath),
			success:  true,
			expected: []api.SecretPath{},
		},
		"path_followed_by_nested_path": {
			raw:      fmt.Sprintf(`${ %s ${ %s }}`, testSecretPath, testSecretPath2),
			success:  true,
			expected: []api.SecretPath{testSecretPath2},
		},
		"nested": {
			raw:      fmt.Sprintf(`${ ${ %s }}`, testSecretPath),
			success:  true,
			expected: []api.SecretPath{testSecretPath},
		},
		"unclosed": {
			raw:      `${ foobar`,
			success:  true,
			expected: []api.SecretPath{},
		},
		"unclosed_with_nested": {
			raw:      fmt.Sprintf(`${ ${ %s }`, testSecretPath),
			success:  true,
			expected: []api.SecretPath{testSecretPath},
		},
		"unclosed_with_empty_nested": {
			raw:      `${ ${}`,
			success:  true,
			expected: []api.SecretPath{},
		},
		"unclosed_with_path_and_empty_nested": {
			raw:      fmt.Sprintf(`${ %s ${}`, testSecretPath),
			success:  true,
			expected: []api.SecretPath{},
		},
		"unclosed_with_path_and_nested": {
			raw:      fmt.Sprintf(`${ %s ${ %s }`, testSecretPath, testSecretPath2),
			success:  true,
			expected: []api.SecretPath{testSecretPath2},
		},
		"YAML": {
			raw:      dataYAML,
			success:  true,
			expected: []api.SecretPath{testSecretPath},
		},
		"JSON": {
			raw:      dataJSON,
			success:  true,
			expected: []api.SecretPath{testSecretPath},
		},
		// TODO: add unhappy test cases
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			actual, err := parse(tc.raw, defaultDelimiters)

			// Assert
			if tc.success {
				assert.OK(t, err)
			} else if err == nil {
				t.Errorf("Expected an error but parse succeeded.")
			}

			assert.Equal(t, actual, tc.expected)
		})
	}
}

func TestInject(t *testing.T) {
	// Arrange
	cases := map[string]struct {
		raw      string
		secrets  map[api.SecretPath][]byte
		expected string
		err      error
	}{
		"empty_string": {
			raw:      "",
			expected: "",
		},
		"none": {
			raw:      "foo=bar",
			expected: "foo=bar",
		},
		"one": {
			raw: fmt.Sprintf(`${%s}`, testSecretPath),
			secrets: map[api.SecretPath][]byte{
				testSecretPath: []byte(testSecretValue),
			},
			expected: testSecretValue,
		},
		"with_space": {
			raw: fmt.Sprintf(`${ %s }`, testSecretPath),
			secrets: map[api.SecretPath][]byte{
				testSecretPath: []byte(testSecretValue),
			},
			expected: testSecretValue,
		},
		"two": {
			raw: fmt.Sprintf(`${ %s }${ %s}`, testSecretPath, testSecretPath2),
			secrets: map[api.SecretPath][]byte{
				testSecretPath:  []byte(testSecretValue),
				testSecretPath2: []byte(testSecretValue2),
			},
			expected: fmt.Sprintf("%s%s", testSecretValue, testSecretValue2),
		},
		"duplicates": {
			raw: fmt.Sprintf(`${ %s }${ %s}${%s }`, testSecretPath, testSecretPath2, testSecretPath2),
			secrets: map[api.SecretPath][]byte{
				testSecretPath:  []byte(testSecretValue),
				testSecretPath2: []byte(testSecretValue2),
			},
			expected: fmt.Sprintf("%s%s%s", testSecretValue, testSecretValue2, testSecretValue2),
		},
		"not_found": {
			raw: fmt.Sprintf(`${ %s }${ %s}`, testSecretPath, testSecretPath2),
			secrets: map[api.SecretPath][]byte{
				testSecretPath: []byte(testSecretValue),
			},
			err:      ErrSecretNotFound(testSecretPath2),
			expected: "",
		},
		"invalid_path": {
			raw:      `${ invalidpath }`,
			expected: `${ invalidpath }`,
		},
		"empty": {
			raw:      `${}`,
			expected: `${}`,
		},
		"empty_nested": {
			raw:      `${${}}`,
			expected: `${${}}`,
		},
		"path_folowed_by_delim": {
			raw:      fmt.Sprintf(`${ %s ${}}`, testSecretPath),
			expected: fmt.Sprintf(`${ %s ${}}`, testSecretPath),
		},
		"path_followed_by_nested_path": {
			raw: fmt.Sprintf(`${ %s ${ %s }}`, testSecretPath, testSecretPath2),
			secrets: map[api.SecretPath][]byte{
				testSecretPath2: []byte(testSecretValue2),
			},
			expected: fmt.Sprintf(`${ %s %s}`, testSecretPath, testSecretValue2),
		},
		"nested": {
			raw: fmt.Sprintf(`${ ${ %s }}`, testSecretPath),
			secrets: map[api.SecretPath][]byte{
				testSecretPath: []byte(testSecretValue),
			},
			expected: fmt.Sprintf(`${ %s}`, testSecretValue),
		},
		"unclosed": {
			raw:      `${ foobar`,
			expected: `${ foobar`,
		},
		"unclosed_with_nested": {
			raw: fmt.Sprintf(`${ ${ %s }`, testSecretPath),
			secrets: map[api.SecretPath][]byte{
				testSecretPath: []byte(testSecretValue),
			},
			expected: fmt.Sprintf(`${ %s`, testSecretValue),
		},
		"unclosed_with_empty_nested": {
			raw:      `${ ${}`,
			expected: `${ ${}`,
		},
		"unclosed_with_path_and_empty_nested": {
			raw:      fmt.Sprintf(`${ %s ${}`, testSecretPath),
			expected: fmt.Sprintf(`${ %s ${}`, testSecretPath),
		},
		"unclosed_with_path_and_nested": {
			raw: fmt.Sprintf(`${ %s ${ %s }`, testSecretPath, testSecretPath2),
			secrets: map[api.SecretPath][]byte{
				testSecretPath2: []byte(testSecretValue2),
			},
			expected: fmt.Sprintf(`${ %s %s`, testSecretPath, testSecretValue2),
		},
		"YAML": {
			raw: dataYAML,
			secrets: map[api.SecretPath][]byte{
				testSecretPath: []byte(testSecretValue),
			},
			expected: expectedYAML,
		},
		"JSON": {
			raw: dataJSON,
			secrets: map[api.SecretPath][]byte{
				testSecretPath: []byte(testSecretValue),
			},
			expected: expectedJSON,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			actual, err := injectRecursive(tc.raw, defaultDelimiters, tc.secrets)

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, actual, tc.expected)
		})
	}
}
