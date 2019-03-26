package posix

// AddNewLine takes a input byte and adds if necessary a newline to be posix compliant.
func AddNewLine(input []byte) []byte {
	if len(input) > 0 {
		if input[len(input)-1] == '\n' {
			return input
		}
	}

	return append(input, '\n')
}
