// +build !production

// Package fakeclip provides fake implementations of the clip.Clipper interface
// to be used for testing.
package fakeclip

import (
	"sync"

	"github.com/secrethub/secrethub-cli/internals/cli/clip"
)

// testClipper implements the clipboard.Clipper interface and
// is used to inject the clipboard with a simple byte array.
type testClipper struct {
	sync.RWMutex
	val []byte
}

// NewTestClipboard creates a new testClipper to replace clip.Clipboard in tests.
func New() clip.Clipper {
	return &testClipper{
		val: []byte{},
	}
}

// NewTestClipboardWithValue creates a new testClipper to replace clip.Clipboard in tests,
// that is initialized with a value.
func NewWithValue(initial []byte) clip.Clipper {
	return &testClipper{
		val: initial,
	}
}

func (c *testClipper) ReadAll() ([]byte, error) {
	c.RLock()
	defer c.RUnlock()

	return c.val, nil
}

func (c *testClipper) WriteAll(value []byte) error {
	c.Lock()
	defer c.Unlock()

	c.val = value
	return nil
}

// NewErrClipboard creates a new errClipper to replace clip.Clipbard in tests.
func NewWithErr(readError error, writeError error) clip.Clipper {
	return &errClipper{
		readError:  readError,
		writeError: writeError,
	}
}

// errClipper implements the clipboard.Clipper interface and
// is used to inject the clipboard with read/write errors.
type errClipper struct {
	writeError error
	readError  error
}

func (c *errClipper) ReadAll() ([]byte, error) {
	return nil, c.readError
}

func (c *errClipper) WriteAll([]byte) error {
	return c.writeError
}
