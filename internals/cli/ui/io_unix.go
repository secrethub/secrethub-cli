// +build darwin linux

package ui

import "os"

// NewUserIO creates a new UserIO middleware from os.Stdin and os.Stdout and adds tty if it is available.
func NewUserIO() UserIO {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err == nil {
		return UserIO{
			Input:        os.Stdin,
			Output:       os.Stdout,
			tty:          tty,
			ttyAvailable: true,
		}
	}

	return NewStdUserIO()
}

// isPiped checks whether the file is a pipe.
// If the file does not exist, it returns false.
func isPiped(file *os.File) bool {
	stat, err := file.Stat()
	if err != nil {
		return false
	}

	return (stat.Mode() & os.ModeCharDevice) == 0
}

func eofKey() string {
	return "CTRL-D"
}
