// +build darwin linux

package ui

import (
	"io"
	"os"
)

// ttyIO is the implementation of the IO interface that can use a TTY.
type ttyIO struct {
	input  *os.File
	output *os.File
	tty    *os.File
}

// NewUserIO creates a new ttyIO if a TTY is available, otherwise it returns a standardIO.
func NewUserIO() IO {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err == nil {
		return ttyIO{
			input:  os.Stdin,
			output: os.Stdout,
			tty:    tty,
		}
	}

	return newStdUserIO()
}

// Prompts simply returns Stdin and Stdout, when both input and output are
// not piped. When either input or output is piped, Prompts attempts to
// bypass stdin and stdout by connecting to /dev/tty on Unix systems when
// available. On systems where tty is not available and when either input
// or output is piped, prompting is not possible so an error is returned.
func (o ttyIO) Prompts() (io.Reader, io.Writer, error) {
	if o.IsOutputPiped() || o.IsInputPiped() {
		return o.tty, o.tty, nil
	}
	return o.input, o.output, nil
}

func (o ttyIO) IsInputPiped() bool {
	return isPiped(o.input)
}

func (o ttyIO) IsOutputPiped() bool {
	return isPiped(o.output)
}

func (o ttyIO) Stdin() *os.File {
	return o.input
}

func (o ttyIO) Stdout() *os.File {
	return o.output
}

// Stdin returns the standardIO's Input.
func (o ttyIO) Input() io.Reader {
	return o.input
}

// Stdout returns the standardIO's Output.
func (o ttyIO) Output() io.Writer {
	return o.output
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

// eofKey returns the key(s) that should be pressed to enter an EOF.
func eofKey() string {
	return "CTRL-D"
}
