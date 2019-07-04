// +build !production

package ui

import (
	"bytes"
	"errors"
)

// FakeIO is a helper type for testing that implements the ui.IO interface
type FakeIO struct {
	StdIn     *FakeReader
	StdOut    *FakeWriter
	PromptIn  *FakeReader
	PromptOut *FakeWriter
	PromptErr error
}

// NewFakeIO creates a new FakeIO with empty buffers.
func NewFakeIO() *FakeIO {
	return &FakeIO{
		StdIn: &FakeReader{
			Buffer: &bytes.Buffer{},
		},
		StdOut: &FakeWriter{
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

// Stdin returns the mocked StdIn.
func (f *FakeIO) Stdin() Reader {
	return f.StdIn
}

// Stdout returns the mocked StdOut.
func (f *FakeIO) Stdout() Writer {
	return f.StdOut
}

// Prompts returns the mocked prompts and error.
func (f *FakeIO) Prompts() (Reader, Writer, error) {
	return f.PromptIn, f.PromptOut, f.PromptErr
}

// FakeReader implements the Reader interface.
type FakeReader struct {
	*bytes.Buffer
	Piped   bool
	i       int
	Reads   []string
	ReadErr error
}

// ReadPassword reads a line from the mocked buffer.
func (f *FakeReader) ReadPassword() ([]byte, error) {
	pass, err := Readln(f)
	if err != nil {
		return nil, err
	}
	return []byte(pass), nil
}

// IsPiped returns the mocked Piped.
func (f *FakeReader) IsPiped() bool {
	return f.Piped
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

// IsPiped returns the mocked Piped.
func (f *FakeWriter) IsPiped() bool {
	return f.Piped
}
