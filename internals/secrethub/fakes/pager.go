package fakes

import "bytes"

type Pager struct {
	Buffer *bytes.Buffer
}

func (f *Pager) Write(p []byte) (n int, err error) {
	return f.Buffer.Write(p)
}

func (f *Pager) Close() error {
	return nil
}

func (f *Pager) IsClosed() bool {
	return false
}
