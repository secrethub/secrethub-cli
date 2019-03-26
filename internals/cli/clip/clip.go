package clip

import (
	"github.com/atotto/clipboard"
	"github.com/secrethub/secrethub-go/internals/errio"
)

var (
	errClip = errio.Namespace("clipboard")

	// ErrCannotRead is returned when data cannot be read to the clipboard.
	ErrCannotRead = errClip.Code("cannot_read").ErrorPref("cannot read from clipboard: %s")
	// ErrCannotWrite is returned when data cannot be written to the clipboard.
	ErrCannotWrite = errClip.Code("cannot_write").ErrorPref("cannot write to clipboard: %s")
)

// Clipper allows you to read from and write to the clipboard.
type Clipper interface {
	ReadAll() ([]byte, error)
	WriteAll(value []byte) error
}

// clip implements the Clipper interface
type clip struct{}

func (c *clip) ReadAll() ([]byte, error) {
	value, err := clipboard.ReadAll()
	if err != nil {
		return nil, ErrCannotRead(err)
	}
	return []byte(value), nil
}

func (c *clip) WriteAll(value []byte) error {
	err := clipboard.WriteAll(string(value))
	if err != nil {
		return ErrCannotWrite(err)
	}
	return nil
}

// NewClipboard creates a new Clipper.
func NewClipboard() Clipper {
	return &clip{}
}
