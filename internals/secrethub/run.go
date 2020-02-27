package secrethub

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-cli/internals/cli/masker"
	"github.com/secrethub/secrethub-cli/internals/cli/validation"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
	"github.com/secrethub/secrethub-cli/internals/secrethub/tpl"
	"github.com/secrethub/secrethub-cli/internals/secretspec"

	"github.com/secrethub/secrethub-go/internals/errio"
)

// Errors
var (
	errRun                    = errio.Namespace("run")
	ErrStartFailed            = errRun.Code("start_failed").ErrorPref("error while starting process: %s")
	ErrSignalFailed           = errRun.Code("signal_failed").ErrorPref("error while propagating signal to process: %s")
	ErrReadEnvDir             = errRun.Code("env_dir_read_error").ErrorPref("could not read the environment directory: %s")
	ErrReadEnvFile            = errRun.Code("env_file_read_error").ErrorPref("could not read the environment file %s: %s")
	ErrReadDefaultEnvFile     = errRun.Code("default_env_file_read_error").ErrorPref("could not read default run env-file %s: %s")
	ErrTemplate               = errRun.Code("invalid_template").ErrorPref("could not parse template at line %d: %s")
	ErrParsingTemplate        = errRun.Code("template_parsing_failed").ErrorPref("error while processing template file '%s': %s")
	ErrInvalidTemplateVar     = errRun.Code("invalid_template_var").ErrorPref("template variable '%s' is invalid: template variables may only contain uppercase letters, digits, and the '_' (underscore) and are not allowed to start with a number")
	ErrSecretsNotAllowedInKey = errRun.Code("secret_in_key").Error("secrets are not allowed in run template keys")
)

const (
	defaultEnvFile = "secrethub.env"
	maskString     = "<redacted by SecretHub>"
	// templateVarEnvVarPrefix is used to prefix environment variables
	// that should be used as template variables.
	templateVarEnvVarPrefix = "SECRETHUB_VAR_"
	// prefix of the values of environment variables that will be
	// substituted with secrets
	secretReferencePrefix = "secrethub://"
)

// RunCommand runs a program and passes environment variables to it that are
// defined with --envar or --env-file flags and secrets.yml files.
// The yml files write to .secretsenv/<env-name> when running the set command.
type RunCommand struct {
	io                           ui.IO
	osEnv                        []string
	readFile                     func(filename string) ([]byte, error)
	osStat                       func(filename string) (os.FileInfo, error)
	command                      []string
	envar                        map[string]string
	envFile                      string
	templateVars                 map[string]string
	templateVersion              string
	env                          string
	noMasking                    bool
	maskingTimeout               time.Duration
	newClient                    newClientFunc
	ignoreMissingSecrets         bool
	dontPromptMissingTemplateVar bool
}

// NewRunCommand creates a new RunCommand.
func NewRunCommand(io ui.IO, newClient newClientFunc) *RunCommand {
	return &RunCommand{
		io:           io,
		osEnv:        os.Environ(),
		readFile:     ioutil.ReadFile,
		osStat:       os.Stat,
		envar:        make(map[string]string),
		templateVars: make(map[string]string),
		newClient:    newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *RunCommand) Register(r command.Registerer) {
	const helpShort = "Pass secrets as environment variables to a process."
	const helpLong = "To protect against secrets leaking via stdout and stderr, those output streams are monitored for secrets. Detected secrets are automatically masked by replacing them with \"" + maskString + "\". " +
		"The output is buffered to detect secrets, but to avoid blocking the buffering is limited to a maximum duration as defined by the --masking-timeout flag. " +
		"Therefore, you should regard the masking as a best effort attempt and should always prevent secrets ending up on stdout and stderr in the first place."

	clause := r.Command("run", helpShort)
	clause.HelpLong(helpLong)
	clause.Alias("exec")
	clause.Arg("command", "The command to execute").Required().StringsVar(&cmd.command)
	clause.Flag("envar", "Source an environment variable from a secret at a given path with `NAME=<path>`").Short('e').StringMapVar(&cmd.envar)
	clause.Flag("env-file", "The path to a file with environment variable mappings of the form `NAME=value`. Template syntax can be used to inject secrets.").StringVar(&cmd.envFile)
	clause.Flag("template", "").Hidden().StringVar(&cmd.envFile)
	clause.Flag("var", "Define the value for a template variable with `VAR=VALUE`, e.g. --var env=prod").Short('v').StringMapVar(&cmd.templateVars)
	clause.Flag("env", "The name of the environment prepared by the set command (default is `default`)").Default("default").Hidden().StringVar(&cmd.env)
	clause.Flag("no-masking", "Disable masking of secrets on stdout and stderr").BoolVar(&cmd.noMasking)
	clause.Flag("masking-timeout", "The maximum time output is buffered. Warning: lowering this value increases the chance of secrets not being masked.").Default("1s").DurationVar(&cmd.maskingTimeout)
	clause.Flag("template-version", "The template syntax version to be used. The options are v1, v2, latest or auto to automatically detect the version.").Default("auto").StringVar(&cmd.templateVersion)
	clause.Flag("ignore-missing-secrets", "Do not return an error when a secret does not exist and use an empty value instead.").BoolVar(&cmd.ignoreMissingSecrets)
	clause.Flag("no-prompt", "Do not prompt when a template variable is missing and return an error instead.").BoolVar(&cmd.dontPromptMissingTemplateVar)

	command.BindAction(clause, cmd.Run)
}

// Run reads files from the .secretsenv/<env-name> directory, sets them as environment variables and runs the given command.
// Note that the environment variables are only passed to the child process and not exported globally, which is nice.
func (cmd *RunCommand) Run() error {
	environment, secrets, err := cmd.sourceEnvironment()
	if err != nil {
		return err
	}

	// This makes sure commands encapsulated in quotes also work.
	if len(cmd.command) == 1 {
		cmd.command = strings.Split(cmd.command[0], " ")
	}

	valuesToMask := make([][]byte, 0, len(secrets))
	for _, val := range secrets {
		if val != "" {
			valuesToMask = append(valuesToMask, []byte(val))
		}
	}

	maskedStdout := masker.NewMaskedWriter(cmd.io.Stdout(), valuesToMask, maskString, cmd.maskingTimeout)
	maskedStderr := masker.NewMaskedWriter(os.Stderr, valuesToMask, maskString, cmd.maskingTimeout)

	command := exec.Command(cmd.command[0], cmd.command[1:]...)
	command.Env = environment
	command.Stdin = os.Stdin
	if cmd.noMasking {
		command.Stdout = cmd.io.Stdout()
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
		return commandErr
	}

	return nil
}

// sourceEnvironment returns the environment of the subcommand, with all the secrets sourced
// and the secret values that need to be masked.
func (cmd *RunCommand) sourceEnvironment() ([]string, []string, error) {
	osEnv, passthroughEnv := parseKeyValueStringsToMap(cmd.osEnv)

	envSources := []EnvSource{}

	referenceEnv := newReferenceEnv(osEnv)
	envSources = append(envSources, referenceEnv)

	// TODO: Validate the flags when parsing by implementing the Flag interface for EnvFlags.
	flagSource, err := NewEnvFlags(cmd.envar)
	if err != nil {
		return nil, nil, err
	}
	envSources = append(envSources, flagSource)

	if cmd.envFile == "" {
		_, err := cmd.osStat(defaultEnvFile)
		if err == nil {
			cmd.envFile = defaultEnvFile
		} else if !os.IsNotExist(err) {
			return nil, nil, ErrReadDefaultEnvFile(defaultEnvFile, err)
		}
	}

	if cmd.envFile != "" {
		templateVariableReader, err := newVariableReader(osEnv, cmd.templateVars)
		if err != nil {
			return nil, nil, err
		}

		if !cmd.dontPromptMissingTemplateVar {
			templateVariableReader = newPromptMissingVariableReader(templateVariableReader, cmd.io)
		}

		raw, err := cmd.readFile(cmd.envFile)
		if err != nil {
			return nil, nil, ErrCannotReadFile(cmd.envFile, err)
		}

		parser, err := getTemplateParser(raw, cmd.templateVersion)
		if err != nil {
			return nil, nil, err
		}

		envFile, err := ReadEnvFile(cmd.envFile, bytes.NewReader(raw), templateVariableReader, parser)
		if err != nil {
			return nil, nil, err
		}
		envSources = append(envSources, envFile)
	}

	envDir := filepath.Join(secretspec.SecretEnvPath, cmd.env)
	_, err = cmd.osStat(envDir)
	if err == nil {
		dirSource, err := NewEnvDir(envDir)
		if err != nil {
			return nil, nil, err
		}
		envSources = append(envSources, dirSource)
	}

	var sr tpl.SecretReader = newSecretReader(cmd.newClient)
	if cmd.ignoreMissingSecrets {
		sr = newIgnoreMissingSecretReader(sr)
	}
	secretReader := newBufferedSecretReader(sr)

	// Construct the environment, sourcing variables from the configured sources.
	environment := make(map[string]string)
	for _, source := range envSources {
		pairs, err := source.Env()
		if err != nil {
			return nil, nil, err
		}

		for key, value := range pairs {
			// Only set a variable if it wasn't set by a previous source.
			_, found := environment[key]
			if !found {
				resolvedValue, err := value.resolve(secretReader)
				if err != nil {
					return nil, nil, err
				}
				environment[key] = resolvedValue
			}
		}
	}

	// Source the remaining envars from the OS environment.
	for key, value := range osEnv {
		// Only set a variable if it wasn't set by a configured source.
		_, found := environment[key]
		if !found {
			environment[key] = value
		}
	}

	// Finally add the unparsed variables
	processedOsEnv := append(passthroughEnv, mapToKeyValueStrings(environment)...)

	return processedOsEnv, secretReader.Values(), nil
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
func parseKeyValueStringsToMap(values []string) (map[string]string, []string) {
	parsedLines := make(map[string]string)
	var unparsableLines []string
	for _, kv := range values {
		split := strings.SplitN(kv, "=", 2)
		key := strings.TrimSpace(split[0])
		value := ""
		if len(split) == 2 {
			value = strings.TrimSpace(split[1])
		}

		err := validation.ValidateEnvarName(key)
		if err != nil {
			unparsableLines = append(unparsableLines, kv)
		} else {
			parsedLines[key] = value
		}
	}

	return parsedLines, unparsableLines
}
