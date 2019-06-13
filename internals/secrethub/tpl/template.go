package tpl

// Parser parses a raw string to a template.
type Parser interface {
	Parse(raw string) (VarTemplate, error)
}

// VarTemplate is a template containing variables. Once variables are injected,
// secret paths can be retrieved and injected as well to retrieve the resulting string.
type VarTemplate interface {
	InjectVars(vars map[string]string) (SecretTemplate, error)
}

// SecretTemplate is a template containing secret paths. The plaintext values corresponding
// to these paths can be injected to retrieve the resulting string.
type SecretTemplate interface {
	InjectSecrets(secrets map[string]string) (string, error)
	Secrets() []string
}

// NewParser returns a parser for the latest template syntax.
func NewParser() Parser {
	return NewV2Parser()
}
