package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

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
// cobra library and adds functionality for verifying environment
// variables used for configuring the cli.
type App struct {
	Root             *CommandClause
	name             string
	delimiters       []string
	separator        string
	knownEnvVars     map[string]struct{}
	extraEnvVarFuncs []func(key string) bool
}

// NewApp defines a new command-line application.
func NewApp(name, help string) *App {
	app := &App{
		Root: &CommandClause{
			Cmd: &cobra.Command{Use: name, Short: help, SilenceErrors: true, SilenceUsage: true},
		},
		name:             formatName(name, "", DefaultEnvSeparator, DefaultCommandDelimiters...),
		delimiters:       DefaultCommandDelimiters,
		separator:        DefaultEnvSeparator,
		knownEnvVars:     make(map[string]struct{}),
		extraEnvVarFuncs: []func(string) bool{},
	}
	app.Root.App = app

	app.registerRootEnvVarParsing()

	return app
}

// Command defines a new top-level command with the given name and help text.
func (a *App) Command(name, help string) *CommandClause {
	clause := &CommandClause{
		Cmd: func() *cobra.Command {
			newCommand := &cobra.Command{Use: name, Short: help, SilenceErrors: true, SilenceUsage: true}
			a.Root.Cmd.AddCommand(newCommand)
			return newCommand
		}(),
		name: name,
		App:  a,
	}
	clause.Cmd.SetUsageFunc(func(command *cobra.Command) error {
		err := ApplyTemplate(os.Stdout, UsageTemplate, clause)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
		}
		return err
	})
	clause.Cmd.SetHelpFunc(func(command *cobra.Command, str []string) {
		err := ApplyTemplate(os.Stdout, HelpTemplate, clause)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
		}
	})
	return clause
}

// Version adds a flag for displaying the application version number.
func (a *App) Version(version string) *App {
	a.Root.Cmd.Version = version
	return a
}

// registerEnvVar ensures the app recognizes an environment variable.
func (a *App) registerEnvVar(name string) {
	a.knownEnvVars[strings.ToUpper(name)] = struct{}{}
}

// unregisterEnvVar ensures the app does not recognize an environment variable.
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

// PrintEnv reads all environment variables starting with the app name and writes
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

// CheckStrictEnv checks that every environment variable that starts with the app name is recognized by the application.
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

// PersistentFlags returns a flag set that allows configuring
// global persistent flags (that work on all commands of the CLI).
func (a *App) PersistentFlags() *FlagSet {
	return &FlagSet{FlagSet: a.Root.Cmd.PersistentFlags(), cmd: a.Root}
}

// registerRootEnvVarParsing ensures that flags on the root command with environment variables are set to
// the value of their corresponding environment variable if they are not set already.
func (a *App) registerRootEnvVarParsing() {
	a.Root.AddPersistentPreRunE(func(_ *cobra.Command, _ []string) error {
		for _, flag := range a.Root.flags {
			err := setFlagFromEnv(flag)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// CommandClause represents a command clause in a command-line application.
type CommandClause struct {
	Cmd   *cobra.Command
	name  string
	App   *App
	Args  []Argument
	flags map[string]*Flag
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
	clause.Cmd.SetUsageFunc(func(command *cobra.Command) error {
		err := ApplyTemplate(os.Stdout, UsageTemplate, clause)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
		}
		return err
	})
	clause.Cmd.SetHelpFunc(func(command *cobra.Command, str []string) {
		err := ApplyTemplate(os.Stdout, HelpTemplate, clause)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
		}
	})
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
		return "secrethub"
	}
	out := []string{c.Cmd.Use}
	for p := c.Cmd.Parent(); p != nil; p = p.Parent() {
		out = append([]string{p.Use}, out...)
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

// registerEnvVarParsing ensures that flags with environment variables are set to
// the value of their corresponding environment variable if they are not set already.
func (c *CommandClause) registerEnvVarParsing() {
	c.AddPreRunE(func(_ *cobra.Command, _ []string) error {
		for _, flag := range c.flags {
			err := setFlagFromEnv(flag)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// Flag defines a new flag with the given long name and help text,
// adding an environment variable default configurable by APP_COMMAND_FLAG_NAME.
// The help text is suffixed with the default value of the flag.
func (c *CommandClause) Flag(name string) *Flag {
	if _, ok := c.flags[name]; ok {
		return c.flags[name]
	}
	fullCmd := strings.Replace(c.fullCommand(), " ", c.App.separator, -1)
	if cmd, ok := c.isPersistentFlag(name); ok {
		fullCmd = strings.Replace(cmd.fullCommand(), " ", c.App.separator, -1)
	}
	prefix := formatName(fullCmd, "", c.App.separator, c.App.delimiters...)
	envVar := formatName(name, prefix, c.App.separator, c.App.delimiters...)

	c.App.registerEnvVar(envVar)
	baseFlag := c.Cmd.Flag(name)
	flag := (&Flag{
		flag:   baseFlag,
		app:    c.App,
		envVar: envVar,
	}).Envar(envVar)
	if c.flags == nil {
		c.registerEnvVarParsing()
		c.flags = make(map[string]*Flag)
	}
	c.flags[name] = flag
	return flag
}

func (c *CommandClause) Flags() *FlagSet {
	return &FlagSet{FlagSet: c.Cmd.Flags(), cmd: c}
}

func (c *CommandClause) isPersistentFlag(name string) (*CommandClause, bool) {
	if c.Cmd == c.Cmd.Root() {
		return nil, false
	}
	var parent *CommandClause
	persistent := false
	for p := c.Cmd.Parent(); p != nil; p = p.Parent() {
		f := p.Flags()
		f.VisitAll(func(flag *pflag.Flag) {
			if flag.Name == name {
				persistent = true
				parent = &CommandClause{Cmd: p}
			}
		})
	}
	return parent, persistent
}

// BindArguments binds a function to a command clause, so that
// it is executed when the command is parsed.
func (c *CommandClause) BindArguments(params []Argument) {
	c.Args = params
	if params != nil {
		c.AddPreRunE(func(cmd *cobra.Command, args []string) error {
			if err := c.argumentError(args); err != nil {
				return err
			}
			return ArgumentRegister(params, args)
		})
	}
}

// BindArgumentsArr binds a function to a command clause, so that
// it is executed when the command is parsed.
func (c *CommandClause) BindArgumentsArr(params []Argument) {
	c.Args = params
	if params != nil {
		c.AddPreRunE(func(cmd *cobra.Command, args []string) error {
			if len(args) <= 0 {
				return c.argumentError(args)
			}
			return ArgumentArrRegister(params, args)
		})
	}
}

func (c *CommandClause) BindAction(fn func() error) {
	if fn != nil {
		c.Cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return fn()
		}
	}
}

func (c *CommandClause) AddPreRunE(f func(*cobra.Command, []string) error) {
	if c.Cmd.PreRunE == nil {
		c.Cmd.PreRunE = f
		return
	}
	f1 := c.Cmd.PreRunE
	c.Cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		err := f1(cmd, args)
		if err != nil {
			return err
		}
		return f(cmd, args)
	}
}

func (c *CommandClause) AddPersistentPreRunE(f func(*cobra.Command, []string) error) {
	if c.Cmd.PersistentPreRunE == nil {
		c.Cmd.PersistentPreRunE = f
		return
	}
	f1 := c.Cmd.PersistentPreRunE
	c.Cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		err := f1(cmd, args)
		if err != nil {
			return err
		}
		return f(cmd, args)
	}
}

// setFlagFromEnv sets the value of a flag to the value found in the environment if the flag has not been
// explicitly set in another way.
func setFlagFromEnv(flag *Flag) error {
	if !flag.flag.Changed && flag.HasEnvarValue() {
		err := flag.flag.Value.Set(os.Getenv(flag.envVar))
		if err != nil {
			return err
		}
	}
	return nil
}

// Registerer allows others to register commands on it.
type Registerer interface {
	Command(cmd string, help string) *CommandClause
}
