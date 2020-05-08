// +build !production

package fakeui

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/secrethub/secrethub-go/internals/assert"
)

// FakeIO is a helper type for testing that implements the ui.IO interface
type FakeIO struct {
	In             *FakeReader
	Out            *FakeWriter
	StdIn          *os.File
	StdOut         *os.File
	PromptIn       *FakeReader
	PromptOut      *FakeWriter
	PasswordReader *FakeReader
	PromptErr      error
}

// NewIO creates a new FakeIO with empty buffers.
func NewIO(t *testing.T) *FakeIO {
	tempDir, err := ioutil.TempDir("", "")
	assert.OK(t, err)
	stdIn, err := ioutil.TempFile(tempDir, "in")
	assert.OK(t, err)
	stdOut, err := ioutil.TempFile(tempDir, "out")
	assert.OK(t, err)

	t.Cleanup(func() {
		err := os.RemoveAll(tempDir)
		if err != nil {
			fmt.Printf("could not remove temp dir: %s", err)
		}
	})

	return &FakeIO{
		In: &FakeReader{
			Buffer: &bytes.Buffer{},
		},
		Out: &FakeWriter{
			Buffer: &bytes.Buffer{},
		},
		StdIn:  stdIn,
		StdOut: stdOut,
		PasswordReader: &FakeReader{
			Buffer: &bytes.Buffer{},
		},
		PromptIn: &FakeReader{
			Buffer: &bytes.Buffer{},
		},
		PromptOut: &FakeWriter{
			Buffer: &bytes.Buffer{},
		},
	}
}

// Stdin returns the mocked In.
func (f *FakeIO) Input() io.Reader {
	return f.In
}

// Stdout returns the mocked Out.
func (f *FakeIO) Output() io.Writer {
	return f.Out
}

func (f *FakeIO) Stdin() *os.File {
	return f.StdIn
}

func (f *FakeIO) Stdout() *os.File {
	return f.StdOut
}

func (f *FakeIO) ReadStdout() ([]byte, error) {
	return ioutil.ReadFile(f.StdOut.Name())
}

// Prompts returns the mocked prompts and error.
func (f *FakeIO) Prompts() (io.Reader, io.Writer, error) {
	return f.PromptIn, f.PromptOut, f.PromptErr
}

func (f *FakeIO) IsInputPiped() bool {
	return f.In.Piped
}

func (f *FakeIO) IsOutputPiped() bool {
	return f.Out.Piped
}

func (f *FakeIO) ReadPassword() ([]byte, error) {
	return ioutil.ReadAll(f.PasswordReader)
}

// FakeReader implements the Reader interface.
type FakeReader struct {
	*bytes.Buffer
	Piped   bool
	i       int
	Reads   []string
	ReadErr error
}

// Read returns the mocked ReadErr or reads from the mocked buffer.
func (f *FakeReader) Read(p []byte) (n int, err error) {
	if f.ReadErr != nil {
		return 0, f.ReadErr
	}
	if len(f.Reads) > 0 {
		if len(f.Reads) <= f.i {
			return 0, errors.New("no more fake lines to read")
		}
		f.Buffer = bytes.NewBufferString(f.Reads[f.i])
		f.i++
	}
	return f.Buffer.Read(p)
}

// FakeWriter implements the Writer interface.
type FakeWriter struct {
	*bytes.Buffer
	Piped bool
}
