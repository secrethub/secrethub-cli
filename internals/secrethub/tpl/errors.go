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

// ErrUnexpectedDollar is returned when an unescaped dollar sign is followed by a letter or underscore.
func ErrUnexpectedDollar(lineNo, colNo int) error {
	return templateSyntaxError{
		lineNo: lineNo,
		colNo:  colNo,
		code:   "unexpected character",
		msg:    "unexpected '$'. Use '\\$' if you want to output a dollar sign.",
	}
}

// ErrIllegalVariableCharacter is returned when a variable tag contains a character that is not allowed.
func ErrIllegalVariableCharacter(lineNo, colNo int, char rune) error {
	return templateSyntaxError{
		lineNo: lineNo,
		colNo:  colNo,
		code:   "illegal_variable_character",
		msg:    fmt.Sprintf("Illegal character '%c'. Variable names can only contain letters, digits and underscores.", char),
	}
}

// ErrIllegalSecretCharacter is returned when a secret tag contains a character that is not allowed.
func ErrIllegalSecretCharacter(lineNo, colNo int, char rune) error {
	return templateSyntaxError{
		lineNo: lineNo,
		colNo:  colNo,
		code:   "illegal_secret_character",
		msg:    fmt.Sprintf("Illegal character '%c'. Secret paths can only contain letters, digits, underscores, hypens, dots, slashes and a colon.", char),
	}
}

// ErrSecretTagNotClosed is returned when a secret tag is opened, but never closed.
func ErrSecretTagNotClosed(lineNo, colNo int) error {
	return templateSyntaxError{
		lineNo: lineNo,
		colNo:  colNo,
		code:   "secret_tag_not_closed",
		msg:    "Expected the closing of a secret tag `}}`, but reached the end of the template.",
	}
}

// ErrVariableTagNotClosed is returned when a variable tag is opened, but never closed.
func ErrVariableTagNotClosed(lineNo, colNo int) error {
	return templateSyntaxError{
		lineNo: lineNo,
		colNo:  colNo,
		code:   "variable_tag_not_closed",
		msg:    "Expected the closing of a variable tag `}`, but reached the end of the template.",
	}
}
