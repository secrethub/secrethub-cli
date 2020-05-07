package tfstate

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
)

type compressionType string

const (
	NoCompression   compressionType = "none"
	GzipCompression                 = "gzip"
)

var errUnknownCompressionType = errors.New("unknown compression type")

func (t compressionType) Decompress(in []byte) ([]byte, error) {
	switch t {
	case NoCompression:
		return in, nil
	case GzipCompression:
		r, err := gzip.NewReader(bytes.NewBuffer(in))
		if err != nil {
			return nil, err
		}
		return ioutil.ReadAll(r)
	default:
		return nil, errUnknownCompressionType
	}
}

func (t compressionType) Compress(in []byte) ([]byte, error) {
	switch t {
	case NoCompression:
		return in, nil
	case GzipCompression:
		buf := &bytes.Buffer{}
		w := gzip.NewWriter(buf)
		_, err := w.Write(in)
		if err != nil {
			return nil, err
		}
		err = w.Close()
		return buf.Bytes(), err
	default:
		return nil, errUnknownCompressionType
	}
}

func UnpackState(in []byte) ([]byte, error) {
	var compressedState compressedState
	err := json.Unmarshal(in, &compressedState)
	if err != nil {
		return nil, fmt.Errorf("json: %v", err)
	}

	state, err := compressedState.Decompress()
	if err != nil {
		return nil, fmt.Errorf("decompress: %v", err)
	}
	return state, nil
}

func PackState(in []byte) ([]byte, error) {
	compressedState, err := compressState(in, GzipCompression)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(compressedState, "", "\t")
}

func compressState(in []byte, compression compressionType) (compressedState, error) {
	compressed, err := compression.Compress(in)
	if err != nil {
		return compressedState{}, fmt.Errorf("compressing state: %v", err)
	}
	return compressedState{
		Compression: compression,
		Content:     compressed,
	}, nil

}

type compressedState struct {
	Compression compressionType `json:"compression"`
	Content     []byte          `json:"content"`
}

func (s compressedState) Decompress() ([]byte, error) {
	return s.Compression.Decompress(s.Content)
}
