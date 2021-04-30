package secrethub

import "bytes"

type FakeClipboardWriter struct {
	Buffer bytes.Buffer
}

func (clipWriter *FakeClipboardWriter) Write(data []byte) error {
	_, err := clipWriter.Buffer.Write(data)
	return err
}
