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
	// Input returns an io,Reader that reads input for the current process.
	// If the process's input is piped, this reads from the pipe otherwise it asks input from the user.
	Input() io.Reader
	// Output returns an io.Writer that writes output for the current process.
	// If the process's output is piped, this writes to the pipe otherwise it prints to the terminal.
	Output() io.Writer
	// Stdin returns the *os.File of the current process's stdin stream.
	Stdin() *os.File
	// Stdin returns the *os.File of the current process's stdout stream.
	Stdout() *os.File
	// Prompts returns an io.Reader and io.Writer that read and write directly to/from the terminal, even if the
	// input or output of the current process is piped.
	// If this is not supported, an error is returned.
	Prompts() (io.Reader, io.Writer, error)
	// ReadSecret reads a line of input from the terminal while hiding the entered characters.
	// Returns an error if secret input is not supported.
	ReadSecret() ([]byte, error)
	// IsInputPiped returns whether the current process's input is piped from another process.
	IsInputPiped() bool
	// IsOutputPiped returns whether the current process's output is piped to another process.
	IsOutputPiped() bool
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

func (o standardIO) Stdin() *os.File {
	return o.input
}

func (o standardIO) Stdout() *os.File {
	return o.output
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
// not piped. When either input or output is piped, it returns an error because standardIO does not have
// access to a tty for prompting.
func (o standardIO) Prompts() (io.Reader, io.Writer, error) {
	if o.IsOutputPiped() || o.IsInputPiped() {
		return nil, nil, ErrCannotAsk
	}
	return o.input, o.output, nil
}

func (o standardIO) IsInputPiped() bool {
	return isPiped(o.input)
}

func (o standardIO) IsOutputPiped() bool {
	return isPiped(o.output)
}

func (o standardIO) ReadSecret() ([]byte, error) {
	return readSecret(o.input)
}

// readSecret reads one line of input from the terminal without echoing the user input.
func readSecret(f *os.File) ([]byte, error) {
	// this case happens among other things when input is piped and ReadSecret is called.
	if !terminal.IsTerminal(int(f.Fd())) {
		return nil, ErrCannotAsk
	}

	password, err := terminal.ReadPassword(int(f.Fd()))
	if err != nil {
		return nil, err
	}
	return password, nil
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
