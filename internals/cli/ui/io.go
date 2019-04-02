package ui

import (
	"bufio"
	"io"

	"os"

	"github.com/secrethub/secrethub-go/internals/errio"
	"golang.org/x/crypto/ssh/terminal"
)

// Errors
var (
	errRead      = errio.Namespace("read")
	ErrReadInput = errRead.Code("read_input").ErrorPref("could not read input: %s")
)

// IO is an interface to work with input/output.
type IO interface {
	Stdin() Reader
	Stdout() Writer
	Prompts() (Reader, Writer, error)
}

// UserIO is a middleware between input and output to the CLI program.
// It implements userIO.Prompter and can be passed to libraries.
type UserIO struct {
	Input        Reader
	Output       Writer
	tty          file
	ttyAvailable bool
}

// NewStdUserIO creates a new UserIO middleware only from os.Stdin and os.Stdout.
func NewStdUserIO() UserIO {
	return UserIO{
		Input:  file{os.Stdin},
		Output: file{os.Stdout},
	}
}

// Stdin returns the UserIO's Input.
func (o UserIO) Stdin() Reader {
	return o.Input
}

// Stdout returns the UserIO's Output.
func (o UserIO) Stdout() Writer {
	return o.Output
}

// Prompts simply returns Stdin and Stdout, when both input and output are
// not piped. When either input or output is piped, Prompts attempts to
// bypass stdin and stdout by connecting to /dev/tty on Unix systems when
// available. On systems where tty is not available and when either input
// or output is piped, prompting is not possible so an error is returned.
func (o UserIO) Prompts() (Reader, Writer, error) {
	if o.Input.IsPiped() || o.Output.IsPiped() {
		if o.ttyAvailable {
			return o.tty, o.tty, nil
		}
		return nil, nil, ErrCannotAsk
	}
	return o.Input, o.Output, nil
}

// Reader can read input for a CLI program.
type Reader interface {
	io.Reader
	// ReadPassword reads a line of input from a terminal without local echo.
	ReadPassword() ([]byte, error)
	IsPiped() bool
}

// Readln reads 1 line of input from a io.Reader. The newline character is not included in the response.
func Readln(r io.Reader) (string, error) {
	s := bufio.NewScanner(r)
	s.Scan()
	err := s.Err()
	if err != nil {
		return "", ErrReadInput(err)
	}
	return s.Text(), nil
}

// Writer can write output for a CLI program.
type Writer interface {
	io.Writer
	IsPiped() bool
}

// file implements the Reader and Writer interface.
type file struct {
	*os.File
}

// ReadPassword reads from a terminal without echoing back the typed input.
func (f file) ReadPassword() ([]byte, error) {
	// this case happens among other things when input is piped and ReadPassword is called.
	if !terminal.IsTerminal(int(f.Fd())) {
		return nil, ErrCannotAsk
	}

	return terminal.ReadPassword(int(f.Fd()))
}

// IsPiped checks whether the file is a pipe.
// If the file does not exist, it returns false.
func (f file) IsPiped() bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}

	return (stat.Mode() & os.ModeCharDevice) == 0
}
