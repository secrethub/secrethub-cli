package ui

import (
	"io"
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
			Input:  file{os.Stdin},
			Output: colorStdout{colorable.NewColorableStdout()},
		}
	}

	return NewStdUserIO()
}

type colorStdout struct {
	io.Writer
}

func (c colorStdout) IsPiped() bool {
	return os.Getenv("TERM") == "dumb" ||
		(!isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()))
}

func eofKey() string {
	return "CTRL-Z + ENTER"
}
