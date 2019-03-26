package validation_test

import (
	"github.com/secrethub/secrethub-go/internals/assert"
	"testing"

	"github.com/keylockerbv/secrethub-cli/internals/cli/validation"
)

func TestIsEnvarNameIEEE(t *testing.T) {
	// Arrange
	cases := map[string]struct {
		input    string
		expected bool
	}{
		"valid_chars": {
			input:    makeUnicodeRangeString(0x0007, 0x000d) + makeUnicodeRangeString(0x0020, 0x003c) + makeUnicodeRangeString(0x003e, 0x007e),
			expected: true,
		},
		"empty": {
			input:    "",
			expected: false,
		},
		"NUL_char": {
			input:    "\u0000abc",
			expected: false,
		},
		"equals": {
			input:    "a=b",
			expected: false,
		},
		"invalid_low_range": {
			input:    makeUnicodeRangeString(0x000f, 0x001f),
			expected: false,
		},
		"invalid_high": {
			input:    "\u007fabc",
			expected: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			actual := validation.IsEnvarName(tc.input)

			// Assert
			t.Log(tc.input)
			assert.Equal(t, actual, tc.expected)
		})
	}
}

func TestIsPosixEnvarName(t *testing.T) {
	// Arrange
	cases := map[string]struct {
		input    string
		expected bool
	}{
		"lowercase": {
			input:    "abc",
			expected: true,
		},
		"uppercase": {
			input:    "AbC",
			expected: true,
		},
		"number": {
			input:    "a1b2c3",
			expected: true,
		},
		"leading_number": {
			input:    "0abc",
			expected: false,
		},
		"empty": {
			input:    "",
			expected: false,
		},
		"NUL_char": {
			input:    "\u0000abc",
			expected: false,
		},
		"equals": {
			input:    "a=b",
			expected: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			actual := validation.IsEnvarNamePosix(tc.input)

			// Assert
			assert.Equal(t, actual, tc.expected)
		})
	}
}

func TestValidateEnvarName(t *testing.T) {
	cases := map[string]struct {
		input string
		err   error
	}{
		"valid": {
			input: "foo",
			err:   nil,
		},
		"invalid": {
			input: "foo=",
			err:   validation.ErrInvalidEnvarName("foo="),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			err := validation.ValidateEnvarName(tc.input)

			// Assert
			assert.Equal(t, err, tc.err)
		})
	}
}

// makeUnicodeRangeString makes a string containing all unicode characters
// contained in the given range.
func makeUnicodeRangeString(low byte, high byte) string {
	diff := high - low
	if diff == 0 {
		return string(low)
	}

	if high < low {
		return ""
	}

	str := ""
	for i := byte(0x0000); i <= diff; i += 0x0001 {
		str += string(low + i)
	}
	return str
}
