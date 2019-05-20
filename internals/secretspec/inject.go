package secretspec

import (
	"os"
	"path/filepath"

	"github.com/secrethub/secrethub-cli/internals/tpl"

	"fmt"

	"io/ioutil"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"golang.org/x/text/encoding"
)

var (
	// ErrCannotReadFile is returned when reading a file fails. Takes the path and an error.
	ErrCannotReadFile = errConsumption.Code("cannot_read_file").ErrorPref("cannot read file %s: %v")
	// ErrInjectParseFailed is returned when parsing the contents to inject failed. Takes an error.
	ErrInjectParseFailed = errConsumption.Code("inject_parse_failed").ErrorPref("failed to parse contents: %v")
	// ErrInjectFailed is returned when injecting secrets failed. Takes an error.
	ErrInjectFailed = errConsumption.Code("inject_failed").ErrorPref("failed to inject secrets: %v")
)

var (
	fieldEncoding = "encoding"
)

// InjectParser parses Inject Consumables.
type InjectParser struct{}

// Type returns the parser type.
func (p InjectParser) Type() string {
	return "inject"
}

// Parse parses a config to create an Inject Consumable.
func (p InjectParser) Parse(rootPath string, allowMountAnywhere bool, config map[string]interface{}) (Consumable, error) {
	sourceName, ok := config[fieldSource].(string)
	if !ok {
		return nil, ErrFieldNotSet(fieldSource, fieldSource)
	}

	source, err := filepath.Abs(sourceName)
	if err != nil {
		return nil, ErrCannotFindAbsPath(sourceName, err)
	}

	targetName, ok := config[fieldTarget].(string)
	if !ok {
		return nil, ErrFieldNotSet(fieldTarget, fieldTarget)
	}

	target, err := createTarget(rootPath, targetName, allowMountAnywhere)
	if err != nil {
		return nil, errio.Error(err)
	}

	filemode := DefaultFileMode
	mode, ok := config[fieldFilemode].(string)
	if ok {
		filemode, err = strToFileMode(mode)
		if err != nil {
			return nil, err
		}
	}

	inj := Inject{
		source:   source,
		target:   target,
		filemode: filemode,
	}

	// Read and parse the file to inject.
	bytes, err := ioutil.ReadFile(source)
	if err != nil {
		return nil, ErrCannotReadFile(source, err)
	}

	encodingString, ok := config[fieldEncoding].(string)
	if ok {
		inj.encoding, err = EncodingFromString(encodingString)
		if err != nil {
			return nil, errio.Error(err)
		}
	} else {
		inj.encoding = DetectEncoding(bytes)
		// If the encoding cannot be detected, it most often is UTF8.
		if inj.encoding == nil {
			inj.encoding = EncodingUTF8
		}

		log.Debugf("no character encoding specified for %s, guessed %s", source, inj.encoding)
	}

	decodedBytes, err := inj.encoding.NewDecoder().Bytes(bytes)
	if err != nil {
		return nil, errio.Error(err)
	}

	inj.template, err = tpl.NewParser().Parse(string(decodedBytes))
	if err != nil {
		return nil, errio.Error(err)
	}

	return &inj, nil
}

// Inject implements a consumable that takes a file and
// injects it with secrets, written to the target file.
type Inject struct {
	source   string
	target   string
	filemode os.FileMode

	encoding encoding.Encoding

	template tpl.Template
}

// Set injects all secrets with data from matching secrets in the map
// and writes to the target file. Though the map may contain other
// secrets, it must contain all source secrets of this consumable.
func (inj *Inject) Set(secrets map[string]api.SecretVersion) error {
	input := make(map[string]string, len(secrets))
	for path, secret := range secrets {
		input[path] = string(secret.Data)
	}

	output, err := inj.template.Inject(input)
	if err != nil {
		return errio.Error(err)
	}

	log.Debugf("writing injected file to %s", inj.target)

	encodedBytes, err := inj.encoding.NewEncoder().Bytes([]byte(output))
	if err != nil {
		return errio.Error(err)
	}

	return overwriteFile(inj.target, encodedBytes, inj.filemode)
}

// Sources returns the full paths of the secrets from which the Consumable is sourced.
func (inj *Inject) Sources() map[string]struct{} {
	sources := make(map[string]struct{})
	for _, path := range inj.template.Keys() {
		sources[path] = struct{}{}
	}
	return sources
}

// Clear removes the injected file from the filesystem.
func (inj *Inject) Clear() error {
	err := os.Remove(inj.target)
	if os.IsNotExist(err) {
		log.Warningf("cannot clear file %s as it does not exist", inj.target)
		return nil
	}
	return err
}

// Equals checks whether two Injects have the same target.
func (inj *Inject) Equals(consumable Consumable) bool {
	injectConsumable, ok := consumable.(*Inject)
	if !ok {
		return false
	}
	return injectConsumable.target == inj.target
}

// String returns the string representation of the Inject.
func (inj *Inject) String() string {
	return fmt.Sprintf("inject:%s", inj.target)
}
