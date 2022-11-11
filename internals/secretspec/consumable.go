package secretspec

import (
	"os"
	"path/filepath"

	"github.com/secrethub/secrethub-cli/internals/cli"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"

	"gopkg.in/yaml.v2"
)

const (
	fieldSource = "source"
	fieldTarget = "target"
)

var (
	log = cli.NewLogger()

	// DefaultParsers contains the default supported parsers.
	DefaultParsers = []Parser{
		FileParser{},
		EnvParser{},
		InjectParser{},
	}

	// DefaultFileMode is the default filemode to use for consumables.
	DefaultFileMode os.FileMode = 0400
)

// Errors
var (
	errConsumption = errio.Namespace("consumption")

	ErrDuplicateParser     = errConsumption.Code("duplicate_parser").ErrorPref("duplicate parser type %s")
	ErrCannotConvertField  = errConsumption.Code("cannot_convert_field").ErrorPref("cannot convert field %s with value %s in config to a %T")
	ErrParserNotAvailable  = errConsumption.Code("parser_not_available").ErrorPref("parser %s is not available")
	ErrFieldNotSet         = errConsumption.Code("field_not_set").ErrorPref("field %s is not set or is not a %T")
	ErrInvalidSourcePath   = errConsumption.Code("invalid_source_path").ErrorPref("invalid source path %s")
	ErrEmptyParserType     = errConsumption.Code("empty_spec_field").Error("cannot parse the spec because the parser type is empty")
	ErrCannotUnmarshalSpec = errConsumption.Code("cannot_unmarshal_spec").ErrorPref("cannot unmarshal spec: %v")
	ErrParserNotFound      = errConsumption.Code("parser_not_found").Error("parser not found for the spec")
	ErrPathNotInRoot       = errConsumption.Code("path_not_in_root").ErrorPref("the path %s is not a subdirectory of the root %s")
	ErrDuplicateSpecEntry  = errConsumption.Code("duplicate_spec_entry").ErrorPref("duplicate entry `%s` defined in spec")
	ErrCannotOverwriteFile = errConsumption.Code("cannot_overwrite").ErrorPref("cannot overwrite existing file %s: %s")
	ErrSecretNotFound      = errConsumption.Code("secret_not_found").ErrorPref("secret with path %s is not found in the result")
)

// Consumable is a secret that can be consumed by a process in an environment.
type Consumable interface {
	// Set sets the consumable to any matching secrets.
	Set(secrets map[string]api.SecretVersion) error
	// Clear clears the consumable of any content.
	Clear() error
	// Sources returns a set of full paths of the secrets corresponding to the consumable.
	Sources() map[string]struct{}
	// Equals returns whether to Consumables have the same target. This can be used to check whether they can exist in the same spec.
	Equals(consumable Consumable) bool
	String() string
}

// Parser can create a consumable from a config.
// Each parser has a Type that must be unique.
type Parser interface {
	Parse(rootPath string, allowMountAnywhere bool, config map[string]interface{}) (Consumable, error)
	Type() string
}

// Presenter contains Consumables, created by Parsers.
type Presenter struct {
	parsers            map[string]Parser
	consumables        []Consumable
	rootPath           string
	allowMountAnywhere bool
}

// NewPresenter creates a Presenter from a given set of Parsers.
func NewPresenter(rootPath string, allowMountAnywhere bool, parsers ...Parser) (*Presenter, error) {
	availableParsers := make(map[string]Parser)
	for _, parser := range parsers {
		t := parser.Type()
		_, exists := availableParsers[t]
		if exists {
			return nil, ErrDuplicateParser(t)
		}
		availableParsers[t] = parser
	}

	return &Presenter{
		parsers:            availableParsers,
		consumables:        []Consumable{},
		rootPath:           rootPath,
		allowMountAnywhere: allowMountAnywhere,
	}, nil
}

// Parse initializes a Presenter with consumables, initializing parsers defined by the config.
func (p *Presenter) Parse(data []byte) error {
	in := SpecFile{
		Secrets: []Spec{},
	}
	err := yaml.Unmarshal(data, &in)
	if err != nil {
		return ErrCannotUnmarshalSpec(err)
	}

	// range over the maps inside the secrets array
	for _, entry := range in.Secrets {

		consumable, err := p.parse(entry)
		if err != nil {
			return err
		}

		for _, c := range p.consumables {
			if c.Equals(consumable) {
				return ErrDuplicateSpecEntry(c)
			}
		}

		p.consumables = append(p.consumables, consumable)
	}

	return nil
}

// Clear clears all consumables.
func (p *Presenter) Clear() error {
	for _, consumable := range p.consumables {
		err := consumable.Clear()
		if err != nil {
			return err
		}
	}
	return nil
}

// parse returns a consumable for the given parser type
func (p *Presenter) parse(config Spec) (Consumable, error) {
	log.Debugf("parsing spec entry: %v", config)

	for parserType, parser := range p.parsers {
		spec, ok := config[parserType]
		if ok {
			consumable, err := parser.Parse(p.rootPath, p.allowMountAnywhere, spec)
			if err != nil {
				return nil, err
			}

			return consumable, nil
		}
	}

	return nil, ErrParserNotFound
}

// Set sets all consumables that correspond to the given secrets.
func (p *Presenter) Set(secrets map[string]api.SecretVersion) error {
	for _, consumable := range p.consumables {
		err := consumable.Set(secrets)
		if err != nil {
			return err
		}
	}

	return nil
}

// Sources returns the full paths of all secrets sourced within the presenter.
func (p *Presenter) Sources() map[string]struct{} {
	total := make(map[string]struct{})
	for _, consumable := range p.consumables {
		for source := range consumable.Sources() {
			total[source] = struct{}{}
		}
	}

	return total
}

// EmptyConsumables returns a list of all consumables that contain no sources.
func (p *Presenter) EmptyConsumables() []Consumable {
	var l []Consumable
	for _, c := range p.consumables {
		if len(c.Sources()) == 0 {
			l = append(l, c)
		}
	}
	return l
}

// SpecFile is used to unmarshal a spec file correctly.
type SpecFile struct {
	Secrets []Spec `json:"secrets" yaml:"secrets"`
}

// Spec is used to unmarshal a consumable block correctly.
type Spec map[string]map[string]interface{}

// parseTargetOnRootPath applies the target on top of the rootPath
// If the target is an absolute path, it is checked whether it is a child of the root path
// Examples (rootPath, target => parseTargetOnRootPath(rootPath, target)):
// - /a/b, c => /a/b/c
// - /a/b, /a/b/d => /a/b/d
// - /a/b, ../c => ErrPathNotInRoot
// - /a/b, /a => ErrPathNotInRoot
func parseTargetOnRootPath(rootPath, target string, allowMountAnywhere bool) (string, error) {
	if !filepath.IsAbs(target) {
		target = filepath.Clean(filepath.Join(rootPath, target))
	}

	if !allowMountAnywhere {
		relPath, err := filepath.Rel(rootPath, target)
		if err != nil {
			return "", ErrCannotFindAbsPath(target, err)
		}

		// If the relative path starts with .. it means it is below the root
		if len(relPath) > 1 && relPath[:2] == ".." {
			return "", ErrPathNotInRoot(target, rootPath)
		}
	}

	return target, nil
}

// createTarget parses target on a rootPath and creates the directory if it does not exist yet
func createTarget(rootPath, target string, allowMountAnywhere bool) (string, error) {
	target, err := parseTargetOnRootPath(rootPath, target, allowMountAnywhere)
	if err != nil {
		return "", err
	}

	err = os.MkdirAll(filepath.Dir(target), 0771)
	if err != nil {
		return "", ErrMkdirError(target, err)
	}

	return target, nil
}

// overwriteFile overwrites a file even if it is read-only.
// This is achieved by first removing it using RemoveAll.
// This also means that the filemode is also guaranteed to be set to the given filemode after writing.
func overwriteFile(filename string, data []byte, perm os.FileMode) error {
	_, err := os.Stat(filename)
	if !os.IsNotExist(err) {
		err := os.RemoveAll(filename)
		if err != nil {
			return ErrCannotOverwriteFile(filename, err)
		}
	}
	err = os.WriteFile(filename, data, perm)
	if err != nil {
		return ErrCannotOverwriteFile(filename, err)
	}
	return nil
}
