package masker

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/randchar"
)

var maskString = "<redacted by SecretHub>"

func TestMasker(t *testing.T) {
	delay10s := time.Second * 10
	delay1us := time.Microsecond * 1

	randomIn, err := randchar.NewGenerator(true).Generate(10000)
	assert.OK(t, err)

	tests := map[string]struct {
		maskStrings []string
		inputFunc   func(io.Writer)
		options     *Options
		expected    string
	}{
		"no_masking": {
			maskStrings: []string{"foo", "bar"},
			inputFunc: func(w io.Writer) {
				_, err := w.Write([]byte("test"))
				assert.OK(t, err)
			},
			expected: "test",
		},
		"single mask": {
			maskStrings: []string{"foo", "bar"},
			inputFunc: func(w io.Writer) {
				_, err := w.Write([]byte("test foo test"))
				assert.OK(t, err)
			},
			expected: "test " + maskString + " test",
		},
		"multiple masks": {
			maskStrings: []string{"foo", "bar"},
			inputFunc: func(w io.Writer) {
				_, err := w.Write([]byte("test foo bar"))
				assert.OK(t, err)
			},
			expected: "test " + maskString + " " + maskString,
		},
		"incomplete mask": {
			maskStrings: []string{"foobar"},
			inputFunc: func(w io.Writer) {
				_, err := w.Write([]byte("test foo"))
				assert.OK(t, err)
			},
			expected: "test foo",
		},
		"mask within a mask": {
			maskStrings: []string{"foo", "bar", "testfoobartestfoo"},
			inputFunc: func(w io.Writer) {
				_, err := w.Write([]byte("testfoobartestfoo bar foo"))
				assert.OK(t, err)
			},
			expected: maskString + " " + maskString + " " + maskString,
		},
		"across multiple writes": {
			maskStrings: []string{"foo", "bar"},
			inputFunc: func(w io.Writer) {
				_, err := w.Write([]byte("fo"))
				assert.OK(t, err)
				_, err = w.Write([]byte("o bar f"))
				assert.OK(t, err)
				_, err = w.Write([]byte("o"))
				assert.OK(t, err)
			},
			expected: maskString + " " + maskString + " fo",
		},
		"within buffer delay": {
			maskStrings: []string{"foo", "bar"},
			inputFunc: func(w io.Writer) {
				_, err := w.Write([]byte("fo"))
				assert.OK(t, err)
				time.Sleep(time.Nanosecond * 1)
				_, err = w.Write([]byte("o test"))
				assert.OK(t, err)
			},
			options:  &Options{BufferDelay: delay10s},
			expected: maskString + " test",
		},
		"outside buffer delay": {
			maskStrings: []string{"foo", "bar"},
			inputFunc: func(w io.Writer) {
				_, err := w.Write([]byte("fo"))
				assert.OK(t, err)
				time.Sleep(time.Millisecond * 10)
				_, err = w.Write([]byte("o bar test"))
				assert.OK(t, err)
			},
			options:  &Options{BufferDelay: delay1us},
			expected: "foo " + maskString + " test",
		},
		"no buffering": {
			maskStrings: []string{"foo", "bar"},
			inputFunc: func(w io.Writer) {
				_, err := w.Write([]byte("test foo test"))
				assert.OK(t, err)
			},
			options:  &Options{DisableBuffer: true},
			expected: "test " + maskString + " test",
		},
		"long input": {
			maskStrings: []string{},
			inputFunc: func(w io.Writer) {
				for _, c := range randomIn {
					_, err := w.Write([]byte{c})
					assert.OK(t, err)
				}
			},
			expected: string(randomIn),
		},
		"reuse input buffer": {
			maskStrings: []string{},
			inputFunc: func(w io.Writer) {
				tmp := make([]byte, 1)
				for _, c := range randomIn {
					copy(tmp, []byte{c})
					_, err := w.Write(tmp)
					assert.OK(t, err)
				}
			},
			expected: string(randomIn),
		},
		"masking unicode": {
			maskStrings: []string{
				"ⓗⓔⓛⓛⓞ",
			},
			inputFunc: func(w io.Writer) {
				_, err := w.Write([]byte("ⓗⓔⓛⓛⓞ world"))
				assert.OK(t, err)
			},
			expected: maskString + " world",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer

			var maskStrings [][]byte
			for _, s := range tc.maskStrings {
				maskStrings = append(maskStrings, []byte(s))
			}

			m := New(maskStrings, tc.options)

			writer := m.AddStream(&buf)
			go m.Start()
			tc.inputFunc(writer)

			err = m.Stop()

			assert.OK(t, err)
			assert.Equal(t, buf.String(), tc.expected)
		})
	}
}

type errWriter struct {
	err error
}

func (w errWriter) Write(p []byte) (n int, err error) {
	return 0, w.err
}

func TestMasker_WriteError(t *testing.T) {
	expectedErr := fmt.Errorf("test")

	m := New([][]byte{[]byte("test")}, nil)
	writer := m.AddStream(&errWriter{err: expectedErr})

	go m.Start()
	_, err := writer.Write([]byte{0x01})
	assert.OK(t, err)

	err = m.Stop()
	assert.Equal(t, err, expectedErr)
}

func TestMasker_MultipleStreams(t *testing.T) {
	sequences := [][]byte{
		[]byte("Gandalf"),
		[]byte("uruk-hai army"),
		[]byte("Aragorn, son of Arathorn"),
		[]byte("hobbit"),
	}

	input := [][]byte{
		[]byte("line 1 "),
		[]byte("line 2 "),
		[]byte("line 3 "),
		[]byte("message from Gandalf the Grey "),
		[]byte("line 5 "),
		[]byte("an uruk-hai army appears "),
		[]byte("say hobbit hobbit hobbit "),
	}

	bufferDelay := 10 * time.Millisecond

	m := New(sequences, &Options{
		BufferDelay: bufferDelay,
	})

	var outputBuffer bytes.Buffer
	var streams [3]io.Writer

	for i := range streams {
		streams[i] = m.AddStream(&outputBuffer)
	}

	go m.Start()

	expected := ""

	for i, b := range input {
		n, err := streams[i%3].Write(b)
		assert.OK(t, err)
		assert.Equal(t, n, len(b))

		expected += string(b)
	}

	assert.Equal(t, outputBuffer.String(), "")

	err := m.Stop()
	assert.OK(t, err)

	for _, sequence := range sequences {
		expected = strings.ReplaceAll(expected, string(sequence), maskString)
	}
	assert.Equal(t, outputBuffer.String(), expected)
}
