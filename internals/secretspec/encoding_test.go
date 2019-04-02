package secretspec

import (
	"testing"

	"github.com/secrethub/secrethub-go/internals/assert"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

func TestDetectEncoding(t *testing.T) {
	cases := map[string]struct {
		input    []byte
		expected encoding.Encoding
	}{
		"utf8": {
			input:    []byte{0xEF, 0xBB, 0xBF, 0x01, 0x02},
			expected: EncodingUTF8,
		},
		"utf16le": {
			input:    []byte{0xFF, 0xFE, 0x03},
			expected: EncodingUTF16LittleEndian,
		},
		"utf16be": {
			input:    []byte{0xFE, 0xFF, 0x03},
			expected: EncodingUTF16BigEndian,
		},
		"utf32le": {
			input:    []byte{0xFF, 0xFE, 0x00, 0x00, 0x03},
			expected: EncodingUTF32LittleEndian,
		},
		"utf32be": {
			input:    []byte{0x00, 0x00, 0xFE, 0xFF, 0x03},
			expected: EncodingUTF32BigEndian,
		},
		"unknown encoding": {
			input:    []byte{0x01, 0x02, 0x03},
			expected: nil,
		},
		"empty": {
			input:    []byte{},
			expected: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual := DetectEncoding(tc.input)

			assert.Equal(t, actual, tc.expected)

		})
	}
}

func TestEncodingFromString(t *testing.T) {
	cases := []struct {
		input    string
		expected encoding.Encoding
		err      error
	}{
		{
			input:    "utf-8",
			expected: EncodingUTF8,
		},
		{
			input:    "UTF-8",
			expected: EncodingUTF8,
		},
		{
			input:    "utf-16",
			expected: EncodingUTF16,
		},
		{
			input:    "utf-16le",
			expected: EncodingUTF16LittleEndian,
		},
		{
			input:    "utf-16be",
			expected: EncodingUTF16BigEndian,
		},
		{
			input:    "UTF-32",
			expected: EncodingUTF32,
		},
		{
			input:    "utf-32le",
			expected: EncodingUTF32LittleEndian,
		},
		{
			input:    "utf-32be",
			expected: EncodingUTF32BigEndian,
		},
		{
			input:    "Windows-1252",
			expected: charmap.Windows1252,
		},
		{
			input: "unknown",
			err:   ErrUnsupportedEncoding("unknown"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			actual, err := EncodingFromString(tc.input)

			if tc.err != nil && err != tc.err {
				t.Errorf("returned error not as expected, %v (actual) != %v (expected)", err, tc.err)
			}

			if tc.expected != nil && tc.expected != actual {
				t.Errorf("unexpected returned encoding, %v (actual) != %v (expected)", actual, tc.expected)
			}

		})
	}
}
