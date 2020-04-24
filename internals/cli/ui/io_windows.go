package ui

import (
	"os"

	"github.com/fatih/color"
	colorable "github.com/mattn/go-colorable"
	isatty "github.com/mattn/go-isatty"
)

// NewUserIO creates a new UserIO middleware from os.Stdin and os.Stdout and adds tty if it is available.
func NewUserIO() UserIO {
	// Ensure colors are printed correctly on Windows.
	if !color.NoColor {
		return UserIO{
			Input:  os.Stdin,
			Output: colorable.NewColorable(os.Stdout),
		}
	}

	return NewStdUserIO()
}

// eofKey returns the key(s) that should be pressed to enter an EOF.
func eofKey() string {
	return "CTRL-Z + ENTER"
}

// isPiped checks whether the file is a pipe.
// If the file does not exist, it returns false.
func isPiped(file *os.File) bool {
	stat, err := file.Stat()
	if err != nil {
		return false
	}

	return os.Getenv("TERM") == "dumb" ||
		(!isatty.IsTerminal(file.Fd()) && !isatty.IsCygwinTerminal(file.Fd()))
}
