package cli

import (
	"testing"
)

func TestSplitVar(t *testing.T) {
	// Arrange
	prefix := "pref"
	separator := "_"

	tests := []struct {
		envVar        string
		expectedKey   string
		expectedValue string
		expectedMatch bool
	}{
		{
			envVar:        "pref_x=y",
			expectedKey:   "pref_x",
			expectedValue: "y",
			expectedMatch: true,
		},
		{
			envVar:        "Pref_x=y",
			expectedKey:   "Pref_x",
			expectedValue: "y",
			expectedMatch: true,
		},
		{
			envVar:        "pref=y",
			expectedKey:   "pref",
			expectedValue: "y",
			expectedMatch: false,
		},
		{
			envVar:        "x=y",
			expectedKey:   "x",
			expectedValue: "y",
			expectedMatch: false,
		},
		{
			envVar:        "pref_x:y",
			expectedKey:   "",
			expectedValue: "",
			expectedMatch: false,
		},
	}

	for _, test := range tests {
		// Act
		key, value, match := splitVar(prefix, separator, test.envVar)

		// Assert
		if key != test.expectedKey {
			t.Errorf("unexpected key for %s: %s (actual) != %s (expected)", test.envVar, key, test.expectedKey)
		}

		if value != test.expectedValue {
			t.Errorf("unexpected value for %s: %s (actual) != %s (expected)", test.envVar, value, test.expectedValue)
		}

		if match != test.expectedMatch {
			t.Errorf("unexpected match for %s: %t (actual) != %t (expected)", test.envVar, match, test.expectedMatch)
		}
	}
}

func TestFormatName(t *testing.T) {
	// Arrange
	prefix := "pref"
	delimiters := []string{"-", "."}
	separator := "_"

	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "name",
			expected: "PREF_NAME",
		},
		{
			name:     "NAME",
			expected: "PREF_NAME",
		},
		{
			name:     "some-name",
			expected: "PREF_SOME_NAME",
		},
		{
			name:     "PREF_NAME",
			expected: "PREF_PREF_NAME",
		},
		{
			name:     "NAME-WITH.DOT",
			expected: "PREF_NAME_WITH_DOT",
		},
	}

	for _, test := range tests {
		// Act
		actual := formatName(test.name, prefix, separator, delimiters...)

		// Assert
		if actual != test.expected {
			t.Errorf("unexpected var name for %s: %s (actual) != %s (expected)", test.name, actual, test.expected)
		}
	}
}
