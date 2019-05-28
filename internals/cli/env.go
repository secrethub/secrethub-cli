package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"bitbucket.org/zombiezen/cardcpx/natsort"
	"github.com/alecthomas/kingpin"
	"github.com/secrethub/secrethub-go/internals/errio"
)

var (
	// DefaultEnvSeparator defines how to join env var names.
	DefaultEnvSeparator = "_"
	// DefaultCommandDelimiters defines which delimiters should be replaced.
	DefaultCommandDelimiters = []string{" ", "-"}
)

// App represents a command-line application that wraps the
// kingpin library and adds additional functionality.
type App struct {
	*kingpin.Application

	name         string
	delimiters   []string
	separator    string
	knownEnvVars map[string]struct{}
}

// NewApp defines a new command-line application.
func NewApp(name, help string) *App {
	return &App{
		Application:  kingpin.New(name, help),
		name:         formatName(name, "", DefaultEnvSeparator, DefaultCommandDelimiters...),
		delimiters:   DefaultCommandDelimiters,
		separator:    DefaultEnvSeparator,
		knownEnvVars: make(map[string]struct{}),
	}
}

// Command defines a new top-level command with the given name and help text.
func (a *App) Command(name, help string) *CommandClause {
	return &CommandClause{
		CmdClause: a.Application.Command(name, help),
		name:      name,
		app:       a,
	}
}

// Version adds a flag for displaying the application version number.
func (a *App) Version(version string) *App {
	a.Application = a.Application.Version(version)
	return a
}

// Flag defines a new flag with the given long name and help text,
// adding an environment variable default configurable by APP_FLAG_NAME.
func (a *App) Flag(name, help string) *Flag {
	envVar := formatName(name, a.name, a.separator, a.delimiters...)
	a.registerEnvVar(envVar)
	flag := a.Application.Flag(name, help).Envar(envVar)
	return &Flag{
		FlagClause: flag,
		app:        a,
		envVar:     envVar,
	}
}

// registerEnvVar ensures the App recognizes an environment variable.
func (a *App) registerEnvVar(name string) {
	a.knownEnvVars[strings.ToUpper(name)] = struct{}{}
}

// unregisterEnvVar ensures the App does not recognize an environment variable.
func (a *App) unregisterEnvVar(name string) {
	delete(a.knownEnvVars, strings.ToUpper(name))
}

// PrintEnv reads all environment variables starting with the app name and writes
// a table with the keys and their status: set, empty, unrecognized. The value
// of environment variables are not printed out for security reasons. The list
// is limited to variables that are actually set in the environment. Setting
// verbose to true will also include all known variables that are not set.
func (a *App) PrintEnv(w io.Writer, verbose bool) error {
	tabWriter := tabwriter.NewWriter(w, 0, 4, 4, ' ', 0)
	fmt.Fprintf(tabWriter, "%s\t%s\n", "NAME", "STATUS")

	envVarStatus := make(map[string]string)
	for _, envVar := range os.Environ() {
		key, _, match := splitVar(a.name, a.separator, envVar)
		key = strings.ToUpper(key)
		if match {
			_, isSet := a.knownEnvVars[key]
			if isSet {
				envVarStatus[key] = "set"
			} else {
				envVarStatus[key] = "unrecognized"
			}
		}
	}

	if verbose {
		for known := range a.knownEnvVars {
			_, isSet := envVarStatus[known]
			if !isSet {
				envVarStatus[known] = "-"
			}
		}
	}

	rows := []string{}
	for envVar, status := range envVarStatus {
		rows = append(rows, fmt.Sprintf("%s\t%s", envVar, status))
	}

	natsort.Strings(rows)
	for _, row := range rows {
		fmt.Fprintln(tabWriter, row)
	}

	err := tabWriter.Flush()
	if err != nil {
		return errio.Error(err)
	}

	return nil
}

// CommandClause represents a command clause in a command0-line application.
type CommandClause struct {
	*kingpin.CmdClause

	name string
	app  *App
}

// Command adds a new subcommand to this command.
func (cmd *CommandClause) Command(name, help string) *CommandClause {
	return &CommandClause{
		CmdClause: cmd.CmdClause.Command(name, help),
		name:      name,
		app:       cmd.app,
	}
}

// Hidden hides the command in help texts.
func (cmd *CommandClause) Hidden() *CommandClause {
	cmd.CmdClause = cmd.CmdClause.Hidden()
	return cmd
}

// Flag defines a new flag with the given long name and help text,
// adding an environment variable default configurable by APP_COMMAND_FLAG_NAME.
// The help text is suffixed with a description of secrthe environment variable default.
func (cmd *CommandClause) Flag(name, help string) *Flag {
	prefix := formatName(cmd.name, cmd.app.name, cmd.app.separator, cmd.app.delimiters...)
	envVar := formatName(name, prefix, cmd.app.separator, cmd.app.delimiters...)

	cmd.app.registerEnvVar(envVar)
	flag := cmd.CmdClause.Flag(name, help).Envar(envVar)
	return &Flag{
		FlagClause: flag,
		app:        cmd.app,
		envVar:     envVar,
	}
}

// Flag represents a command-line flag.
type Flag struct {
	*kingpin.FlagClause

	envVar string
	app    *App
}

// Envar overrides the environment variable name that configures the default
// value for a flag.
func (f *Flag) Envar(name string) *Flag {
	name = strings.ToUpper(name)
	if f.envVar != "" {
		f.app.unregisterEnvVar(f.envVar)
	}
	f.app.registerEnvVar(name)
	f.envVar = name
	f.FlagClause = f.FlagClause.Envar(f.envVar)
	return f
}

// NoEnvar forces environment variable defaults to be disabled for this flag.
func (f *Flag) NoEnvar() *Flag {
	if f.envVar != "" {
		f.app.unregisterEnvVar(f.envVar)
	}
	f.envVar = ""
	f.FlagClause = f.FlagClause.NoEnvar()
	return f
}

// Hidden hides the flag in help texts.
func (f *Flag) Hidden() *Flag {
	f.FlagClause = f.FlagClause.Hidden()
	return f
}

// formatName takes a name and converts it to an uppercased name,
// joined by the given separator and prefixed with the given prefix.
func formatName(name, prefix, separator string, delimiters ...string) string {
	for _, delim := range delimiters {
		name = strings.Replace(name, delim, separator, -1)
	}

	if prefix == "" {
		return strings.ToUpper(name)
	}

	return strings.ToUpper(strings.Join([]string{prefix, name}, separator))
}

// splitVar gets the key value pair from a key=value declaration, returning
// true if it matches the given prefix.
func splitVar(prefix, separator, envVar string) (string, string, bool) {
	envVar = strings.TrimSpace(envVar)
	split := strings.Split(envVar, "=")
	if len(split) != 2 {
		return "", "", false
	}

	prefix = fmt.Sprintf("%s%s", strings.ToUpper(prefix), separator)
	return split[0], split[1], strings.HasPrefix(strings.ToUpper(split[0]), prefix)
}
