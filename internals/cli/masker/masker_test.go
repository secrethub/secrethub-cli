package masker

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/randchar"
)

var maskString = "<redacted by SecretHub>"

func TestMasker(t *testing.T) {
	delay10s := time.Second * 10
	delay1us := time.Microsecond * 1
	delay0s := time.Second * 0

	randomIn, err := randchar.NewGenerator(true).Generate(10000)
	assert.OK(t, err)

	tests := map[string]struct {
		maskStrings []string
		inputFunc   func(io.Writer)
		delay       *time.Duration
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
			delay:    &delay10s,
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
			delay:    &delay1us,
			expected: "foo " + maskString + " test",
		},
		"no timeout": {
			maskStrings: []string{"foo", "bar"},
			inputFunc: func(w io.Writer) {
				_, err := w.Write([]byte("test foo test"))
				assert.OK(t, err)
			},
			delay:    &delay0s,
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
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer

			var maskStrings [][]byte
			for _, s := range tc.maskStrings {
				maskStrings = append(maskStrings, []byte(s))
			}

			m := New(maskStrings)
			m.BufferDelay = time.Millisecond * 10
			if tc.delay != nil {
				m.BufferDelay = *tc.delay
			}

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

	m := New([][]byte{[]byte("test")})
	writer := m.AddStream(&errWriter{err: expectedErr})

	go m.Start()
	_, err := writer.Write([]byte{0x01})
	assert.OK(t, err)

	err = m.Stop()
	assert.Equal(t, err, expectedErr)
}
