package tpl

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/tpl"
)

// Template is a template containing secret paths.
//
// Secret paths can be supplied between brackets:
// {{ path/to/secret }}.
// Secret paths can contain variables. These can be
// supplied using: ${ var }.
// Combined this can look like:
// {{ ${app}/db/secret }}
type Template struct {
	template tpl.Template
	// secrets is a map of template keys (can contain variables) and the corresponding
	// template variable templates.
	secrets map[string]tpl.Template
}

// Parse parses a secret template from a raw string.
// See tpl.Template for the format of the template.
func Parse(raw string) (Template, error) {
	t, err := tpl.NewParser("{{", "}}").Parse(raw)
	if err != nil {
		return Template{}, err
	}

	keys := t.Keys()

	secrets := make(map[string]tpl.Template, len(keys))

	templateVarParser := tpl.NewParser("${", "}")
	for _, k := range keys {
		parsed, err := templateVarParser.Parse(k)
		if err != nil {
			return Template{}, err
		}
		secrets[k] = parsed
	}

	return Template{
		template: t,
		secrets:  secrets,
	}, nil
}

// SecretTemplate is a template that only contains secret keys. Template variables
// are already replaced.
type SecretTemplate struct {
	template tpl.Template
	// secrets is a map of template keys (can contain variables) and the corresponding
	// secret paths (with variables replaced by their values).
	secrets map[string]string
}

// InjectVars takes a map of template variables with their corresponding values. It replaces
// the template variables with their values in the template.
func (t Template) InjectVars(vars map[string]string) (SecretTemplate, error) {
	secrets := make(map[string]string, len(t.secrets))
	for k, template := range t.secrets {
		secretpath, err := template.Inject(vars)
		if err != nil {
			return SecretTemplate{}, err
		}
		secrets[k] = secretpath
	}

	return SecretTemplate{
		template: t.template,
		secrets:  secrets,
	}, nil
}

// InjectSecrets takes a map of secret paths with their corresponding values. It replaces
// the secret paths with the corresponding values in the template.
func (t SecretTemplate) InjectSecrets(secrets map[string]string) (string, error) {
	keys := make(map[string]string, len(t.secrets))
	for k, secretpath := range t.secrets {
		v, ok := secrets[secretpath]
		if !ok {
			return "", fmt.Errorf("no value supplied for secret %s", secretpath)
		}
		keys[k] = v
	}
	return t.template.Inject(keys)
}

// Secrets returns a list of paths to secrets that are used in the template.
func (t SecretTemplate) Secrets() []string {
	set := map[string]struct{}{}
	for _, path := range t.secrets {
		set[path] = struct{}{}
	}

	result := make([]string, len(set))
	i := 0
	for path := range set {
		result[i] = path
		i++
	}
	return result
}
