package secrethub

import (
	"fmt"
	"strings"

	"github.com/secrethub/secrethub-cli/internals/cli/validation"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/tpl"
)

type variableReader struct {
	vars map[string]string
}

func newVariableReader(osEnv map[string]string, commandTemplateVars map[string]string) (*variableReader, error) {
	templateVars := make(map[string]string)

	for k, v := range osEnv {
		if strings.HasPrefix(k, templateVarEnvVarPrefix) {
			k = strings.TrimPrefix(k, templateVarEnvVarPrefix)
			templateVars[strings.ToLower(k)] = v
		}
	}

	for k, v := range commandTemplateVars {
		templateVars[strings.ToLower(k)] = v
	}

	for k := range templateVars {
		if !validation.IsEnvarNamePosix(k) {
			return nil, ErrInvalidTemplateVar(k)
		}
	}

	return &variableReader{
		vars: templateVars,
	}, nil
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
