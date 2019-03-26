package secrettpl

import (
	"strings"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// Errors
var (
	errInject         = errio.Namespace("inject")
	ErrParseFailed    = errInject.Code("parse_failed").ErrorPref("failed to parse contents: %v")
	ErrSecretNotFound = errInject.Code("secret_not_found").ErrorPref("no secret found to inject for path %s")
)

const (
	// DefaultStartDelimiter defines the characters a template block starts with.
	DefaultStartDelimiter = "${"
	// DefaultEndDelimiter defines the characters a template block ends with.
	DefaultEndDelimiter = "}"
	// DefaultTrimChars defines the cutset of characters that will be trimmed from template blocks.
	DefaultTrimChars = " "
)

var (
	// defaultDelimiters contain the default delimiters for injection.
	defaultDelimiters = delimiters{
		start:     DefaultStartDelimiter,
		end:       DefaultEndDelimiter,
		trimChars: DefaultTrimChars,
	}
)

// Template helps with injecting values into strings that contain the template syntax.
type Template struct {
	Raw     string
	Secrets []api.SecretPath

	delimiters delimiters
}

// New creates a new template and parses it.
func New(raw string) (*Template, error) {
	tpl := &Template{
		Raw:        raw,
		delimiters: defaultDelimiters,
	}

	secrets, err := parse(tpl.Raw, tpl.delimiters)

	if err != nil {
		return nil, errio.Error(err)
	}

	tpl.Secrets = secrets
	return tpl, nil
}

// Inject inserts the given secrets at their corresponding places in
// the raw template and returns the injected template.
func (t *Template) Inject(secrets map[api.SecretPath][]byte) (string, error) {
	return injectRecursive(t.Raw, t.delimiters, secrets)
}

// parse is a recursive helper function that parses a string to a list of SecretPaths contained between inject delimiters.
func parse(raw string, delims delimiters) ([]api.SecretPath, error) {
	// Find the first occurrence of the start delimiter and the first occurrence of the end delimiter thereafter.
	start, end := delims.find(raw)
	if end == 0 {
		return nil, nil
	}

	// The text from the start delimiter til the end delimiter (including both delimiters themselves).
	rawPath := raw[start:end]

	var paths []api.SecretPath
	path, err := delims.parsePath(rawPath)
	if err != nil {
		// Not a valid SecretPath between delimiters.
		// So parse again, starting after the currently used start delimiter.
		paths, err = parse(raw[start+len(delims.start):], delims)
		if err != nil {
			return nil, err
		}

	} else {
		paths, err = parse(raw[end:], delims)
		if err != nil {
			return nil, err
		}

		// Eliminate duplicates
		duplicate := false
		for _, p := range paths {
			if p == path {
				duplicate = true
				break
			}
		}

		if !duplicate {
			paths = append([]api.SecretPath{path}, paths...)
		}
	}

	return paths, nil
}

// injectRecursive is a helper function to recursively inject secrets into strings.
func injectRecursive(raw string, delims delimiters, secrets map[api.SecretPath][]byte) (string, error) {

	// Find the first occurrence of the start delimiter and the first occurrence of the end delimiter thereafter.
	start, end := delims.find(raw)
	if end == 0 {
		return raw, nil
	}

	// The text from the start delimiter til the end delimiter (including both delimiters themselves).
	rawPath := raw[start:end]

	path, err := delims.parsePath(rawPath)
	if err != nil {
		// Not a valid SecretPath between delimiters.
		// So parse again, starting after the currently used start delimiter.
		res, err := injectRecursive(raw[start+len(delims.start):], delims, secrets)
		if err != nil {
			return "", err
		}

		return raw[:start] + delims.start + res, nil
	}

	// Path between delimiters is valid, so add it to the the list.
	secret, ok := secrets[path]
	if !ok {
		return "", ErrSecretNotFound(path)
	}

	// Continue parsing after the end delimiter.
	res, err := injectRecursive(raw[end:], delims, secrets)
	if err != nil {
		return "", err
	}

	injected := raw[:start] + string(secret) + res
	return injected, nil

}

// delimiters is able to find segments of text between the configured delimiters.
type delimiters struct {
	start     string
	end       string
	trimChars string
}

// find finds the first occurrence of the start delimiter in the text
// and the first occurrence of the end delimiter after that.
// If not both can be found, both returns values are 0.
func (d delimiters) find(raw string) (int, int) {
	// Find the first occurrence of the first start delimiter.
	start := strings.Index(raw, d.start)
	if start < 0 {
		return 0, 0
	}

	// Find the first occurrence of the end delimiter.
	length := strings.Index(raw[start:], d.end) + 1
	if length < len(d.start)+len(d.end) {
		return 0, 0
	}

	end := start + length

	return start, end
}

// parsePath is a helper function to parse an api.SecretPath
// The start and end delimiters should be included in the passed string.
func (d delimiters) parsePath(raw string) (api.SecretPath, error) {
	// Make sure we don't go out of bounds on the slice.
	// This path should not be reached when parsePath is used correctly.
	if len(d.start) > len(raw)-len(d.end) {
		return api.SecretPath(""), ErrParseFailed(raw)
	}

	// Trim ${ and }
	raw = raw[len(d.start) : len(raw)-len(d.end)]
	// Trim spaces
	raw = strings.Trim(raw, d.trimChars)
	return api.NewSecretPath(raw)
}
