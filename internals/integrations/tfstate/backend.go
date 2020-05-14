package tfstate

import (
	"fmt"
	"io"
)

type Backend interface {
	Serve() error
}

type prefixWriter struct {
	io.Writer
	prefix string
}

func (l prefixWriter) Write(p []byte) (int, error) {
	_, err := fmt.Fprintf(l.Writer, "%s%s", l.prefix, p)
	return len(p), err
}
