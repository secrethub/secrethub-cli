package cli

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"bitbucket.org/zombiezen/cardcpx/natsort"
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
func (a *App) Command(name, help string) *CommandClause {
	return &CommandClause{
		Cmd: func() *cobra.Command {
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
	Cmd  *cobra.Command
	name string
	App  *App
}

func (f *FlagSet) BoolVarP(reference *bool, name, shorthand string, def bool, usage string) *Flag {
	f.FlagSet.BoolVarP(reference, name, shorthand, def, usage)
	return f.cmd.Flag(name)
}

func (f *FlagSet) IntVarP(reference *int, name, shorthand string, def int, usage string) *Flag {
	f.FlagSet.IntVarP(reference, name, shorthand, def, usage)
	return f.cmd.Flag(name)
}

func (f *FlagSet) StringVarP(reference *string, name, shorthand string, def string, usage string) *Flag {
	f.FlagSet.StringVarP(reference, name, shorthand, def, usage)
	return f.cmd.Flag(name)
}

func (f *FlagSet) DurationVarP(reference *time.Duration, name, shorthand string, def time.Duration, usage string) *Flag {
	f.FlagSet.DurationVarP(reference, name, shorthand, def, usage)
	return f.cmd.Flag(name)
}

func (f *FlagSet) BoolVar(reference *bool, name string, def bool, usage string) *Flag {
	f.FlagSet.BoolVar(reference, name, def, usage)
	return f.cmd.Flag(name)
}

func (f *FlagSet) IntVar(reference *int, name string, def int, usage string) *Flag {
	f.FlagSet.IntVar(reference, name, def, usage)
	return f.cmd.Flag(name)
}

func (f *FlagSet) StringVar(reference *string, name string, def string, usage string) *Flag {
	f.FlagSet.StringVar(reference, name, def, usage)
	return f.cmd.Flag(name)
}

func (f *FlagSet) DurationVar(reference *time.Duration, name string, def time.Duration, usage string) *Flag {
	f.FlagSet.DurationVar(reference, name, def, usage)
	return f.cmd.Flag(name)
}

func (f *FlagSet) VarP(reference pflag.Value, name string, shorthand string, usage string) *Flag {
	f.FlagSet.VarP(reference, name, shorthand, usage)
	return f.cmd.Flag(name)
}

func (f *FlagSet) Var(reference pflag.Value, name string, usage string) *Flag {
	f.FlagSet.Var(reference, name, usage)
	return f.cmd.Flag(name)
}

func (f *FlagSet) VarPF(reference pflag.Value, name string, shorthand string, usage string) *pflag.Flag {
	flag := f.FlagSet.VarPF(reference, name, shorthand, usage)
	f.cmd.Flag(name)
	return flag
}

func (c *CommandClause) Flags() *FlagSet {
	return &FlagSet{FlagSet: c.Cmd.Flags(), cmd: c}
}

func (c *CommandClause) PersistentFlags() *FlagSet {
	return &FlagSet{FlagSet: c.Cmd.Flags(), cmd: c}
}

type FlagSet struct {
	*pflag.FlagSet
	cmd *CommandClause
}

// Command adds a new subcommand to this command.
func (c *CommandClause) Command(name, help string) *CommandClause {
	clause := &CommandClause{
		Cmd: func() *cobra.Command {
			newCommand := &cobra.Command{Use: name, Short: help}
			return newCommand
		}(),
		name: name,
		App:  c.App,
	}
	c.Cmd.AddCommand(clause.Cmd)
	return clause
}

// Hidden hides the command in help texts.
func (c *CommandClause) Hidden() *CommandClause {
	c.Cmd.Hidden = true
	return c
}

func (c *CommandClause) fullCommand() string {
	if c.Cmd.Use == c.Cmd.Root().Use {
		return ""
	}
	out := []string{c.Cmd.Use}
	for p := c.Cmd.Parent(); p != nil; p = p.Parent() {
		if p.Use != c.Cmd.Root().Use {
			out = append([]string{p.Use}, out...)
		}
	}
	return strings.Join(out, " ")
}

func (c *CommandClause) HelpLong(helpLong string) {
	c.Cmd.Long = helpLong
}

func (c *CommandClause) Alias(alias string) {
	if c.Cmd.Aliases == nil {
		c.Cmd.Aliases = []string{alias}
	} else {
		c.Cmd.Aliases = append(c.Cmd.Aliases, alias)
	}
}

// Flag defines a new flag with the given long name and help text,
// adding an environment variable default configurable by APP_COMMAND_FLAG_NAME.
// The help text is suffixed with a description of secrthe environment variable default.
func (c *CommandClause) Flag(name string) *Flag {
	fullCmd := strings.Replace(c.fullCommand(), " ", c.App.separator, -1)
	prefix := formatName(fullCmd, c.App.name, c.App.separator, c.App.delimiters...)
	envVar := formatName(name, prefix, c.App.separator, c.App.delimiters...)

	c.App.registerEnvVar(envVar)
	flag := c.Cmd.Flag(name)
	return (&Flag{
		Flag:   flag,
		app:    c.App,
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

func (f *Flag) NoEnvar() *Flag {
	if f.envVar != "" {
		f.app.unregisterEnvVar(f.envVar)
	}
	f.envVar = ""
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

type Argument struct {
	Store    ArgValue
	Name     string
	Required bool
}

type ArgValue interface {
	Set(string) error
}

func ArgumentRegister(params []Argument, args []string) error {
	for i, arg := range args {
		err := params[i].Store.Set(arg)
		if err != nil {
			return err
		}
	}
	return nil
}

type ArgArrValue interface {
	Set([]string) error
}

type StringArgValue struct {
	Param string
}

func (s *StringArgValue) Set(replacer string) error {
	s.Param = replacer
	return nil
}

type StringArrArgValue struct {
	Param []string
}

func (s *StringArrArgValue) Set(replacer []string) error {
	copy(s.Param, replacer)
	return nil
}

type URLArgValue struct {
	Param *url.URL
}

func (s *URLArgValue) Set(replacer string) error {
	var err error
	s.Param, err = url.Parse(replacer)
	return err
}

type ByteValue struct {
	Param []byte
}

func (s *ByteValue) Set(replacer string) error {
	s.Param = []byte(replacer)
	return nil
}

// Registerer allows others to register commands on it.
type Registerer interface {
	Command(cmd string, help string) *CommandClause
}

// BindAction binds a function to a command clause, so that
// it is executed when the command is parsed.
func (c *CommandClause) BindArguments(params []Argument) {
	if params != nil {
		c.Cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
			if len(args) < getRequired(params) {
				return fmt.Errorf("required argument '%s' not provided", params[len(args)].Name)
			}
			if len(args) > len(params) {
				return fmt.Errorf("too many arguments")
			}
			return ArgumentRegister(params, args)
		}
	}
}

func getRequired(params []Argument) int {
	required := 0
	for _, arg := range params {
		if arg.Required {
			required++
		}
	}
	return required
}

func (c *CommandClause) BindAction(fn func() error) {
	if fn != nil {
		c.Cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return fn()
		}
	}
}

func (c *CommandClause) BindArgumentsArr(param ArgArrValue) {
	if param != nil {
		c.Cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
			return param.Set(args)
		}
	}
}
