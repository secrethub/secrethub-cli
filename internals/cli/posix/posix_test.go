package posix_test

import (
	"bytes"
	"testing"

	"github.com/keylockerbv/secrethub-cli/internals/cli/posix"
)

func TestAddNewLine_EmptyData(t *testing.T) {
	// Arrange
	input := []byte{}

	expected := []byte("\n")

	// Act
	actual := posix.AddNewLine(input)

	// Assert

	if !bytes.Equal(actual, expected) {
		t.Errorf("actual (%s) != expected (%s)", actual, expected)
	}
}

func TestAddNewLine_TrailingNewLine(t *testing.T) {
	// Arrange
	input := []byte("trailing_newline_secret\n")

	expected := input

	// Act
	actual := posix.AddNewLine(input)

	// Assert

	if !bytes.Equal(actual, expected) {
		t.Errorf("actual (%s) != expected (%s)", actual, expected)
	}
}

func TestAddNewLine_NoNewline(t *testing.T) {
	// Arrange
	input := []byte("no_newline_secret")

	expected := append(input, '\n')

	// Act
	actual := posix.AddNewLine(input)

	// Assert

	if !bytes.Equal(actual, expected) {
		t.Errorf("actual (%s) != expected (%s)", actual, expected)
	}
}
