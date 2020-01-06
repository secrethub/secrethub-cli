package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/tpl"
)

type variableReader struct {
	vars map[string]string
}

func newVariableReader(vars map[string]string) *variableReader {
	return &variableReader{
		vars: vars,
	}
}

func (v *variableReader) ReadVariable(name string) (string, error) {
	variable, ok := v.vars[name]
	if !ok {
		return "", tpl.ErrTemplateVarNotFound(name)
	}
	return variable, nil
}

type promptMissingVariableReader struct {
	reader tpl.VariableReader
	io     ui.IO
}

func newPromptMissingVariableReader(reader tpl.VariableReader, io ui.IO) *promptMissingVariableReader {
	return &promptMissingVariableReader{
		reader: reader,
		io:     io,
	}
}

func (p *promptMissingVariableReader) ReadVariable(name string) (string, error) {
	variable, err := p.reader.ReadVariable(name)
	if err == tpl.ErrTemplateVarNotFound(name) {
		question := fmt.Sprintf("%s=", name)
		return ui.Ask(p.io, question)
	} else if err != nil {
		return "", err
	}

	return variable, err
}
