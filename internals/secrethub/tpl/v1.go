package tpl

import (
	"github.com/secrethub/secrethub-cli/internals/tpl"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// Errors
var (
	ErrTemplateVarsNotSupported = errio.Namespace("template").Code("template_vars_not_supported").Error("the v1 template syntax does not support template variables")
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
func (p parserV1) Parse(raw string) (VarTemplate, error) {
	t, err := tpl.NewParser("${", "}").Parse(raw)
	if err != nil {
		return nil, err
	}

	return templateV1{
		template: t,
	}, nil
}

// secretTemplateV1 is a template that only contains secret keys.
type secretTemplateV1 struct {
	template tpl.Template
}

// InjectVars takes a map of template variables with their corresponding values. It replaces
// the template variables with their values in the template.
func (t templateV1) InjectVars(vars map[string]string) (SecretTemplate, error) {
	if len(vars) > 0 {
		return nil, ErrTemplateVarsNotSupported
	}

	return secretTemplateV1(t), nil
}

// InjectSecrets takes a map of secret paths with their corresponding values. It replaces
// the secret paths with the corresponding values in the template.
func (t secretTemplateV1) InjectSecrets(secrets map[string]string) (string, error) {
	return t.template.Inject(secrets)
}

// Secrets returns a list of paths to secrets that are used in the template.
func (t secretTemplateV1) Secrets() []string {
	return t.template.Keys()
}
