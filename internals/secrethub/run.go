package secrethub

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/masker"
	"github.com/secrethub/secrethub-cli/internals/cli/validation"
	"github.com/secrethub/secrethub-cli/internals/secrethub/tpl"
	"github.com/secrethub/secrethub-cli/internals/secretspec"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"

	"gopkg.in/yaml.v2"
)

// Errors
var (
	errRun                = errio.Namespace("run")
	ErrStartFailed        = errRun.Code("start_failed").ErrorPref("error while starting process: %s")
	ErrSignalFailed       = errRun.Code("signal_failed").ErrorPref("error while propagating signal to process: %s")
	ErrReadEnvDir         = errRun.Code("env_dir_read_error").ErrorPref("could not read the environment directory: %s")
	ErrReadEnvFile        = errRun.Code("env_file_read_error").ErrorPref("could not read the environment file %s: %s")
	ErrEnvDirNotFound     = errRun.Code("env_dir_not_found").Error(fmt.Sprintf("could not find specified environment. Make sure you have executed `%s set`.", ApplicationName))
	ErrTemplate           = errRun.Code("invalid_template").ErrorPref("could not parse template at line %d: %s")
	ErrTemplateFile       = errRun.Code("invalid_template_file").ErrorPref("template file '%s' is invalid: %s")
	ErrInvalidTemplateVar = errRun.Code("invalid_template_var").ErrorPref("template variable '%s' is invalid: template variables may only contain uppercase letters, digits, and the '_' (underscore) and are not allowed to start with a number")
)

const (
	maskString = "<redacted by SecretHub>"
	// templateVarEnvVarPrefix is used to prefix environment variables
	// that should be used as template variables.
	templateVarEnvVarPrefix = "SECRETHUB_VAR_"
)

// RunCommand runs a program and passes environment variables to it that are
// defined with --envar or --template flags and secrets.yml files.
// The yml files write to .secretsenv/<env-name> when running the set command.
type RunCommand struct {
	command        []string
	envar          map[string]string
	template       string
	templateVars   map[string]string
	env            string
	noMasking      bool
	maskingTimeout time.Duration
	newClient      newClientFunc
}

// NewRunCommand creates a new RunCommand.
func NewRunCommand(newClient newClientFunc) *RunCommand {
	return &RunCommand{
		envar:        make(map[string]string),
		templateVars: make(map[string]string),
		newClient:    newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *RunCommand) Register(r Registerer) {
	clause := r.Command("run", "Pass secrets as environment variables to a process.")
	clause.Arg("command", "The command to execute").Required().StringsVar(&cmd.command)
	clause.Flag("envar", "Source an environment variable from a secret at a given path with `NAME=<path>`").Short('e').StringMapVar(&cmd.envar)
	clause.Flag("template", "The path to a .yml template file with environment variable mappings of the form `NAME: value`. Templates are automatically injected with secrets when referenced.").StringVar(&cmd.template)
	clause.Flag("var", "Define the value for a template variable with `VAR=VALUE`, e.g. --var env=prod").Short('v').StringMapVar(&cmd.templateVars)
	clause.Flag("env", "The name of the environment prepared by the set command (default is `default`)").Default("default").Hidden().StringVar(&cmd.env)
	clause.Flag("no-masking", "Disable masking of secrets on stdout and stderr").BoolVar(&cmd.noMasking)
	clause.Flag("masking-timeout", "The time to wait for a partial secret that is written to stdout or stderr to be completed for masking.").Default("1s").DurationVar(&cmd.maskingTimeout)

	BindAction(clause, cmd.Run)
}

// Run reads files from the .secretsenv/<env-name> directory, sets them as environment variables and runs the given command.
// Note that the environment variables are only passed to the child process and not exported globally, which is nice.
func (cmd *RunCommand) Run() error {
	// Parse
	envSources := []EnvSource{}

	// TODO: Validate the flags when parsing by implementing the Flag interface for EnvFlags.
	flagSource, err := NewEnvFlags(cmd.envar)
	if err != nil {
		return errio.Error(err)
	}
	envSources = append(envSources, flagSource)

	if cmd.template == "" {
		const defaultTemplate = "secrethub.env"
		_, err := os.Stat(defaultTemplate)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("could not read default run template: %s", err)
			}
		} else {
			cmd.template = defaultTemplate
		}
	}

	osEnv, err := parseKeyValueStringsToMap(os.Environ())
	if err != nil {
		return errio.Error(err)
	}

	templateVars := make(map[string]string)

	for k, v := range osEnv {
		if strings.HasPrefix(k, templateVarEnvVarPrefix) {
			k = strings.TrimPrefix(k, templateVarEnvVarPrefix)
			templateVars[k] = v
		}
	}

	for k, v := range cmd.templateVars {
		templateVars[k] = v
	}

	for k := range templateVars {
		if !validation.IsEnvarNamePosix(k) {
			return ErrInvalidTemplateVar(k)
		}
	}

	client, err := cmd.newClient()
	if err != nil {
		return errio.Error(err)
	}

	if cmd.template != "" {
		tplSource, err := NewEnvFile(cmd.template, templateVars, newSecretReader(client))
		if err != nil {
			return err
		}
		envSources = append(envSources, tplSource)
	}

	envDir := filepath.Join(secretspec.SecretEnvPath, cmd.env)
	_, err = os.Stat(envDir)
	if err == nil {
		dirSource, err := NewEnvDir(envDir)
		if err != nil {
			return errio.Error(err)
		}
		envSources = append(envSources, dirSource)
	}

	// Collect all secrets
	secrets := make(map[string]string)
	for _, source := range envSources {
		for _, path := range source.Secrets() {
			secrets[path] = ""
		}
	}

	for path := range secrets {
		secret, err := client.Secrets().Versions().GetWithData(path)
		if err != nil {
			return errio.Error(err)
		}
		secrets[path] = string(secret.Data)
	}

	// Construct the environment, sourcing variables from the configured sources.
	environment := make(map[string]string)
	for _, source := range envSources {
		pairs, err := source.Env(secrets)
		if err != nil {
			return errio.Error(err)
		}

		for key, value := range pairs {
			// Only set a variable if it wasn't set by a previous source.
			_, found := environment[key]
			if !found {
				environment[key] = value
			}
		}
	}

	// Finally, source the remaining envars from the OS environment.
	for key, value := range osEnv {
		// Only set a variable if it wasn't set by a configured source.
		_, found := environment[key]
		if !found {
			environment[key] = value
		}
	}

	// This makes sure commands encapsulated in quotes also work.
	if len(cmd.command) == 1 {
		cmd.command = strings.Split(cmd.command[0], " ")
	}

	maskStrings := make([][]byte, len(secrets))
	i := 0
	for _, val := range secrets {
		maskStrings[i] = []byte(val)
		i++
	}

	maskedStdout := masker.NewMaskedWriter(os.Stdout, maskStrings, maskString, cmd.maskingTimeout)
	maskedStderr := masker.NewMaskedWriter(os.Stderr, maskStrings, maskString, cmd.maskingTimeout)

	command := exec.Command(cmd.command[0], cmd.command[1:]...)
	command.Env = mapToKeyValueStrings(environment)
	command.Stdin = os.Stdin
	if cmd.noMasking {
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
	} else {
		command.Stdout = maskedStdout
		command.Stderr = maskedStderr

		go maskedStdout.Run()
		go maskedStderr.Run()
	}

	err = command.Start()
	if err != nil {
		return ErrStartFailed(err)
	}

	done := make(chan bool, 1)

	// Pass all signals to child process
	signals := make(chan os.Signal, 1)
	signal.Notify(signals)

	go func() {
		select {
		case s := <-signals:
			err := command.Process.Signal(s)
			if err != nil && !strings.Contains(err.Error(), "process already finished") {
				fmt.Fprintln(os.Stderr, ErrSignalFailed(err))
			}
		case <-done:
			signal.Stop(signals)
			return
		}
	}()

	commandErr := command.Wait()
	done <- true

	if !cmd.noMasking {
		err = maskedStdout.Flush()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		err = maskedStderr.Flush()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	if commandErr != nil {
		// Check if the program exited with an error
		exitErr, ok := commandErr.(*exec.ExitError)
		if ok {
			waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
			if ok {
				// Return the status code returned by the process
				os.Exit(waitStatus.ExitStatus())
				return nil
			}

		}
		return errio.Error(commandErr)
	}

	return nil
}

// mapToKeyValueStrings converts a map to a slice of key=value pairs.
func mapToKeyValueStrings(pairs map[string]string) []string {
	result := make([]string, len(pairs))
	i := 0
	for key, value := range pairs {
		result[i] = key + "=" + value
		i++
	}

	return result
}

// parseKeyValueStringsToMap converts a slice of "key=value" strings to a
// map of "key":"value" pairs. When duplicate keys occur, the last value is
// used.
func parseKeyValueStringsToMap(values []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, kv := range values {
		split := strings.SplitN(kv, "=", 2)
		key := strings.TrimSpace(split[0])
		value := ""
		if len(split) == 2 {
			value = strings.TrimSpace(split[1])
		}

		err := validation.ValidateEnvarName(key)
		if err != nil {
			return nil, errio.Error(err)
		}

		result[key] = value
	}

	return result, nil
}

// EnvSource defines a method of reading environment variables from a source.
type EnvSource interface {
	// Env returns a map of key value pairs.
	Env(secrets map[string]string) (map[string]string, error)
	// Secrets returns a list of paths to secrets that are used in the environment.
	Secrets() []string
}

type envTemplate struct {
	envVars      map[string]tpl.Template
	templateVars map[string]string
	secretReader tpl.SecretReader
}

// Env injects the given secrets in the environment values and returns
// a map of the resulting environment.
func (t envTemplate) Env(secrets map[string]string) (map[string]string, error) {
	result := make(map[string]string)
	for key, template := range t.envVars {
		value, err := template.Evaluate(t.templateVars, t.secretReader)
		if err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, nil
}

// Secrets implements the EnvSource.Secrets function.
// The envTemplate fetches its secrets using a tpl.SecretReader.
func (t envTemplate) Secrets() []string {
	return []string{}
}

// NewEnvFile returns an new environment from a file.
func NewEnvFile(filepath string, vars map[string]string, sr tpl.SecretReader) (EnvFile, error) {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		return EnvFile{}, ErrCannotReadFile(filepath, err)
	}
	env, err := NewEnv(string(content), vars, sr)
	if err != nil {
		return EnvFile{}, err
	}
	return EnvFile{
		path: filepath,
		env:  env,
	}, nil
}

// EnvFile contains an environment that is read from a file.
type EnvFile struct {
	path string
	env  EnvSource
}

// Env returns a map of key value pairs read from the environment file.
func (e EnvFile) Env(secrets map[string]string) (map[string]string, error) {
	env, err := e.env.Env(secrets)
	if err != nil {
		return nil, ErrTemplateFile(e.path, err)
	}
	return env, nil
}

// Secrets returns a list of paths to secrets that are used in the environment.
func (e EnvFile) Secrets() []string {
	return e.env.Secrets()
}

// NewEnv loads an environment of key-value pairs from a string.
// The format of the string can be `key: value` or `key=value` pairs.
func NewEnv(raw string, vars map[string]string, sr tpl.SecretReader) (EnvSource, error) {
	env, parser, err := parseEnvironment(raw)
	if err != nil {
		return nil, err
	}

	secretTemplates := make(map[string]tpl.Template, len(env))
	for _, envvar := range env {
		err = validation.ValidateEnvarName(envvar.key)
		if err != nil {
			return nil, templateError(envvar, err)
		}

		template, err := parser.Parse(envvar.value, envvar.lineNumber, envvar.columnNumber)
		if err != nil {
			return nil, err
		}
		secretTemplates[envvar.key] = template
	}

	return envTemplate{
		envVars:      secretTemplates,
		templateVars: vars,
		secretReader: sr,
	}, nil
}

func templateError(envvar envvar, err error) error {
	if envvar.lineNumber > 0 {
		return ErrTemplate(envvar.lineNumber, err)
	}
	return err
}

type envvar struct {
	key          string
	value        string
	lineNumber   int
	columnNumber int
}

// parseEnvironment parses envvars from a string.
// It first tries the key=value format. When that returns an error,
// the yml format is tried.
// The default parser to be used with the format is also returned.
func parseEnvironment(raw string) ([]envvar, tpl.Parser, error) {
	parser := tpl.NewV2Parser()
	env, err := parseEnv(raw)
	if err != nil {
		var ymlErr error
		parser = tpl.NewV1Parser()
		env, ymlErr = parseYML(raw)
		if ymlErr != nil {
			return nil, nil, err
		}
	}
	return env, parser, nil
}

func parseEnv(raw string) ([]envvar, error) {
	vars := map[string]envvar{}
	scanner := bufio.NewScanner(strings.NewReader(raw))

	i := 1
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, ErrTemplate(i, errors.New("template is not formatted as key=value pairs"))
		}

		key := strings.TrimRight(parts[0], " ")
		value := strings.TrimLeft(parts[1], " ")

		vars[key] = envvar{
			key:          key,
			value:        value,
			lineNumber:   i,
			columnNumber: len(parts[0]) + 1 + len(parts[1]) - len(value) + 1, // the length of the key (including extra spaces) + the length of the = char + the length of the spaces before the value + 1 for the current char
		}
		i++
	}

	i = 0
	res := make([]envvar, len(vars))
	for _, envvar := range vars {
		res[i] = envvar
		i++
	}

	return res, nil
}

func parseYML(raw string) ([]envvar, error) {
	pairs := make(map[string]string)
	err := yaml.Unmarshal([]byte(raw), pairs)
	if err != nil {
		return nil, err
	}

	vars := make([]envvar, len(pairs))
	i := 0
	for key, value := range pairs {
		vars[i] = envvar{
			key:        key,
			value:      value,
			lineNumber: -1,
		}
		i++
	}
	return vars, nil
}

// EnvDir defines environment variables sourced from files in a directory.
type EnvDir map[string]string

// NewEnvDir sources environment variables from files in a given directory,
// using the file name as key and contents as value.
func NewEnvDir(path string) (EnvDir, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, ErrEnvDirNotFound
	} else if err != nil {
		return nil, ErrReadEnvDir(err)
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, ErrReadEnvDir(err)
	}

	env := make(map[string]string)
	for _, f := range files {
		if !f.IsDir() {
			filePath := filepath.Join(path, f.Name())
			fileContent, err := ioutil.ReadFile(filePath)
			if err != nil {
				return nil, ErrReadEnvFile(f.Name(), err)
			}

			env[f.Name()] = string(fileContent)
		}
	}

	return env, nil
}

// Env returns a map of environment variables sourced from the directory.
func (dir EnvDir) Env(secrets map[string]string) (map[string]string, error) {
	return dir, nil
}

// Secrets returns a list of paths to secrets that are used in the environment.
func (dir EnvDir) Secrets() []string {
	return []string{}
}

// EnvFlags defines environment variables sourced from command-line flags.
type EnvFlags map[string]string

// NewEnvFlags parses a map of flag values.
func NewEnvFlags(flags map[string]string) (EnvFlags, error) {
	for name, path := range flags {
		err := validation.ValidateEnvarName(name)
		if err != nil {
			return nil, errio.Error(err)
		}

		err = api.ValidateSecretPath(path)
		if err != nil {
			return nil, err
		}
	}

	return flags, nil
}

// Env returns a map of environment variables sourced from
// command-line flags and set to their corresponding value.
func (ef EnvFlags) Env(secrets map[string]string) (map[string]string, error) {
	result := make(map[string]string)
	for name, path := range ef {
		result[name] = secrets[path]
	}
	return result, nil
}

// Secrets returns the paths to the secrets that are used in the flags.
func (ef EnvFlags) Secrets() []string {
	result := make([]string, len(ef))
	i := 0
	for _, v := range ef {
		result[i] = v
		i++
	}
	return result
}
