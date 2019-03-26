// +build darwin linux

package ui

import "os"

// NewUserIO creates a new UserIO middleware from os.Stdin and os.Stdout and adds tty if it is available.
func NewUserIO() UserIO {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err == nil {
		return UserIO{
			Input:        file{os.Stdin},
			Output:       file{os.Stdout},
			tty:          file{tty},
			ttyAvailable: true,
		}
	}

	return NewStdUserIO()
}
