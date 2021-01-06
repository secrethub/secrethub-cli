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
	"gt":                       cobra.Gt,
	"hasSubCommands":           hasSubCommands,
	"hasManagementSubCommands": hasManagementSubCommands,
	"operationSubCommands":     operationSubCommands,
	"managementSubCommands":    managementSubCommands,
	"listCommands":             listCommands,
	"useLine":                  useLine,
	"hasArgs":                  hasArgs,
	"argUsages":                argUsages,
	"flagUsages":               flagUsages,
	"numFlags":                 numFlags,
}

// ApplyTemplate executes the given template text on data, writing the result to `w`.
func ApplyTemplate(w io.Writer, text string, data interface{}) error {
	t := template.New("top")
	t.Funcs(templateFuncs)
	template.Must(t.Parse(text))
	return t.Execute(w, data)
}

// trimRightSpace returns a slice of the string s
// with all trailing space characters removed.
func trimRightSpace(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}

// hasSubCommands checks if the current command has any subcommands that
// do not have subcommands.
func hasSubCommands(cmd *cobra.Command) bool {
	return len(operationSubCommands(cmd)) > 0
}

// hasManagementSubcommands checks if the current command has any
// subcommands that have subcommands as well.
func hasManagementSubCommands(cmd *cobra.Command) bool {
	return len(managementSubCommands(cmd)) > 0
}

// operationSubCommands makes a list of all commands that do not have subcommands.
func operationSubCommands(cmd *cobra.Command) []*cobra.Command {
	var cmds []*cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.IsAvailableCommand() && !sub.HasSubCommands() {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

// managementSubCommands makes a list of all commands that have subcommands.
// Applicable only for the root command.
func managementSubCommands(cmd *cobra.Command) []*cobra.Command {
	var cmds []*cobra.Command
	for _, sub := range (*cmd).Commands() {
		if sub.IsAvailableCommand() && sub.HasSubCommands() {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func listCommands(commands []*cobra.Command) string {
	buf := new(bytes.Buffer)
	lines := make([]string, 0, len(commands))
	maxlen := 0

	cols := 80
	if w, _, err := term.GetSize(0); err == nil {
		cols = w
	}

	for _, cmd := range commands {
		if cmd.Hidden {
			continue
		}
		line := "  " + cmd.Name()

		// This special character will be replaced with spacing once the
		// correct alignment is calculated
		line += "\x00"
		if len(line) > maxlen {
			maxlen = len(line)
		}

		line += cmd.Short
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


// hasArgs checks whether the command accepts any arguments.
func hasArgs(args []Argument) bool {
	return len(args) > 0
}

// useLine makes the string that shows how the command looks like,
// with optional flags and the arguments required in the specific order.
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
		placeHolder := arg.Placeholder
		if placeHolder == "" {
			placeHolder = "<" + arg.Name + ">"
		}
		if arg.Required {
			useLine += " " + placeHolder
		} else {
			useLine += " [" + placeHolder + "]"
		}
	}

	return useLine
}

// flagUsages returns the string listing all flags and their usage for a command.
func flagUsages(c CommandClause) string {
	flagSet := c.Cmd.LocalFlags()

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

// argUsages returns a string listing all arguments and their usage for a command.
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
			line += arg.Placeholder
		} else {
			line += "<" + arg.Name + ">"
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

// Wraps the given text to a maximum `width` with leading `indent`.
// The first line is not indented (this is assumed to be done by
// caller). Pass `w` == 0 to do no wrapping
func wrap(indent, width int, textToWrap string) string {
	if width == 0 {
		return strings.Replace(textToWrap, "\n", "\n"+strings.Repeat(" ", indent), -1)
	}

	// space between indent indent and end of line width width into which
	// we should wrap the text.
	wrap := width - indent

	var r, l string

	// Not enough space for sensible wrapping. Wrap as a block on
	// the next line instead.
	if wrap < 24 {
		indent = 16
		wrap = width - indent
		r += "\n" + strings.Repeat(" ", indent)
	}
	// If still not enough space then don't even try to wrap.
	if wrap < 24 {
		return strings.Replace(textToWrap, "\n", r, -1)
	}

	// Try to avoid short orphan words on the final line, by
	// allowing wrapOnce to go a bit over if that would fit in the
	// remainder of the line.
	slop := 5
	wrap = wrap - slop

	// Handle first line, which is indented by the caller (or the
	// special case above)
	l, textToWrap = wrapOnce(wrap, slop, textToWrap)
	r = r + strings.Replace(l, "\n", "\n"+strings.Repeat(" ", indent), -1)

	// Now wrap the rest
	for textToWrap != "" {
		var t string

		t, textToWrap = wrapOnce(wrap, slop, textToWrap)
		r = r + "\n" + strings.Repeat(" ", indent) + strings.Replace(t, "\n", "\n"+strings.Repeat(" ", indent), -1)
	}

	return r
}

// wrapOnce splits the given text on whitespace into an initial substring
// up to size `firstPartLength` and the remainder. Will go `firstPartLengthSlack`
// over `firstPartLength` if that encompasses the entire string
// (which allows the caller to avoid short orphan words on the final line).
func wrapOnce(firstPartLength, firstPartLengthSlack int, textToWrap string) (string, string) {
	if firstPartLength+firstPartLengthSlack > len(textToWrap) {
		return textToWrap, ""
	}

	w := strings.LastIndexAny(textToWrap[:firstPartLength], " \t\n")
	if w <= 0 {
		return textToWrap, ""
	}
	nlPos := strings.LastIndex(textToWrap[:firstPartLength], "\n")
	if nlPos > 0 && nlPos < w {
		return textToWrap[:nlPos], textToWrap[nlPos+1:]
	}
	return textToWrap[:w], textToWrap[w+1:]
}

// defaultIsZeroValue checks if the default value for e given type
// is different than its standard one.
func defaultIsZeroValue(f *pflag.Flag) bool {
	switch f.Value.Type() {
	case "boolFlag":
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

// numFlags gets the number of flags a command has.
func numFlags(flagSet pflag.FlagSet) int {
	num := 0
	flagSet.VisitAll(func(flag *pflag.Flag) {
		num++
	})
	return num
}

// UsageTemplate is the custom usage template of the command.
// Changes in comparison to cobra's default template:
// 1. Usage section
//	  a. `[flags]` is placed before the arguments.
//	  b. All arguments are put in the usage in their proper order.
//    c. Where applicable, the argument name is replaced with its placeholder.
// 2. Commands section
// 	  a. For the root command (`secrethub`) the commands are grouped into
//		 `Management commands` and `Commands`
// 2. Flags section
// 	  a. Flag's type was removed.
// 	  b. The help text for flags is well divided into its own column, thus
//		 making the visibility of the flags better.
// 	  c. At the end of a flag's help text, the name of its environment variable is
//       displayed between brackets.
//    d. The section is hidden if the only flag is `--help`.
// 4. Arguments section (created by us)
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
{{ listCommands (managementSubCommands .Cmd) }}
{{- end}}
{{- if hasSubCommands .Cmd}}

Commands:
{{ listCommands (operationSubCommands .Cmd) }}
{{- end}}
{{- if or (not .Cmd.HasSubCommands) (gt (numFlags .Cmd.LocalFlags) 1)}}

Flags:
{{flagUsages . | trimTrailingWhitespaces}}
{{- end}}
{{- if .Cmd.HasAvailableInheritedFlags}}

Global Flags:
{{flagUsages .App.Root | trimTrailingWhitespaces}}
{{- end}}
{{- if hasArgs .Args}}

Arguments:
{{argUsages .Args}}
{{- end}}
{{- if .Cmd.HasAvailableSubCommands}}

Use "{{.Cmd.CommandPath}} [command] --help" for more information about a command.
{{- end}}
`

// HelpTemplate is the custom help template for the command.
var HelpTemplate = `
{{if or .Cmd.Runnable .Cmd.HasSubCommands}}{{.Cmd.UsageString}}{{end}}`
