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
	Input() io.Reader
	Output() io.Writer
	Prompts() (io.Reader, io.Writer, error)
	IsStdinPiped() bool
	IsStdoutPiped() bool
}

// standardIO is a middleware between input and output to the CLI program.
// It implements standardIO.Prompter and can be passed to libraries.
type standardIO struct {
	input  *os.File
	output *os.File
}

// newStdUserIO creates a new standardIO middleware only from os.Stdin and os.Stdout.
func newStdUserIO() standardIO {
	return standardIO{
		input:  os.Stdin,
		output: os.Stdout,
	}
}

// Stdin returns the standardIO's Input.
func (o standardIO) Input() io.Reader {
	return o.input
}

// Stdout returns the standardIO's Output.
func (o standardIO) Output() io.Writer {
	return o.output
}

// Prompts simply returns Stdin and Stdout, when both input and output are
// not piped. When either input or output is piped, Prompts attempts to
// bypass stdin and stdout by connecting to /dev/tty on Unix systems when
// available. On systems where tty is not available and when either input
// or output is piped, prompting is not possible so an error is returned.
func (o standardIO) Prompts() (io.Reader, io.Writer, error) {
	if o.IsStdoutPiped() || o.IsStdinPiped() {
		return nil, nil, ErrCannotAsk
	}
	return o.input, o.output, nil
}

func (o standardIO) IsStdinPiped() bool {
	return isPiped(o.input)
}

func (o standardIO) IsStdoutPiped() bool {
	return isPiped(o.output)
}

// readPassword reads one line of input from the terminal without echoing the user input.
func readPassword(r io.Reader) (string, error) {
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

// EOFKey returns the key that should be pressed to enter an EOF.
// This can be used to end multiline input.
func EOFKey() string {
	return eofKey()
}
