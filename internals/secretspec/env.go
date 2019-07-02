package secretspec

import (
	"os"
	"sort"
	"strings"

	"path/filepath"

	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli/validation"
	"github.com/secrethub/secrethub-go/internals/api"
)

const (
	fieldName = "name"
	fieldVars = "vars"

	defaultEnvName = "default"

	// SecretEnvPath is the path used to store the environment variable files
	SecretEnvPath = ".secretenv"
)

var (
	// DefaultEnvDirFileMode is the filemode used for the environment directory.
	DefaultEnvDirFileMode os.FileMode = 0700
)

// Errors
var (
	ErrCannotClearEnvironmentVariable = errConsumption.Code("cannot_clear_env_var").ErrorPref("the environment variable could not be cleared: %s")
	ErrCannotSetEnvironmentVariable   = errConsumption.Code("cannot_set_env_var").ErrorPref("the environment variable could not be set: %s")
	ErrCannotCreateEnvDir             = errConsumption.Code("cannot_create_env_dir").ErrorPref("could not create the required directory for storing environment variables: %s")
)

// EnvParser implements a Parser for Env Consumables.
type EnvParser struct{}

// Type returns the parser type.
func (p EnvParser) Type() string {
	return "env"
}

// Parse parses a config to create an Env Consumable.
func (p EnvParser) Parse(rootPath string, allowMountAnywhere bool, config map[string]interface{}) (Consumable, error) {
	name, _ := config[fieldName].(string)

	vars, ok := config[fieldVars].(map[interface{}]interface{})
	if !ok {
		return nil, ErrFieldNotSet(fieldVars, vars)
	}

	envars := make([]*envar, len(vars))
	i := 0
	for name, val := range vars {
		source, ok := val.(string)
		if !ok {
			return nil, ErrCannotConvertField(name, val, source)
		}

		target, ok := name.(string)
		if !ok {
			return nil, ErrCannotConvertField(name, name, target)
		}

		v, err := newEnvar(source, target)
		if err != nil {
			return nil, err
		}
		envars[i] = v
		i++
	}

	return newEnv(name, rootPath, envars...), nil
}

// envar is a variable within an environment
type envar struct {
	source string
	target string
}

// newEnvar creates a new envar, validating the source and target.
func newEnvar(source string, target string) (*envar, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	err := api.ValidateSecretPath(source)
	if err != nil {
		return nil, ErrInvalidSourcePath(err)
	}

	target = strings.TrimSpace(target)
	err = validation.ValidateEnvarName(target)
	if err != nil {
		return nil, err
	}

	return &envar{
		source: source,
		target: target,
	}, nil
}

// env implements a Consumable containing multiple environment variables
type env struct {
	name    string
	dirPath string
	vars    []*envar
}

// newEnv creates a new environment consumable, with the location for variable
// files rooted at the given rootPath. When the given name is empty, it defaults
// to the defaultEnvName.
func newEnv(name string, rootPath string, vars ...*envar) *env {
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultEnvName
	}

	// Vars are sorted for predictability and easy testing.
	sort.Sort(sortEnvarsByTarget(vars))

	dirPath := ""
	if filepath.Base(rootPath) == SecretEnvPath {
		dirPath = filepath.Join(rootPath, name)
	} else {
		dirPath = filepath.Join(rootPath, SecretEnvPath, name)
	}

	return &env{
		name:    name,
		dirPath: dirPath,
		vars:    vars,
	}
}

// Set sets all environment variables to the matching secrets
// contained in the given argument. Though the map may contain
// other secrets, it must contain all source secrets of this
// consumable.
func (e env) Set(secrets map[string]api.SecretVersion) error {
	err := os.MkdirAll(e.dirPath, DefaultEnvDirFileMode)
	if err != nil {
		return ErrCannotCreateEnvDir(err)
	}

	for _, v := range e.vars {
		log.Debugf("setting env var: %s (source) => %s (target)", v.source, v.target)
		version, found := secrets[v.source]
		if !found {
			return ErrSecretNotFound(v.source)
		}

		err := overwriteFile(e.getVarPath(v), version.Data, DefaultFileMode)
		if err != nil {
			return ErrCannotSetEnvironmentVariable(err)
		}
	}
	return nil
}

// Clear removes the environment variable directory from the filesystem.
func (e *env) Clear() error {
	err := os.RemoveAll(e.dirPath)
	if err != nil {
		return ErrCannotClearEnvironmentVariable(err)
	}

	return nil
}

// Sources returns the full path of the secret from which the consumable is sourced.
func (e *env) Sources() map[string]struct{} {
	sources := make(map[string]struct{})
	for _, v := range e.vars {
		sources[v.source] = struct{}{}
	}
	return sources
}

// Equals checks whether two envs have the same name.
func (e *env) Equals(consumable Consumable) bool {
	envConsumable, ok := consumable.(*env)
	if !ok {
		return false
	}
	return envConsumable.name == e.name
}

// String returns the string representation of the env.
func (e *env) String() string {
	return fmt.Sprintf("env:%s", e.name)
}

// getVarPath returns the path to a secret environment file given its name and env
func (e *env) getVarPath(v *envar) string {
	return filepath.Join(e.dirPath, v.target)
}

type sortEnvarsByTarget []*envar

func (s sortEnvarsByTarget) Len() int {
	return len(s)
}
func (s sortEnvarsByTarget) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s sortEnvarsByTarget) Less(i, j int) bool {
	return s[i].target < s[j].target
}
