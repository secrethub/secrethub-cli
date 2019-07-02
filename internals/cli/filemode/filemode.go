// Package filemode provides a wrapper around os.FileMode so that it can be parsed from a CLI flag.
package filemode

import (
	"os"
	"strconv"

	"github.com/secrethub/secrethub-go/internals/errio"
)

// Errors
var (
	ErrInvalidFilemode = errio.Namespace("filemode").Code("invalid_filemode").ErrorPref("%s is not a valid filemode: %v")
)

// FileMode extends os.FileMode to implement the flag.Value interface,
// so that the file mode can be parsed from a flag.
type FileMode os.FileMode

// New creates a new FileMode.
func New(fileMode os.FileMode) FileMode {
	return FileMode(fileMode)
}

// Parse converts a string like 0644 to an os.FileMode.
func Parse(mode string) (FileMode, error) {
	if len(mode) < 3 {
		return 0, ErrInvalidFilemode(mode, "filemodes must contain at least three digits")
	}

	filemode, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return 0, ErrInvalidFilemode(mode, err)
	}
	return FileMode(filemode), nil
}

// Set implements the flag.Value interface.
func (m *FileMode) Set(value string) error {
	fileMode, err := Parse(value)
	if err != nil {
		return err
	}
	*m = fileMode
	return nil
}

// String implements the flag.Value interface.
func (m FileMode) String() string {
	return string(m)
}

// FileMode returns the file mode as an os.FileMode.
func (m FileMode) FileMode() os.FileMode {
	return os.FileMode(m)
}
