package tpl

import (
	"fmt"
)

// Evaluate errors
var (
	ErrTemplateVarNotFound = tplError.Code("template_var_not_found").ErrorPref("no value was supplied for template variable '%s'")
)

// Parse errors
type templateSyntaxError struct {
	lineNo int
	colNo  int
	code   string
	msg    string
}

func (err templateSyntaxError) Error() string {
	return tplError.Code(err.code).Errorf("template syntax error at %d:%d: %s", err.lineNo, err.colNo, err.msg).Error()
}

// ErrUnexpectedCharacter is returned when expecting a specific character, for example
// the first character of a closing delimiter after a space occurred in a tag, or
// the second character of a closing delimiter after the first character of the closing
// delimiter.
func ErrUnexpectedCharacter(lineNo, colNo int, actual, expected rune) error {
	return templateSyntaxError{
		lineNo: lineNo,
		colNo:  colNo,
		code:   "unexpected character",
		msg:    fmt.Sprintf("unexpected '%c', expected '%c'", actual, expected),
	}
}

// ErrIllegalVariableCharacter is returned when a variable tag contains a character that is not allowed.
func ErrIllegalVariableCharacter(lineNo, colNo int, char rune) error {
	return templateSyntaxError{
		lineNo: lineNo,
		colNo:  colNo,
		code:   "illegal_variable_character",
		msg:    fmt.Sprintf("illegal character '%c'. Variable names can only contain letters, digits and underscores.", char),
	}
}

// ErrIllegalSecretCharacter is returned when a secret tag contains a character that is not allowed.
func ErrIllegalSecretCharacter(lineNo, colNo int, char rune) error {
	return templateSyntaxError{
		lineNo: lineNo,
		colNo:  colNo,
		code:   "illegal_secret_character",
		msg:    fmt.Sprintf("illegal character '%c'. Secret paths can only contain letters, digits, underscores, hypens, dots, slashes and a colon.", char),
	}
}

// ErrSecretTagNotClosed is returned when a secret tag is opened, but never closed.
func ErrSecretTagNotClosed(lineNo, colNo int) error {
	return templateSyntaxError{
		lineNo: lineNo,
		colNo:  colNo,
		code:   "secret_tag_not_closed",
		msg:    "expected the closing of a secret tag `}}`, but reached the end of the template.",
	}
}

// ErrVariableTagNotClosed is returned when a variable tag is opened, but never closed.
func ErrVariableTagNotClosed(lineNo, colNo int) error {
	return templateSyntaxError{
		lineNo: lineNo,
		colNo:  colNo,
		code:   "variable_tag_not_closed",
		msg:    "expected the closing of a variable tag `}`, but reached the end of the template.",
	}
}
