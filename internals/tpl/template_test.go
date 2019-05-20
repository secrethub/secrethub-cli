package tpl

import (
	"fmt"
	"sort"
	"testing"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
)

var (
	testSecretPath   = "danny/example-repo/hello"
	testSecretValue  = "hello world"
	testSecretPath2  = "danny/example-repo/hello2"
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
		expected []string
		err      error
	}{
		"empty_string": {
			raw:      "",
			expected: []string{},
		},
		"none": {
			raw:      "foo=bar",
			expected: []string{},
		},
		"one": {
			raw:      fmt.Sprintf(`${%s}`, testSecretPath),
			expected: []string{testSecretPath},
		},
		"with_space": {
			raw:      fmt.Sprintf(`${ %s }`, testSecretPath),
			expected: []string{testSecretPath},
		},
		"two": {
			raw:      fmt.Sprintf(`${ %s }${ %s}`, testSecretPath, testSecretPath2),
			expected: []string{testSecretPath, testSecretPath2},
		},
		"duplicates": {
			raw:      fmt.Sprintf(`${ %s }${ %s}${%s }`, testSecretPath, testSecretPath2, testSecretPath2),
			expected: []string{testSecretPath, testSecretPath2},
		},
		"invalid_path": {
			raw: `${ invalidpath }`,
			err: api.ErrInvalidSecretPath("invalidpath"),
		},
		"empty": {
			raw: `${}`,
			err: api.ErrInvalidSecretName,
		},
		"empty_nested": {
			raw: `${${}}`,
			err: api.ErrInvalidSecretName,
		},
		"path_folowed_by_delim": {
			raw: fmt.Sprintf(`${ %s ${}}`, testSecretPath),
			err: api.ErrInvalidSecretName,
		},
		"path_followed_by_nested_path": {
			raw: fmt.Sprintf(`${ %s ${ %s }}`, testSecretPath, testSecretPath2),
			err: api.ErrInvalidSecretPath(fmt.Sprintf("%s ${ %s", testSecretPath, testSecretPath2)),
		},
		"nested": {
			raw: fmt.Sprintf(`${ ${ %s }}`, testSecretPath),
			err: api.ErrInvalidSecretPath(fmt.Sprintf("${ %s", testSecretPath)),
		},
		"unclosed": {
			raw: `${ foobar`,
			err: ErrReplacementNotClosed(DefaultEndDelimiter),
		},
		"unopened": {
			raw: `{ foobar }`,
			err: ErrReplacementNotOpened(DefaultStartDelimiter),
		},
		"unclosed_with_nested": {
			raw: fmt.Sprintf(`${ ${ %s }`, testSecretPath),
			err: api.ErrInvalidSecretPath(fmt.Sprintf("${ %s", testSecretPath)),
		},
		"unclosed_with_empty_nested": {
			raw: `${ ${}`,
			err: api.ErrInvalidSecretName,
		},
		"unclosed_with_path_and_empty_nested": {
			raw: fmt.Sprintf(`${ %s ${}`, testSecretPath),
			err: api.ErrInvalidSecretName,
		},
		"unclosed_with_path_and_nested": {
			raw: fmt.Sprintf(`${ %s ${ %s }`, testSecretPath, testSecretPath2),
			err: api.ErrInvalidSecretPath(fmt.Sprintf("%s ${ %s", testSecretPath, testSecretPath2)),
		},
		"YAML": {
			raw:      dataYAML,
			expected: []string{testSecretPath},
		},
		"JSON": { // TODO: Decide what to do in this case
			raw:      dataJSON,
			expected: []string{testSecretPath},
		},
		// TODO: add unhappy test cases
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			tpl, err := NewParser().Parse(tc.raw)
			if err == nil {
				actual := tpl.Secrets()
				sort.Strings(actual)
				assert.Equal(t, actual, tc.expected)
			}

			// Assert
			assert.Equal(t, err, tc.err)

		})
	}
}

func TestInject(t *testing.T) {
	// Arrange
	cases := map[string]struct {
		raw      string
		secrets  map[string][]byte
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
			secrets: map[string][]byte{
				testSecretPath: []byte(testSecretValue),
			},
			expected: testSecretValue,
		},
		"with_space": {
			raw: fmt.Sprintf(`${ %s }`, testSecretPath),
			secrets: map[string][]byte{
				testSecretPath: []byte(testSecretValue),
			},
			expected: testSecretValue,
		},
		"two": {
			raw: fmt.Sprintf(`${ %s }${ %s}`, testSecretPath, testSecretPath2),
			secrets: map[string][]byte{
				testSecretPath:  []byte(testSecretValue),
				testSecretPath2: []byte(testSecretValue2),
			},
			expected: fmt.Sprintf("%s%s", testSecretValue, testSecretValue2),
		},
		"duplicates": {
			raw: fmt.Sprintf(`${ %s }${ %s}${%s }`, testSecretPath, testSecretPath2, testSecretPath2),
			secrets: map[string][]byte{
				testSecretPath:  []byte(testSecretValue),
				testSecretPath2: []byte(testSecretValue2),
			},
			expected: fmt.Sprintf("%s%s%s", testSecretValue, testSecretValue2, testSecretValue2),
		},
		"not_found": {
			raw: fmt.Sprintf(`${ %s }${ %s}`, testSecretPath, testSecretPath2),
			secrets: map[string][]byte{
				testSecretPath: []byte(testSecretValue),
			},
			err:      ErrSecretNotFound(testSecretPath2),
			expected: "",
		},
		"YAML": {
			raw: dataYAML,
			secrets: map[string][]byte{
				testSecretPath: []byte(testSecretValue),
			},
			expected: expectedYAML,
		},
		"JSON": {
			raw: dataJSON,
			secrets: map[string][]byte{
				testSecretPath: []byte(testSecretValue),
			},
			expected: expectedJSON,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			tpl, err := NewParser().Parse(tc.raw)
			assert.OK(t, err)
			actual, err := tpl.Inject(tc.secrets)

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, actual, tc.expected)
		})
	}
}
