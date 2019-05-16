package secrethub

import (
	"os/exec"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/masker"
	"github.com/secrethub/secrethub-cli/internals/cli/validation"

	"github.com/secrethub/secrethub-cli/internals/tpl"
	"github.com/secrethub/secrethub-go/internals/api"
	"gopkg.in/yaml.v2"

	"os"

	"syscall"

	"os/signal"

	"strings"

	"fmt"
	"io/ioutil"

	"path/filepath"

	"github.com/secrethub/secrethub-cli/internals/secretspec"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// Errors
var (
	errRun            = errio.Namespace("read_secret")
	ErrStartFailed    = errRun.Code("start_failed").ErrorPref("error while starting process: %s")
	ErrSignalFailed   = errRun.Code("signal_failed").ErrorPref("error while propagating signal to process: %s")
	ErrReadEnvDir     = errRun.Code("env_dir_read_error").ErrorPref("could not read the environment directory: %s")
	ErrReadEnvFile    = errRun.Code("env_file_read_error").ErrorPref("could not read the environment file %s: %s")
	ErrEnvDirNotFound = errRun.Code("env_dir_not_found").Error(fmt.Sprintf("could not find specified environment. Make sure you have executed `%s set`.", ApplicationName))
	ErrEnvFileFormat  = errRun.Code("invalid_env_file_format").ErrorPref("env-file templates must be a valid yaml file with a map of string key and value pairs: %v")
)

var (
	maskString = "<redacted by SecretHub>"
)

// RunCommand runs a program and passes environment variables to it that are
// defined with --envar or --template flags and secrets.yml files.
// The yml files write to .secretsenv/<env-name> when running the set command.
type RunCommand struct {
	command   []string
	envar     map[string]string
	template  string
	env       string
	newClient newClientFunc
}

// NewRunCommand creates a new RunCommand.
func NewRunCommand(newClient newClientFunc) *RunCommand {
	return &RunCommand{
		envar:     make(map[string]string),
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *RunCommand) Register(r Registerer) {
	clause := r.Command("run", "Pass secrets as environment variables to a process.")
	clause.Arg("command", "The command to execute").Required().StringsVar(&cmd.command)
	clause.Flag("envar", "Source an environment variable from a secret at a given path with `NAME=<path>`").Short('e').StringMapVar(&cmd.envar)
	clause.Flag("template", "The path to a .yml template file with environment variable mappings of the form `NAME: value`. Templates are automatically injected with secrets when referenced.").StringVar(&cmd.template)
	clause.Flag("env", "The name of the environment prepared by the set command (default is `default`)").Default("default").Hidden().StringVar(&cmd.env)

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
		const defaultTemplate = "secrethub-env.yml"
		if _, err := os.Stat(defaultTemplate); err == nil {
			cmd.template = defaultTemplate
		}
	}

	if cmd.template != "" {
		file, err := ioutil.ReadFile(cmd.template)
		if err != nil {
			return ErrCannotReadFile(cmd.template, err)
		}

		tplSource, err := NewEnvTemplate(string(file))
		if err != nil {
			return errio.Error(err)
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
	secrets := make(map[api.SecretPath][]byte)
	for _, source := range envSources {
		for _, path := range source.Secrets() {
			secrets[path] = []byte{}
		}
	}

	client, err := cmd.newClient()
	if err != nil {
		return errio.Error(err)
	}

	for path := range secrets {
		secret, err := client.Secrets().Versions().GetWithData(path.Value())
		if err != nil {
			return errio.Error(err)
		}
		secrets[path] = secret.Data
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
	osEnv, err := parseKeyValueStringsToMap(os.Environ())
	if err != nil {
		return errio.Error(err)
	}

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
		maskStrings[i] = val
		i++
	}

	maskedStdOut := masker.NewMaskedWriter(os.Stdout, maskStrings, maskString, time.Millisecond*500)
	maskedStdErr := masker.NewMaskedWriter(os.Stderr, maskStrings, maskString, time.Millisecond*500)

	command := exec.Command(cmd.command[0], cmd.command[1:]...)
	command.Env = mapToKeyValueStrings(environment)
	command.Stdin = os.Stdin
	command.Stdout = maskedStdOut
	command.Stderr = maskedStdErr

	go maskedStdOut.Run()
	go maskedStdErr.Run()

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

	err = maskedStdOut.Flush()
	if err != nil {
		fmt.Println(err)
	}
	err = maskedStdErr.Flush()
	if err != nil {
		fmt.Println(err)
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
	// Secrets returns the secrets contained in the source
	// that need to be set to their corresponding value.
	Secrets() []api.SecretPath
	// Env returns a map of key value pairs, with the given secrets
	// set to their corresponding value.
	Env(secrets map[api.SecretPath][]byte) (map[string]string, error)
}

// EnvTemplate defines a method to load environment variables from a
// file of key=value statements, separated by newlines and optionally
// containing template syntax to inject secrets into.
type EnvTemplate struct {
	Template *tpl.Template
}

// NewEnvTemplate parses a raw string template.
func NewEnvTemplate(raw string) (*EnvTemplate, error) {
	template, err := tpl.New(raw)
	if err != nil {
		return nil, err
	}

	return &EnvTemplate{
		Template: template,
	}, nil
}

// Secrets returns the secret paths contained in the template.
func (tpl EnvTemplate) Secrets() []api.SecretPath {
	return tpl.Template.Secrets
}

// Env returns a map of environment key value pairs, with the given secrets
// set to their corresponding value.
func (tpl EnvTemplate) Env(secrets map[api.SecretPath][]byte) (map[string]string, error) {
	raw, err := tpl.Template.Inject(secrets)
	if err != nil {
		return nil, errio.Error(err)
	}

	return parseEnvFile(raw)
}

// parseEnvFile parses an environment file with key=value statements,
// separated by a newline.
func parseEnvFile(raw string) (map[string]string, error) {
	result := make(map[string]string)

	err := yaml.Unmarshal([]byte(raw), result)
	if err != nil {
		return nil, ErrEnvFileFormat(err)
	}

	for name := range result {
		err := validation.ValidateEnvarName(name)
		if err != nil {
			return nil, errio.Error(err)
		}
	}

	return result, nil
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

// Secrets returns the secrets that need to be set to their
// corresponding value. Because the env dir can only contain
// values, this returns an empty slice.
func (dir EnvDir) Secrets() []api.SecretPath {
	return []api.SecretPath{}
}

// Env returns a map of environment variables sourced from the directory.
func (dir EnvDir) Env(secrets map[api.SecretPath][]byte) (map[string]string, error) {
	return dir, nil
}

// EnvFlags defines environment variables sourced from command-line flags.
type EnvFlags map[string]api.SecretPath

// NewEnvFlags parses a map of flag values.
func NewEnvFlags(flags map[string]string) (EnvFlags, error) {
	result := make(map[string]api.SecretPath)

	for name, path := range flags {
		err := validation.ValidateEnvarName(name)
		if err != nil {
			return nil, errio.Error(err)
		}

		secretPath, err := api.NewSecretPath(path)
		if err != nil {
			return nil, errio.Error(err)
		}

		result[name] = secretPath
	}

	return result, nil
}

// Secrets returns the secrets contained in the source
// that need to be set to their corresponding value.
func (ef EnvFlags) Secrets() []api.SecretPath {
	secrets := make([]api.SecretPath, len(ef))
	i := 0
	for _, path := range ef {
		secrets[i] = path
		i++
	}
	return secrets
}

// Env returns a map of environment variables sourced from
// command-line flags and set to their corresponding value.
func (ef EnvFlags) Env(secrets map[api.SecretPath][]byte) (map[string]string, error) {
	result := make(map[string]string)
	for name, path := range ef {
		result[name] = string(secrets[path])
	}
	return result, nil
}
