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
	Stdin() io.Reader
	Stdout() io.Writer
	Prompts() (io.Reader, io.Writer, error)
	IsStdinPiped() bool
	IsStdoutPiped() bool
}

// UserIO is a middleware between input and output to the CLI program.
// It implements userIO.Prompter and can be passed to libraries.
type UserIO struct {
	Input        *os.File
	Output       *os.File
	tty          *os.File
	ttyAvailable bool
}

// NewStdUserIO creates a new UserIO middleware only from os.Stdin and os.Stdout.
func NewStdUserIO() UserIO {
	return UserIO{
		Input:  os.Stdin,
		Output: os.Stdout,
	}
}

// Stdin returns the UserIO's Input.
func (o UserIO) Stdin() io.Reader {
	return o.Input
}

// Stdout returns the UserIO's Output.
func (o UserIO) Stdout() io.Writer {
	return o.Output
}

// Prompts simply returns Stdin and Stdout, when both input and output are
// not piped. When either input or output is piped, Prompts attempts to
// bypass stdin and stdout by connecting to /dev/tty on Unix systems when
// available. On systems where tty is not available and when either input
// or output is piped, prompting is not possible so an error is returned.
func (o UserIO) Prompts() (io.Reader, io.Writer, error) {
	if o.IsStdoutPiped() || o.IsStdinPiped() {
		if o.ttyAvailable {
			return o.tty, o.tty, nil
		}
		return nil, nil, ErrCannotAsk
	}
	return o.Input, o.Output, nil
}

func (o UserIO) IsStdinPiped() bool {
	return isPiped(o.Input)
}

func (o UserIO) IsStdoutPiped() bool {
	return isPiped(o.Output)
}

type PasswordReader interface {
	Read(reader io.Reader) (string, error)
}

type passwordReader struct{}

// NewPasswordReader returns a reader that reads a string from the terminal without echoing the user input.
func NewPasswordReader() PasswordReader {
	return &passwordReader{}
}

// Read reads one line of input from the terminal without echoing the user input.
func (pr *passwordReader) Read(r io.Reader) (string, error) {
	file, ok := r.(*os.File)
	if !ok {
		return "", ErrCannotAsk
	}
	// this case happens among other things when input is piped and ReadPassword is called.
	if !terminal.IsTerminal(int(file.Fd())) {
		return "", ErrCannotAsk
	}

	password, err := terminal.ReadPassword(int(file.Fd()))
	if err != nil {
		return "", err
	}
	return string(password), nil
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

// isPiped checks whether the file is a pipe.
// If the file does not exist, it returns false.
func isPiped(file *os.File) bool {
	stat, err := file.Stat()
	if err != nil {
		return false
	}

	return (stat.Mode() & os.ModeCharDevice) == 0
}

// EOFKey returns the key that should be pressed to enter an EOF.
// This can be used to end multiline input.
func EOFKey() string {
	return eofKey()
}
