package secrethub

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/secrethub/secrethub-cli/internals/cli/validation"
	"github.com/secrethub/secrethub-cli/internals/secretspec"
	"github.com/secrethub/secrethub-cli/internals/tpl"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"

	yaml "gopkg.in/yaml.v2"
)

// Errors
var (
	errRun            = errio.Namespace("read_secret")
	ErrStartFailed    = errRun.Code("start_failed").ErrorPref("error while starting process: %s")
	ErrSignalFailed   = errRun.Code("signal_failed").ErrorPref("error while propagating signal to process: %s")
	ErrReadEnvDir     = errRun.Code("env_dir_read_error").ErrorPref("could not read the environment directory: %s")
	ErrReadEnvFile    = errRun.Code("env_file_read_error").ErrorPref("could not read the environment file %s: %s")
	ErrEnvDirNotFound = errRun.Code("env_dir_not_found").Error(fmt.Sprintf("could not find specified environment. Make sure you have executed `%s set`.", ApplicationName))
	ErrTemplate       = errRun.Code("invalid_template").ErrorPref("could not parse template: %s")
	ErrEnvFileFormat  = errRun.Code("invalid_env_file_format").Error("template is not formatted as key=value or key: value pairs")
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

		tplSource := NewEnv(string(file))
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

	client, err := cmd.newClient()
	if err != nil {
		return errio.Error(err)
	}

	// Construct the environment, sourcing variables from the configured sources.
	environment := make(map[string]string)
	for _, source := range envSources {
		pairs, err := source.Env(client)
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

	command := exec.Command(cmd.command[0], cmd.command[1:]...)
	command.Env = mapToKeyValueStrings(environment)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

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
			if err != nil {
				fmt.Fprintln(os.Stderr, ErrSignalFailed(err))
			}
		case <-done:
			signal.Stop(signals)
			return
		}
	}()

	err = command.Wait()
	done <- true

	if err != nil {
		// Check if the program exited with an error
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
			if ok {
				// Return the status code returned by the process
				os.Exit(waitStatus.ExitStatus())
				return nil
			}

		}
		return errio.Error(err)
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
	// Env returns a map of key value pairs, with the given secrets
	// set to their corresponding value.
	Env(client secrethub.Client) (map[string]string, error)
}

// Env describes a set of key value pairs.
//
// The file can be formatted as `key: value` or `key=value` pairs.
// Secrets can be injected into the values by using the template syntax.
type Env struct {
	raw string
}

// NewEnv loads an environment of key-value pairs from a string.
// The format of the string can be `key: value` or `key=value` pairs.
func NewEnv(raw string) Env {
	return Env{
		raw: raw,
	}
}

// Env returns a map of environment key value pairs, with the given secrets
// set to their corresponding value.
func (e Env) Env(client secrethub.Client) (map[string]string, error) {

	// Parse key-value pairs
	pairs, err := parseEnvPairs(e.raw)
	if err != nil {
		pairs, err = parseYMLPairs(e.raw)
		if err != nil {
			return nil, ErrEnvFileFormat
		}
	}

	// Validate keys
	for key := range pairs {
		err := validation.ValidateEnvarName(key)
		if err != nil {
			return nil, err
		}
	}

	// Inject secrets into values
	result := make(map[string]string)
	for key, value := range pairs {
		t, err := tpl.NewParser().Parse(value)
		if err != nil {
			return nil, ErrTemplate(err)
		}

		secrets := make(map[string][]byte)
		for _, path := range t.Secrets() {
			secret, err := client.Secrets().Versions().GetWithData(path)
			if err != nil {
				return nil, err
			}
			secrets[path] = secret.Data
		}

		injected, err := t.Inject(secrets)
		if err != nil {
			return nil, err
		}
		result[key] = injected
	}

	return result, nil
}

func parseEnvPairs(raw string) (map[string]string, error) {
	pairs := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, ErrEnvFileFormat
		}
		pairs[parts[0]] = parts[1]
	}
	return pairs, nil
}

func parseYMLPairs(raw string) (map[string]string, error) {
	pairs := make(map[string]string)
	err := yaml.Unmarshal([]byte(raw), pairs)
	return pairs, err
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
func (dir EnvDir) Env(client secrethub.Client) (map[string]string, error) {
	return dir, nil
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
func (ef EnvFlags) Env(client secrethub.Client) (map[string]string, error) {
	result := make(map[string]string)
	for name, path := range ef {
		secret, err := client.Secrets().Versions().GetWithData(path)
		if err != nil {
			return nil, err
		}
		result[name] = string(secret.Data)
	}
	return result, nil
}
