package secretspec

import (
	"bytes"

	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
)

// Errors
var (
	ErrUnsupportedEncoding = errConsumption.Code("unsupported_encoding").ErrorPref("encoding %s not supported")
)

// These are the different types of encoding currently supported.
var (
	EncodingUTF8              = unicode.UTF8
	EncodingUTF16             = unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
	EncodingUTF16LittleEndian = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	EncodingUTF16BigEndian    = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	EncodingUTF32             = utf32.UTF32(utf32.BigEndian, utf32.UseBOM)
	EncodingUTF32LittleEndian = utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM)
	EncodingUTF32BigEndian    = utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM)
)

// EncodingFromString converts a string to the corresponding encoding.Encoding.
// Argument is case-insensitive.
func EncodingFromString(encodingString string) (encoding.Encoding, error) {

	// Detect some encodings that are not detected by ianaindex.IANA.Encoding()
	switch strings.ToLower(encodingString) {
	case "utf-32":
		return EncodingUTF32, nil
	case "utf-32le":
		return EncodingUTF32LittleEndian, nil
	case "utf-32be":
		return EncodingUTF32BigEndian, nil
	}

	enc, err := ianaindex.IANA.Encoding(encodingString)
	if err != nil || enc == nil {
		return nil, ErrUnsupportedEncoding(encodingString)
	}
	return enc, nil
}

// bom represents a Byte-Order Mark and its corresponding encoding.
type bom struct {
	bom      []byte
	encoding encoding.Encoding
}

var bomList = []bom{
	{
		bom:      []byte{0x00, 0x00, 0xFE, 0xFF},
		encoding: EncodingUTF32BigEndian,
	},
	{
		bom:      []byte{0xFF, 0xFE, 0x00, 0x00},
		encoding: EncodingUTF32LittleEndian,
	},
	{
		bom:      []byte{0xEF, 0xBB, 0xBF},
		encoding: EncodingUTF8,
	},
	{
		bom:      []byte{0xFE, 0xFF},
		encoding: EncodingUTF16BigEndian,
	},
	{
		bom:      []byte{0xFF, 0xFE},
		encoding: EncodingUTF16LittleEndian,
	},
}

// DetectEncoding detects the encoding of a text based on its BOM (byte-order mark),
// returning nil if it cannot detect it. In that case, the character encoding is most often UTF8.
//
// The BOM is added to most UTF16, UTF32 and some UTF8 strings to indicate whether it is BigEndian or LittleEndian encoded.
// If a valid BOM is found, you can therefore be quite sure about the character encoding used.
// However, you can never be 100% sure of this result, because you can't tell apart a string without BOM that happens
// to start with the bytes of a valid BOM and a string with a BOM. So the result of this function should be treated
// as a best guess. If there is any information specified about the character encoding, that should always be used
// instead of the result of this function.
func DetectEncoding(input []byte) encoding.Encoding {
	for _, b := range bomList {
		if len(input) < len(b.bom) {
			continue
		}

		if bytes.Equal(input[:len(b.bom)], b.bom) {
			return b.encoding
		}
	}

	return nil
}
