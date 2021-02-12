package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

type FlagSet struct {
	*pflag.FlagSet
	cmd *CommandClause
}

func (f *FlagSet) Bool(name string, def bool, usage string) *Flag {
	f.FlagSet.Bool(name, def, usage)
	return f.cmd.Flag(name)
}

func (f *FlagSet) BoolP(name, shorthand string, def bool, usage string) *Flag {
	f.FlagSet.BoolP(name, shorthand, def, usage)
	return f.cmd.Flag(name)
}

func (f *FlagSet) BoolVarP(reference *bool, name, shorthand string, def bool, usage string) *Flag {
	f.FlagSet.BoolVarP(reference, name, shorthand, def, usage)
	return f.cmd.Flag(name)
}

func (f *FlagSet) IntVarP(reference *int, name, shorthand string, def int, usage string) *Flag {
	f.FlagSet.IntVarP(reference, name, shorthand, def, usage)
	f.cmd.Flag(name).flag.DefValue = strconv.Itoa(def)
	return f.cmd.Flag(name)
}

func (f *FlagSet) StringVarP(reference *string, name, shorthand string, def string, usage string) *Flag {
	f.FlagSet.StringVarP(reference, name, shorthand, def, usage)
	f.cmd.Flag(name).flag.DefValue = def
	return f.cmd.Flag(name)
}

func (f *FlagSet) DurationVarP(reference *time.Duration, name, shorthand string, def time.Duration, usage string) *Flag {
	f.FlagSet.DurationVarP(reference, name, shorthand, def, usage)
	f.cmd.Flag(name).flag.DefValue = shortDur(reference)
	return f.cmd.Flag(name)
}

func (f *FlagSet) BoolVar(reference *bool, name string, def bool, usage string) *Flag {
	f.FlagSet.BoolVar(reference, name, def, usage)
	return f.cmd.Flag(name)
}

func (f *FlagSet) IntVar(reference *int, name string, def int, usage string) *Flag {
	f.FlagSet.IntVar(reference, name, def, usage)
	f.cmd.Flag(name).flag.DefValue = strconv.Itoa(def)
	return f.cmd.Flag(name)
}

func (f *FlagSet) StringVar(reference *string, name string, def string, usage string) *Flag {
	f.FlagSet.StringVar(reference, name, def, usage)
	f.cmd.Flag(name).flag.DefValue = def
	return f.cmd.Flag(name)
}

func (f *FlagSet) DurationVar(reference *time.Duration, name string, def time.Duration, usage string) *Flag {
	f.FlagSet.DurationVar(reference, name, def, usage)
	f.cmd.Flag(name).flag.DefValue = shortDur(reference)
	return f.cmd.Flag(name)
}

func (f *FlagSet) VarP(reference pflag.Value, name string, shorthand string, usage string) *Flag {
	f.FlagSet.VarP(reference, name, shorthand, usage)
	return f.cmd.Flag(name)
}

func (f *FlagSet) Var(reference pflag.Value, name string, usage string) *Flag {
	f.FlagSet.Var(reference, name, usage)
	f.cmd.Flag(name).flag.DefValue = reference.String()
	return f.cmd.Flag(name)
}

func (f *FlagSet) VarPF(reference pflag.Value, name string, shorthand string, usage string) *Flag {
	f.FlagSet.VarPF(reference, name, shorthand, usage)
	return f.cmd.Flag(name)
}

// Flag represents a command-line flag.
type Flag struct {
	flag *pflag.Flag

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
	if os.Getenv(f.envVar) != "" {
		f.flag.DefValue = os.Getenv(f.envVar)
	}
	return f
}

func (f *Flag) HasEnvarValue() bool {
	return os.Getenv(f.envVar) != ""
}

func (f *Flag) NoEnvar() *Flag {
	if f.envVar != "" {
		f.app.unregisterEnvVar(f.envVar)
	}
	f.envVar = ""
	return f
}

func (f *Flag) Hidden() *Flag {
	f.flag.Hidden = true
	return f
}

func (f *Flag) Changed() bool {
	return f.flag.Changed
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

func shortDur(d *time.Duration) string {
	s := d.String()
	if strings.HasSuffix(s, "m0s") {
		s = s[:len(s)-2]
	}
	if strings.HasSuffix(s, "h0m") {
		s = s[:len(s)-2]
	}
	return s
}
