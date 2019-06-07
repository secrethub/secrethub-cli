package tpl

import (
	"fmt"
	"testing"

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
		expected []node
		err      error
	}{
		"empty_string": {
			raw:      "",
			expected: []node{},
		},
		"none": {
			raw:      "foo=bar",
			expected: []node{val("foo=bar")},
		},
		"one": {
			raw:      fmt.Sprintf(`${%s}`, testSecretPath),
			expected: []node{key(testSecretPath)},
		},
		"with_space": {
			raw:      fmt.Sprintf(`${ %s }`, testSecretPath),
			expected: []node{key(testSecretPath)},
		},
		"two": {
			raw:      fmt.Sprintf(`${ %s }${ %s}`, testSecretPath, testSecretPath2),
			expected: []node{key(testSecretPath), key(testSecretPath2)},
		},
		"duplicates": {
			raw:      fmt.Sprintf(`${ %s }${ %s}${%s }`, testSecretPath, testSecretPath2, testSecretPath2),
			expected: []node{key(testSecretPath), key(testSecretPath2), key(testSecretPath2)},
		},
		"invalid_path": {
			raw:      `${ invalidpath }`,
			expected: []node{key("invalidpath")},
		},
		"empty": {
			raw:      `${}`,
			expected: []node{key("")},
		},
		"empty_nested": {
			raw:      `${${}}`,
			expected: []node{key("${"), val("}")},
		},
		"path_folowed_by_delim": {
			raw:      fmt.Sprintf(`${ %s ${}}`, testSecretPath),
			expected: []node{key(fmt.Sprintf("%s ${", testSecretPath)), val("}")},
		},
		"path_followed_by_nested_path": {
			raw:      fmt.Sprintf(`${ %s ${ %s }}`, testSecretPath, testSecretPath2),
			expected: []node{key(fmt.Sprintf("%s ${ %s", testSecretPath, testSecretPath2)), val("}")},
		},
		"nested": {
			raw:      fmt.Sprintf(`${ ${ %s }}`, testSecretPath),
			expected: []node{key(fmt.Sprintf("${ %s", testSecretPath)), val("}")},
		},
		"unclosed": {
			raw: `${ foobar`,
			err: ErrTagNotClosed("}"),
		},
		"unopened": {
			raw:      `{ foobar }`,
			expected: []node{val("{ foobar }")},
		},
		"unclosed_with_nested": {
			raw:      fmt.Sprintf(`${ ${ %s }`, testSecretPath),
			expected: []node{key(fmt.Sprintf("${ %s", testSecretPath))},
		},
		"unclosed_with_empty_nested": {
			raw:      `${ ${}`,
			expected: []node{key("${")},
		},
		"unclosed_with_path_and_empty_nested": {
			raw:      fmt.Sprintf(`${ %s ${}`, testSecretPath),
			expected: []node{key(fmt.Sprintf("%s ${", testSecretPath))},
		},
		"unclosed_with_path_and_nested": {
			raw:      fmt.Sprintf(`${ %s ${ %s }`, testSecretPath, testSecretPath2),
			expected: []node{key(fmt.Sprintf("%s ${ %s", testSecretPath, testSecretPath2))},
		},
		"YAML": {
			raw: dataYAML,
			expected: []node{
				val(`config:
			some_field: "some value"
			secret_field: "`),
				key(testSecretPath),
				val("\"")},
		},
		"JSON": {
			raw: dataJSON,
			expected: []node{
				val(`{
			"some_field" : "some value",
			"secret_field" : "`),
				key(testSecretPath),
				val(`"
		}`),
			},
		},
		// TODO: add unhappy test cases
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			actual, err := parser{
				startDelim: "${",
				endDelim:   "}",
			}.parse(tc.raw)

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, actual, tc.expected)
		})
	}
}

func TestInject(t *testing.T) {
	// Arrange
	cases := map[string]struct {
		raw      string
		secrets  map[string]string
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
			secrets: map[string]string{
				testSecretPath: testSecretValue,
			},
			expected: testSecretValue,
		},
		"with_space": {
			raw: fmt.Sprintf(`${ %s }`, testSecretPath),
			secrets: map[string]string{
				testSecretPath: testSecretValue,
			},
			expected: testSecretValue,
		},
		"two": {
			raw: fmt.Sprintf(`${ %s }${ %s}`, testSecretPath, testSecretPath2),
			secrets: map[string]string{
				testSecretPath:  testSecretValue,
				testSecretPath2: testSecretValue2,
			},
			expected: fmt.Sprintf("%s%s", testSecretValue, testSecretValue2),
		},
		"duplicates": {
			raw: fmt.Sprintf(`${ %s }${ %s}${%s }`, testSecretPath, testSecretPath2, testSecretPath2),
			secrets: map[string]string{
				testSecretPath:  testSecretValue,
				testSecretPath2: testSecretValue2,
			},
			expected: fmt.Sprintf("%s%s%s", testSecretValue, testSecretValue2, testSecretValue2),
		},
		"not_found": {
			raw: fmt.Sprintf(`${ %s }${ %s}`, testSecretPath, testSecretPath2),
			secrets: map[string]string{
				testSecretPath: testSecretValue,
			},
			err:      ErrKeyNotFound(testSecretPath2),
			expected: "",
		},
		"YAML": {
			raw: dataYAML,
			secrets: map[string]string{
				testSecretPath: testSecretValue,
			},
			expected: expectedYAML,
		},
		"JSON": {
			raw: dataJSON,
			secrets: map[string]string{
				testSecretPath: testSecretValue,
			},
			expected: expectedJSON,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			tpl, err := NewParser("${", "}").Parse(tc.raw)
			assert.OK(t, err)
			actual, err := tpl.Inject(tc.secrets)

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, actual, tc.expected)
		})
	}
}
