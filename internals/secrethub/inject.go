package secrethub

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/clip"
	"github.com/secrethub/secrethub-cli/internals/cli/filemode"
	"github.com/secrethub/secrethub-cli/internals/cli/posix"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/cli/validation"

	"github.com/docker/go-units"
)

// Errors
var (
	ErrUnknownTemplateVersion = errMain.Code("unknown_template_version").ErrorPref("unknown template version: '%s' supported versions are 1, 2 and latest")
	ErrReadFile               = errMain.Code("in_file_read_error").ErrorPref("could not read the input file %s: %s")
)

// InjectCommand is a command to read a secret.
type InjectCommand struct {
	outFile             string
	inFile              string
	fileMode            filemode.FileMode
	force               bool
	io                  ui.IO
	useClipboard        bool
	clearClipboardAfter time.Duration
	clipper             clip.Clipper
	newClient           newClientFunc
	templateVars        map[string]string
	templateVersion     string
}

// NewInjectCommand creates a new InjectCommand.
func NewInjectCommand(io ui.IO, newClient newClientFunc) *InjectCommand {
	return &InjectCommand{
		clipper:             clip.NewClipboard(),
		clearClipboardAfter: defaultClearClipboardAfter,
		io:                  io,
		newClient:           newClient,
		templateVars:        make(map[string]string),
	}
}

// Register adds a CommandClause and it's args and flags to a cli.App.
// Register adds args and flags.
func (cmd *InjectCommand) Register(r Registerer) {
	clause := r.Command("inject", "Inject secrets into a template.")
	clause.Flag(
		"clip",
		fmt.Sprintf(
			"Copy the injected template to the clipboard instead of stdout. The clipboard is automatically cleared after %s.",
			units.HumanDuration(cmd.clearClipboardAfter),
		),
	).Short('c').BoolVar(&cmd.useClipboard)
	clause.Flag("in-file", "The filename of a template file to inject.").Short('i').StringVar(&cmd.inFile)
	clause.Flag("out-file", "Write the injected template to a file instead of stdout.").Short('o').StringVar(&cmd.outFile)
	clause.Flag("file", "").Hidden().StringVar(&cmd.outFile) // Alias of --out-file (for backwards compatibility)
	clause.Flag("file-mode", "Set filemode for the output file if it does not yet exist. Defaults to 0600 (read and write for current user) and is ignored without the --out-file flag.").Default("0600").SetValue(&cmd.fileMode)
	clause.Flag("var", "Define the value for a template variable with `VAR=VALUE`, e.g. --var env=prod").Short('v').StringMapVar(&cmd.templateVars)
	clause.Flag("template-version", "The template syntax version to be used. The options are v1, v2, latest or auto to automatically detect the version.").Default("auto").StringVar(&cmd.templateVersion)
	registerForceFlag(clause).BoolVar(&cmd.force)

	BindAction(clause, cmd.Run)
}

// Run handles the command with the options as specified in the command.
func (cmd *InjectCommand) Run() error {
	if cmd.useClipboard && cmd.outFile != "" {
		return ErrFlagsConflict("--clip and --file")
	}

	var err error
	var raw []byte

	if cmd.inFile != "" {
		raw, err = ioutil.ReadFile(cmd.inFile)
		if err != nil {
			return ErrReadFile(cmd.inFile, err)
		}
	} else {
		if !cmd.io.Stdin().IsPiped() {
			return ErrNoDataOnStdin
		}

		raw, err = ioutil.ReadAll(cmd.io.Stdin())
		if err != nil {
			return err
		}
	}

	templateVars := make(map[string]string)

	osEnv, _ := parseKeyValueStringsToMap(os.Environ())

	for k, v := range osEnv {
		if strings.HasPrefix(k, templateVarEnvVarPrefix) {
			k = strings.TrimPrefix(k, templateVarEnvVarPrefix)
			templateVars[strings.ToLower(k)] = v
		}
	}

	for k, v := range cmd.templateVars {
		templateVars[strings.ToLower(k)] = v
	}

	for k := range templateVars {
		if !validation.IsEnvarNamePosix(k) {
			return ErrInvalidTemplateVar(k)
		}
	}

	parser, err := getTemplateParser(raw, cmd.templateVersion)
	if err != nil {
		return err
	}

	template, err := parser.Parse(string(raw), 1, 1)
	if err != nil {
		return err
	}

	injected, err := template.Evaluate(templateVars, newSecretReader(cmd.newClient))
	if err != nil {
		return err
	}

	out := []byte(injected)
	if cmd.useClipboard {
		err = WriteClipboardAutoClear(out, cmd.clearClipboardAfter, cmd.clipper)
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.io.Stdout(), fmt.Sprintf("Copied injected template to clipboard. It will be cleared after %s.", units.HumanDuration(cmd.clearClipboardAfter)))
	} else if cmd.outFile != "" {
		_, err := os.Stat(cmd.outFile)
		if err == nil && !cmd.force {
			if cmd.io.Stdout().IsPiped() {
				return ErrFileAlreadyExists
			}

			confirmed, err := ui.AskYesNo(
				cmd.io,
				fmt.Sprintf(
					"File %s already exists, overwrite it?",
					cmd.outFile,
				),
				ui.DefaultNo,
			)
			if err != nil {
				return err
			}

			if !confirmed {
				fmt.Fprintln(cmd.io.Stdout(), "Aborting.")
				return nil
			}
		}

		err = ioutil.WriteFile(cmd.outFile, posix.AddNewLine(out), cmd.fileMode.FileMode())
		if err != nil {
			return ErrCannotWrite(cmd.outFile, err)
		}

		absPath, err := filepath.Abs(cmd.outFile)
		if err != nil {
			return ErrCannotWrite(err)
		}

		fmt.Fprintf(cmd.io.Stdout(), "%s\n", absPath)
	} else {
		fmt.Fprintf(cmd.io.Stdout(), "%s", posix.AddNewLine(out))
	}

	return nil
}
