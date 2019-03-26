package secretspec

import (
	"os"
	"strconv"
	"strings"

	"fmt"

	"github.com/keylockerbv/secrethub-cli/internals/cli/posix"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
)

const (
	fieldFilemode = "filemode"
)

// Errors
var (
	ErrMkdirError            = errConsumption.Code("mkdir_error").ErrorPref("could not create directory %s: %v")
	ErrTargetAlreadyExists   = errConsumption.Code("target_already_exists").ErrorPref("target %s already exists")
	ErrCannotFindAbsPath     = errConsumption.Code("cannot_find_abs_path").ErrorPref("cannot find absolute path of file %s: %v")
	ErrCannotConvertFilemode = errConsumption.Code("cannot_convert_filemode").ErrorPref("cannot convert %s to filemode: %v")
	ErrInvalidTargetPath     = errConsumption.Code("invalid_target_path").ErrorPref("target path %s is invalid")
	ErrInvalidFileMode       = errConsumption.Code("invalid_filemode").ErrorPref("file mode %s is invalid")
)

// FileParser is a Parser to parse File Consumables.
type FileParser struct{}

// Type returns the parser type.
func (p FileParser) Type() string {
	return "file"
}

// Parse parses a config to create a file Consumable.
func (p FileParser) Parse(rootPath string, allowMountAnywhere bool, config map[string]interface{}) (Consumable, error) {
	source, ok := config[fieldSource].(string)
	if !ok {
		return nil, ErrFieldNotSet(fieldSource, fieldSource)
	}
	source = strings.ToLower(source)

	target, _ := config[fieldTarget].(string)

	var filemode os.FileMode
	var err error
	mode, ok := config[fieldFilemode].(string)
	if ok && mode != "" {
		filemode, err = strToFileMode(mode)
		if err != nil {
			return nil, errio.Error(err)
		}
	}

	file, err := newFile(api.SecretPath(source), target, filemode)
	if err != nil {
		return nil, errio.Error(err)
	}

	err = file.createTarget(rootPath, allowMountAnywhere)
	if err != nil {
		return nil, errio.Error(err)
	}

	return file, nil
}

// file implements a Consumable written to a file.
type file struct {
	source   api.SecretPath
	target   string
	filemode os.FileMode
}

// newFile creates a new file consumable and sets default values.
// When target is empty, it defaults to the source name. When
// filemode is empty, it defaults to the DefaultFileMode.
func newFile(source api.SecretPath, target string, filemode os.FileMode) (*file, error) {
	err := source.Validate()
	if err != nil {
		return nil, ErrInvalidSourcePath(err)
	}

	if target == "" {
		target = source.GetSecret()
	}

	if filemode == 0 {
		filemode = DefaultFileMode
	}

	return &file{
		source:   source,
		target:   target,
		filemode: filemode,
	}, nil
}

// createTarget creates the target in the root path and sets the file's target accordingly.
func (f *file) createTarget(rootPath string, allowMountAnywhere bool) error {
	target, err := createTarget(rootPath, f.target, allowMountAnywhere)
	if err != nil {
		return errio.Error(err)
	}
	f.target = target
	return nil
}

// Set writes the contents of a matching secret in the given map to
// the file.
func (f *file) Set(secrets map[api.SecretPath]api.SecretVersion) error {
	log.Debugf("setting file: %s (source) => %s (target)", f.source, f.target)
	version, found := secrets[f.source]
	if !found {
		return ErrSecretNotFound(f.source)
	}

	return overwriteFile(f.target, posix.AddNewLine(version.Data), f.filemode)
}

// Clear removes the file from the filesystem.
func (f *file) Clear() error {
	err := os.Remove(f.target)
	if os.IsNotExist(err) {
		log.Warningf("cannot clear file %s as it does not exist", f.target)
		return nil
	}
	return err
}

// Equals checks whether two Files have the same target.
func (f *file) Equals(consumable Consumable) bool {
	fileConsumable, ok := consumable.(*file)
	if !ok {
		return false
	}
	return strings.EqualFold(fileConsumable.target, f.target)
}

// String returns the string representation of the file.
func (f *file) String() string {
	return fmt.Sprintf("file:%s", f.target)
}

// Sources returns the full path of the secret from which the consumable is sourced.
func (f *file) Sources() map[api.SecretPath]struct{} {
	sources := make(map[api.SecretPath]struct{})
	sources[f.source] = struct{}{}
	return sources
}

// strToFileMode converts a string like 0644 to an os.FileMode.
func strToFileMode(mode string) (os.FileMode, error) {
	filemode, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return 0, ErrCannotConvertFilemode(mode, err)
	}
	return os.FileMode(filemode), nil
}
