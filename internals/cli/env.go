package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"bitbucket.org/zombiezen/cardcpx/natsort"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	Application      cobra.Command
	name             string
	delimiters       []string
	separator        string
	knownEnvVars     map[string]struct{}
	extraEnvVarFuncs []func(key string) bool
}

// NewApp defines a new command-line application.
func NewApp(name, help string) *App {
	return &App{
		Application:      cobra.Command{Use: name, Short: help, SilenceErrors: true, SilenceUsage: true},
		name:             formatName(name, "", DefaultEnvSeparator, DefaultCommandDelimiters...),
		delimiters:       DefaultCommandDelimiters,
		separator:        DefaultEnvSeparator,
		knownEnvVars:     make(map[string]struct{}),
		extraEnvVarFuncs: []func(string) bool{},
	}
}

// Command defines a new top-level command with the given name and help text.
func (a *App) CreateCommand(name, help string) *CommandClause {
	return &CommandClause{
		Command: func() *cobra.Command {
			newCommand := &cobra.Command{Use: name, Short: help, SilenceErrors: true, SilenceUsage: true}
			a.Application.AddCommand(newCommand)
			return newCommand
		}(),
		name: name,
		App:  a,
	}
}

//
// Version adds a flag for displaying the application version number.
func (a *App) Version(version string) *App {
	a.Application.Version = version
	return a
}

// registerEnvVar ensures the App recognizes an environment variable.
func (a *App) registerEnvVar(name string) {
	a.knownEnvVars[strings.ToUpper(name)] = struct{}{}
}

// unregisterEnvVar ensures the App does not recognize an environment variable.
func (a *App) unregisterEnvVar(name string) {
	delete(a.knownEnvVars, strings.ToUpper(name))
}

// ExtraEnvVarFunc takes a function that determines additional environment variables
// recognized by the application.
func (a *App) ExtraEnvVarFunc(f func(key string) bool) *App {
	if f != nil {
		a.extraEnvVarFuncs = append(a.extraEnvVarFuncs, f)
	}
	return a
}

func (a *App) isExtraEnvVar(key string) bool {
	for _, check := range a.extraEnvVarFuncs {
		if check(key) {
			return true
		}
	}
	return false
}

// PrintEnv reads all environment variables starting with the App name and writes
// a table with the keys and their status: set, empty, unrecognized. The value
// of environment variables are not printed out for security reasons. The list
// is limited to variables that are actually set in the environment. Setting
// verbose to true will also include all known variables that are not set.
func (a *App) PrintEnv(w io.Writer, verbose bool, osEnv func() []string) error {
	tabWriter := tabwriter.NewWriter(w, 0, 4, 4, ' ', 0)
	fmt.Fprintf(tabWriter, "%s\t%s\n", "NAME", "STATUS")

	envVarStatus := make(map[string]string)
	for _, envVar := range osEnv() {
		key, _, match := splitVar(a.name, a.separator, envVar)
		key = strings.ToUpper(key)
		if match {
			_, isKnown := a.knownEnvVars[key]
			if isKnown || a.isExtraEnvVar(key) {
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
		return err
	}

	return nil
}

// CheckStrictEnv checks that every environment variable that starts with the App name is recognized by the application.
func (a *App) CheckStrictEnv() error {
	for _, envVar := range os.Environ() {
		key, _, match := splitVar(a.name, a.separator, envVar)
		if match {
			key = strings.ToUpper(key)
			_, isKnown := a.knownEnvVars[key]
			if !(isKnown || a.isExtraEnvVar(key)) {
				return fmt.Errorf("environment variable set, but not recognized: %s", key)
			}
		}
	}
	return nil
}

// CommandClause represents a command clause in a command0-line application.
type CommandClause struct {
	*cobra.Command

	name string
	App  *App
}

func (cmd *CommandClause) BoolVarP(reference *bool, name, shorthand string, def bool, usage string, hasEnv bool, persistent bool) {
	if persistent {
		cmd.PersistentFlags().BoolVarP(reference, name, shorthand, def, usage)
	} else {
		cmd.Flags().BoolVarP(reference, name, shorthand, def, usage)
	}

	if hasEnv {
		cmd.FlagRegister(name, usage)
	}
}

func (cmd *CommandClause) IntVarP(reference *int, name, shorthand string, def int, usage string, hasEnv bool, persistent bool) {
	if persistent {
		cmd.PersistentFlags().IntVarP(reference, name, shorthand, def, usage)
	} else {
		cmd.Flags().IntVarP(reference, name, shorthand, def, usage)
	}

	if hasEnv {
		cmd.FlagRegister(name, usage)
	}
}

func (cmd *CommandClause) StringVarP(reference *string, name, shorthand string, def string, usage string, hasEnv bool, persistent bool) {
	if persistent {
		cmd.PersistentFlags().StringVarP(reference, name, shorthand, def, usage)
	} else {
		cmd.Flags().StringVarP(reference, name, shorthand, def, usage)
	}

	if hasEnv {
		cmd.FlagRegister(name, usage)
	}
}

func (cmd *CommandClause) DurationVarP(reference *time.Duration, name, shorthand string, def time.Duration, usage string, hasEnv bool, persistent bool) {
	if persistent {
		cmd.PersistentFlags().DurationVarP(reference, name, shorthand, def, usage)
	} else {
		cmd.Flags().DurationVarP(reference, name, shorthand, def, usage)
	}

	if hasEnv {
		cmd.FlagRegister(name, usage)
	}
}

func (cmd *CommandClause) BoolVar(reference *bool, name string, def bool, usage string, hasEnv bool, persistent bool) {
	if persistent {
		cmd.PersistentFlags().BoolVar(reference, name, def, usage)
	} else {
		cmd.Flags().BoolVar(reference, name, def, usage)
	}

	if hasEnv {
		cmd.FlagRegister(name, usage)
	}
}

func (cmd *CommandClause) IntVar(reference *int, name string, def int, usage string, hasEnv bool, persistent bool) {
	if persistent {
		cmd.PersistentFlags().IntVar(reference, name, def, usage)
	} else {
		cmd.Flags().IntVar(reference, name, def, usage)
	}

	if hasEnv {
		cmd.FlagRegister(name, usage)
	}
}

func (cmd *CommandClause) StringVar(reference *string, name string, def string, usage string, hasEnv bool, persistent bool) {
	if persistent {
		cmd.PersistentFlags().StringVar(reference, name, def, usage)
	} else {
		cmd.Flags().StringVar(reference, name, def, usage)
	}

	if hasEnv {
		cmd.FlagRegister(name, usage)
	}
}

func (cmd *CommandClause) DurationVar(reference *time.Duration, name string, def time.Duration, usage string, hasEnv bool, persistent bool) {
	if persistent {
		cmd.PersistentFlags().DurationVar(reference, name, def, usage)
	} else {
		cmd.Flags().DurationVar(reference, name, def, usage)
	}

	if hasEnv {
		cmd.FlagRegister(name, usage)
	}
}

func (cmd *CommandClause) VarP(reference pflag.Value, name string, shorthand string, usage string, hasEnv bool, persistent bool) {
	if persistent {
		cmd.PersistentFlags().VarP(reference, name, shorthand, usage)
	} else {
		cmd.Flags().VarP(reference, name, shorthand, usage)
	}

	if hasEnv {
		cmd.FlagRegister(name, usage)
	}
}

func (cmd *CommandClause) Var(reference pflag.Value, name string, usage string, hasEnv bool, persistent bool) {
	if persistent {
		cmd.PersistentFlags().Var(reference, name, usage)
	} else {
		cmd.Flags().Var(reference, name, usage)
	}

	if hasEnv {
		cmd.FlagRegister(name, usage)
	}
}

func (cmd *CommandClause) VarPF(reference pflag.Value, name string, shorthand string, usage string, hasEnv bool, persistent bool) *pflag.Flag {
	var flag *pflag.Flag
	if persistent {
		flag = cmd.PersistentFlags().VarPF(reference, name, shorthand, usage)
	} else {
		flag = cmd.Flags().VarPF(reference, name, shorthand, usage)
	}

	if hasEnv {
		cmd.FlagRegister(name, usage)
	}
	return flag
}

// Command adds a new subcommand to this command.
func (cmd *CommandClause) CreateCommand(name, help string) *CommandClause {
	clause := &CommandClause{
		Command: func() *cobra.Command {
			newCommand := &cobra.Command{Use: name, Short: help}
			return newCommand
		}(),
		name: name,
		App:  cmd.App,
	}
	cmd.AddCommand(clause.Command)
	return clause
}

// Hidden hides the command in help texts.
func (cmd *CommandClause) Hidden() *CommandClause {
	cmd.Command.Hidden = true
	return cmd
}

func (cmd *CommandClause) FullCommand() string {
	if cmd.Use == cmd.Root().Use {
		return ""
	}
	out := []string{cmd.Use}
	for p := cmd.Parent(); p != nil; p = p.Parent() {
		if p.Use != cmd.Root().Use {
			out = append([]string{p.Use}, out...)
		}
	}
	return strings.Join(out, " ")
}

func (cmd *CommandClause) HelpLong(helpLong string) {
	cmd.Long = helpLong
}

func (cmd *CommandClause) Alias(alias string) {
	if cmd.Aliases == nil {
		cmd.Aliases = []string{alias}
	} else {
		cmd.Aliases = append(cmd.Aliases, alias)
	}
}

// Flag defines a new flag with the given long name and help text,
// adding an environment variable default configurable by APP_COMMAND_FLAG_NAME.
// The help text is suffixed with a description of secrthe environment variable default.
func (cmd *CommandClause) FlagRegister(name, help string) *Flag {
	fullCmd := strings.Replace(cmd.FullCommand(), " ", cmd.App.separator, -1)
	prefix := formatName(fullCmd, cmd.App.name, cmd.App.separator, cmd.App.delimiters...)
	envVar := formatName(name, prefix, cmd.App.separator, cmd.App.delimiters...)

	cmd.App.registerEnvVar(envVar)
	flag := cmd.Flag(name)
	return (&Flag{
		Flag:   flag,
		app:    cmd.App,
		envVar: envVar,
	}).Envar(envVar)
}

// Flag represents a command-line flag.
type Flag struct {
	*pflag.Flag

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
	f.Flag.DefValue = os.Getenv(f.envVar)
	return f
}

// formatName takes a name and converts it to an uppercased name,
// joined by the given separator and prefixed with the given prefix.
func formatName(name, prefix, separator string, delimiters ...string) string {
	if name == "" {
		return strings.ToUpper(prefix)
	}
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
