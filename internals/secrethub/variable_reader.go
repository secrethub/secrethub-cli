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

// newVariableReader returns a new template variable reader that fetches template variables from the
// specified OS environment variables and commandFlags. An error is returned if any of the provided variable
// names is invalid.
func newVariableReader(osEnv map[string]string, commandTemplateVars map[string]string) (tpl.VariableReader, error) {
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

// ReadVariable fetches a template variable by name and errors if it is not found.
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

func newPromptMissingVariableReader(reader tpl.VariableReader, io ui.IO) tpl.VariableReader {
	return &promptMissingVariableReader{
		reader: reader,
		io:     io,
	}
}

// ReadVariable fetches a template variable and prompts the user if it is not found.
func (p *promptMissingVariableReader) ReadVariable(name string) (string, error) {
	variable, err := p.reader.ReadVariable(name)
	if err == tpl.ErrTemplateVarNotFound(name) {
		question := fmt.Sprintf("What is the value of the \"%s\" template variable?", name)
		return ui.Ask(p.io, question)
	} else if err != nil {
		return "", err
	}

	return variable, err
}
