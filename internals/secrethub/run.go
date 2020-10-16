package secrethub

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/masker"
	"github.com/secrethub/secrethub-cli/internals/secrethub/tpl"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/errio"

	"github.com/secrethub/secrethub-cli/internals/cli/validation"
	// "github.com/spf13/cobra"
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
	io                   ui.IO
	osEnv                []string
	command              cli.StringArrArgValue
	environment          *environment
	noMasking            bool
	maskerOptions        masker.Options
	newClient            newClientFunc
	ignoreMissingSecrets bool
}

// NewRunCommand creates a new RunCommand.
func NewRunCommand(io ui.IO, newClient newClientFunc) *RunCommand {
	return &RunCommand{
		io:          io,
		osEnv:       os.Environ(),
		environment: newEnvironment(io, newClient),
		newClient:   newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *RunCommand) Register(r cli.Registerer) {
	const helpShort = "Pass secrets as environment variables to a process."
	const helpLong = "To protect against secrets leaking via stdout and stderr, those output streams are monitored for secrets. Detected secrets are automatically masked by replacing them with \"" + maskString + "\". " +
		"The output is buffered to scan for secrets and can be adjusted using the masking-buffer-period flag. " +
		"You should regard the masking as a best effort attempt and should always prevent secrets ending up on stdout and stderr in the first place."

	clause := r.Command("run", helpShort)
	clause.HelpLong(helpLong)
	clause.Alias("exec")
	////clause.Cmd.Args = cobra.MinimumNArgs(1)
	//clause.Arg("command", "The command to execute").Required().StringsVar(&cmd.command)
	clause.Flags().BoolVar(&cmd.noMasking, "no-masking", false, "Disable masking of secrets on stdout and stderr")
	clause.Flags().BoolVar(&cmd.maskerOptions.DisableBuffer, "no-output-buffering", false, "Disable output buffering. This increases output responsiveness, but decreases the probability that secrets get masked.")
	clause.Flags().DurationVar(&cmd.maskerOptions.BufferDelay, "masking-buffer-period", time.Millisecond*50, "The time period for which output is buffered. A higher value increases the probability that secrets get masked but decreases output responsiveness.")
	clause.Flags().BoolVar(&cmd.ignoreMissingSecrets, "ignore-missing-secrets", false, "Do not return an error when a secret does not exist and use an empty value instead.")
	cmd.environment.register(clause)
	clause.BindAction(cmd.Run)
	clause.BindArgumentsArr(&cmd.command)
}

// Run reads files from the .secretsenv/<env-name> directory, sets them as environment variables and runs the given command.
// Note that the environment variables are only passed to the child process and not exported globally, which is nice.
func (cmd *RunCommand) Run() error {
	environment, secrets, err := cmd.sourceEnvironment()
	if err != nil {
		return err
	}

	// This makes sure commands encapsulated in quotes also work.
	if len(cmd.command.Param) == 1 {
		cmd.command.Param = strings.Split(cmd.command.Param[0], " ")
	}

	sequences := make([][]byte, 0, len(secrets))
	for _, val := range secrets {
		if val != "" {
			sequences = append(sequences, []byte(val))
		}
	}
	m := masker.New(sequences, &cmd.maskerOptions)

	command := exec.Command(cmd.command.Param[0], cmd.command.Param[1:]...)
	command.Env = environment
	command.Stdin = os.Stdin
	if cmd.noMasking {
		command.Stdout = cmd.io.Stdout()
		command.Stderr = os.Stderr
	} else {
		command.Stdout = m.AddStream(cmd.io.Stdout())
		command.Stderr = m.AddStream(os.Stderr)

		go m.Start()
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
		err := m.Stop()
		if err != nil {
			return err
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
	_, passthroughEnv := parseKeyValueStringsToMap(cmd.osEnv)
	newEnv := map[string]string{}

	envValues, err := cmd.environment.env()
	if err != nil {
		return nil, nil, err
	}

	var sr tpl.SecretReader = newSecretReader(cmd.newClient)
	if cmd.ignoreMissingSecrets {
		sr = newIgnoreMissingSecretReader(sr)
	}
	secretReader := newBufferedSecretReader(sr)

	for name, value := range envValues {
		newEnv[name], err = value.resolve(secretReader)
		if err != nil {
			return nil, nil, err
		}
	}

	// Finally add the unparsed variables
	processedOsEnv := append(passthroughEnv, mapToKeyValueStrings(newEnv)...)

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
