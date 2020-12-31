package cli

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

var templateFuncs = template.FuncMap{
	"add":                      func(a, b int) int { return a + b },
	"trim":                     strings.TrimSpace,
	"trimRightSpace":           trimRightSpace,
	"trimTrailingWhitespaces":  trimRightSpace,
	"rpad":                     rpad,
	"gt":                       cobra.Gt,
	"hasSubCommands":           hasSubCommands,
	"hasManagementSubCommands": hasManagementSubCommands,
	"operationSubCommands":     operationSubCommands,
	"managementSubCommands":    managementSubCommands,
	"decoratedName":            decoratedName,
	"useLine":                  useLine,
	"hasArgs":                  hasArgs,
	"argUsages":                argUsages,
	"flagUsages":               flagUsages,
	"numFlags":                 numFlags,
}

// ApplyTemplate executes the given template text on data, writing the result to w.
func ApplyTemplate(w io.Writer, text string, data interface{}) error {
	t := template.New("top")
	t.Funcs(templateFuncs)
	template.Must(t.Parse(text))
	return t.Execute(w, data)
}

// rpad adds padding to the right of a string.
func rpad(s string, padding int) string {
	tmpl := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(tmpl, s)
}

func trimRightSpace(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}

func hasSubCommands(cmd *cobra.Command) bool {
	return len(operationSubCommands(cmd)) > 0
}

func hasManagementSubCommands(cmd *cobra.Command) bool {
	return len(managementSubCommands(cmd)) > 0
}

func operationSubCommands(cmd *cobra.Command) []*cobra.Command {
	var cmds []*cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.IsAvailableCommand() && !sub.HasSubCommands() {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func managementSubCommands(cmd *cobra.Command) []*cobra.Command {
	var cmds []*cobra.Command
	for _, sub := range (*cmd).Commands() {
		if sub.IsAvailableCommand() && sub.HasSubCommands() {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func decoratedName(cmd cobra.Command) string {
	return cmd.Name() + " "
}

func hasArgs(args []Argument) bool {
	return len(args) > 0
}

func argUsages(args []Argument) string {
	buf := new(bytes.Buffer)
	lines := make([]string, 0, len(args))
	maxlen := 0

	cols := 80
	if w, _, err := term.GetSize(0); err == nil {
		cols = w
	}

	for _, arg := range args {
		if arg.Hidden {
			continue
		}
		line := "  "
		if arg.Placeholder != "" {
			if !arg.Required {
				line += "[" + arg.Placeholder + "]"
			} else {
				line += arg.Placeholder
			}
		} else {
			if !arg.Required {
				line += "[<" + arg.Name + ">]"
			} else {
				line += "<" + arg.Name + ">"
			}
		}

		// This special character will be replaced with spacing once the
		// correct alignment is calculated
		line += "\x00"
		if len(line) > maxlen {
			maxlen = len(line)
		}

		line += arg.Description
		lines = append(lines, line)
	}

	for _, line := range lines {
		sidx := strings.Index(line, "\x00")
		spacing := strings.Repeat(" ", maxlen-sidx)
		// maxlen + 2 comes from + 1 for the \x00 and + 1 for the (deliberate) off-by-one in maxlen-sidx
		fmt.Fprintln(buf, line[:sidx], spacing, wrap(maxlen+2, cols, line[sidx+1:]))
	}

	return buf.String()
}

func useLine(c *cobra.Command, args []Argument) string {
	var useLine string

	if c.HasParent() {
		useLine = c.Parent().CommandPath() + " " + c.Use
	} else {
		useLine = c.Use
	}

	if c.HasAvailableFlags() && !strings.Contains(useLine, "[flags]") {
		useLine += " [flags]"
	}

	for _, arg := range args {
		if arg.Hidden {
			continue
		}
		if arg.Placeholder != "" {
			useLine += " " + arg.Placeholder
		} else {
			useLine += " <" + arg.Name + ">"
		}
	}

	return useLine
}

func flagUsages(c CommandClause, isGlobal bool) string {
	flagSet := c.Cmd.LocalFlags()
	if isGlobal {
		flagSet = c.Cmd.InheritedFlags()
	}
	buf := new(bytes.Buffer)

	lines := make([]string, 0, flagSet.NFlag())

	maxlen := 0

	cols := 80
	if w, _, err := term.GetSize(0); err == nil {
		cols = w
	}

	flagSet.VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}

		line := ""
		if f.Shorthand != "" && f.ShorthandDeprecated == "" {
			line = fmt.Sprintf("  -%s, --%s", f.Shorthand, f.Name)
		} else {
			line = fmt.Sprintf("      --%s", f.Name)
		}

		// TODO: Decide whether we want to put the type as well. If not, remove lines.
		//varname, _ := pflag.UnquoteUsage(f)
		//if varname != "" {
		//	line += " " + varname
		//}

		// This special character will be replaced with spacing once the
		// correct alignment is calculated
		line += "\x00"
		if len(line) > maxlen {
			maxlen = len(line)
		}
		line += f.Usage
		if !defaultIsZeroValue(f) {
			if f.Value.Type() == "string" {
				line += fmt.Sprintf(" (default %q)", f.DefValue)
			} else {
				line += fmt.Sprintf(" (default %s)", f.DefValue)
			}
		}

		if c.Flag(f.Name).envVar != "" && f.Name != "help" {
			line += " ($" + c.Flag(f.Name).envVar + ")"
		}

		lines = append(lines, line)
	})

	for _, line := range lines {
		sidx := strings.Index(line, "\x00")
		spacing := strings.Repeat(" ", maxlen-sidx)
		// maxlen + 2 comes from + 1 for the \x00 and + 1 for the (deliberate) off-by-one in maxlen-sidx
		fmt.Fprintln(buf, line[:sidx], spacing, wrap(maxlen+2, cols, line[sidx+1:]))
	}

	return buf.String()
}

// Wraps the string `s` to a maximum width `w` with leading indent
// `i`. The first line is not indented (this is assumed to be done by
// caller). Pass `w` == 0 to do no wrapping
func wrap(i, w int, s string) string {
	if w == 0 {
		return strings.Replace(s, "\n", "\n"+strings.Repeat(" ", i), -1)
	}

	// space between indent i and end of line width w into which
	// we should wrap the text.
	wrap := w - i

	var r, l string

	// Not enough space for sensible wrapping. Wrap as a block on
	// the next line instead.
	if wrap < 24 {
		i = 16
		wrap = w - i
		r += "\n" + strings.Repeat(" ", i)
	}
	// If still not enough space then don't even try to wrap.
	if wrap < 24 {
		return strings.Replace(s, "\n", r, -1)
	}

	// Try to avoid short orphan words on the final line, by
	// allowing wrapN to go a bit over if that would fit in the
	// remainder of the line.
	slop := 5
	wrap = wrap - slop

	// Handle first line, which is indented by the caller (or the
	// special case above)
	l, s = wrapN(wrap, slop, s)
	r = r + strings.Replace(l, "\n", "\n"+strings.Repeat(" ", i), -1)

	// Now wrap the rest
	for s != "" {
		var t string

		t, s = wrapN(wrap, slop, s)
		r = r + "\n" + strings.Repeat(" ", i) + strings.Replace(t, "\n", "\n"+strings.Repeat(" ", i), -1)
	}

	return r
}

// Splits the string `s` on whitespace into an initial substring up to
// `i` runes in length and the remainder. Will go `slop` over `i` if
// that encompasses the entire string (which allows the caller to
// avoid short orphan words on the final line).
func wrapN(i, slop int, s string) (string, string) {
	if i+slop > len(s) {
		return s, ""
	}

	w := strings.LastIndexAny(s[:i], " \t\n")
	if w <= 0 {
		return s, ""
	}
	nlPos := strings.LastIndex(s[:i], "\n")
	if nlPos > 0 && nlPos < w {
		return s[:nlPos], s[nlPos+1:]
	}
	return s[:w], s[w+1:]
}

func defaultIsZeroValue(f *pflag.Flag) bool {
	switch f.Value.Type() {
	case "boolFlag", "debugFlag", "noColorFlag", "mlockFlag":
		return f.DefValue == "false"
	case "durationValue":
		// Beginning in Go 1.7, duration zero values are "0s"
		return f.DefValue == "0" || f.DefValue == "0s"
	case "intValue", "int8Value", "int32Value", "int64Value", "uintValue", "uint8Value", "uint16Value", "uint32Value", "uint64Value", "countValue", "float32Value", "float64Value":
		return f.DefValue == "0"
	case "stringValue", "charsetValue":
		return f.DefValue == ""
	case "ipValue", "ipMaskValue", "ipNetValue":
		return f.DefValue == "<nil>"
	case "intSliceValue", "stringSliceValue", "stringArrayValue":
		return f.DefValue == "[]"
	case "urlValue", "ConfigDir":
		return f.DefValue == ""
	default:
		switch f.Value.String() {
		case "false":
			return true
		case "<nil>":
			return true
		case "":
			return true
		case "0":
			return true
		case "[]":
			return true
		default:
			if f.Changed {
				return true
			}
			return false
		}
	}
}

func numFlags(flagSet pflag.FlagSet) int {
	num := 0
	flagSet.VisitAll(func(flag *pflag.Flag) {
		num++
	})
	return num
}

var UsageTemplate = `Usage:
{{if not .Cmd.HasSubCommands}} {{(useLine .Cmd .Args)}}{{end}}
{{- if .Cmd.HasSubCommands}}  {{ .Cmd.CommandPath}}{{- if .Cmd.HasAvailableFlags}} [flags]{{end}} [command]{{end}}

{{if ne .Cmd.Long ""}}{{ .Cmd.Long | trim }}{{ else }}{{ .Cmd.Short | trim }}{{end}}
{{- if gt (len .Cmd.Aliases) 0}}

Aliases:
  {{.Cmd.NameAndAliases}}
{{- end}}
{{- if .Cmd.HasExample}}

Examples:
{{ .Cmd.Example }}
{{- end}}
{{- if hasManagementSubCommands .Cmd }}

Management Commands:
{{- range managementSubCommands .Cmd }}
  {{rpad (decoratedName .) (add .NamePadding 1)}}{{.Short}}
{{- end}}
{{- end}}
{{- if hasSubCommands .Cmd}}

Commands:
{{- range operationSubCommands .Cmd }}
  {{rpad .Name .NamePadding }} {{.Short}}
{{- end}}
{{- end}}
{{- if or (not .Cmd.HasSubCommands) (gt (numFlags .Cmd.LocalFlags) 1)}}

Flags:
{{flagUsages . false | trimTrailingWhitespaces}}
{{- end}}
{{- if .Cmd.HasAvailableInheritedFlags}}

Global Flags:
{{flagUsages . true | trimTrailingWhitespaces}}
{{- end}}
{{- if hasArgs .Args}}

Arguments:
{{argUsages .Args}}
{{- end}}
{{- if .Cmd.HasAvailableSubCommands}}

Use "{{.Cmd.CommandPath}} [command] --help" for more information about a command.
{{- end}}
`

var HelpTemplate = `
{{if or .Cmd.Runnable .Cmd.HasSubCommands}}{{.Cmd.UsageString}}{{end}}`
