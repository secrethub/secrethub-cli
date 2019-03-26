package secretspec

import (
	"testing"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

func TestDetectEncoding(t *testing.T) {
	cases := []struct {
		name     string
		input    []byte
		expected encoding.Encoding
	}{
		{
			name:     "utf8",
			input:    []byte{0xEF, 0xBB, 0xBF, 0x01, 0x02},
			expected: EncodingUTF8,
		},
		{
			name:     "utf16le",
			input:    []byte{0xFF, 0xFE, 0x03},
			expected: EncodingUTF16LittleEndian,
		},
		{
			name:     "utf16be",
			input:    []byte{0xFE, 0xFF, 0x03},
			expected: EncodingUTF16BigEndian,
		},
		{
			name:     "utf32le",
			input:    []byte{0xFF, 0xFE, 0x00, 0x00, 0x03},
			expected: EncodingUTF32LittleEndian,
		},
		{
			name:     "utf32be",
			input:    []byte{0x00, 0x00, 0xFE, 0xFF, 0x03},
			expected: EncodingUTF32BigEndian,
		},
		{
			name:     "unknown encoding",
			input:    []byte{0x01, 0x02, 0x03},
			expected: nil,
		},
		{
			name:     "empty",
			input:    []byte{},
			expected: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			encoding := DetectEncoding(c.input)

			if c.expected != encoding {
				t.Errorf("unexpected returned encoding, %v (actual) != %v (expected)", encoding, c.expected)
			}

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

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			encoding, err := EncodingFromString(c.input)

			if c.err != nil && err != c.err {
				t.Errorf("returned error not as expected, %v (actual) != %v (expected)", err, c.err)
			}

			if c.expected != nil && c.expected != encoding {
				t.Errorf("unexpected returned encoding, %v (actual) != %v (expected)", encoding, c.expected)
			}

		})
	}
}
