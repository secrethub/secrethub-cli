package tpl

import (
	"github.com/secrethub/secrethub-cli/internals/tpl"
)

// NewV1Parser returns a parser for the v1 template syntax.
//
// V1 templates can contain secret paths between ${}:
// ${ path/to/secret }
//
// V1 templates do not support template variables.
func NewV1Parser() Parser {
	return parserV1{}
}

type templateV1 struct {
	template tpl.Template
}

type parserV1 struct{}

// Parse parses a secret template from a raw string.
// See tpl.Template for the format of the template.
func (p parserV1) Parse(raw string, _, _ int) (Template, error) {
	t, err := tpl.NewParser("${", "}").Parse(raw)
	if err != nil {
		return nil, err
	}

	return templateV1{
		template: t,
	}, nil
}

// InjectVars takes a map of template variables with their corresponding values. It replaces
// the template variables with their values in the template.
func (t templateV1) Evaluate(_ VariableReader, sr SecretReader) (string, error) {
	keys := t.template.Keys()
	secrets := make(map[string]string, len(keys))
	for _, path := range keys {
		secret, err := sr.ReadSecret(path)
		if err != nil {
			return "", err
		}
		secrets[path] = secret
	}

	return t.template.Inject(secrets)
}
