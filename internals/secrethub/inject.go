package secrethub

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/secrethub/secrethub-cli/internals/secrethub/tpl"

	"github.com/secrethub/secrethub-cli/internals/cli/clip"
	"github.com/secrethub/secrethub-cli/internals/cli/filemode"
	"github.com/secrethub/secrethub-cli/internals/cli/posix"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/docker/go-units"
)

// Errors
var (
	ErrUnknownTemplateVersion = errMain.Code("unknown_template_version").ErrorPref("unknown template version: '%s' supported versions are 1, 2 and latest")
	ErrReadFile               = errMain.Code("in_file_read_error").ErrorPref("could not read the input file %s: %s")
)

// InjectCommand is a command to read a secret.
type InjectCommand struct {
	outFile                       string
	inFile                        string
	fileMode                      filemode.FileMode
	force                         bool
	io                            ui.IO
	useClipboard                  bool
	clearClipboardAfter           time.Duration
	clipper                       clip.Clipper
	osEnv                         []string
	newClient                     newClientFunc
	templateVars                  MapValue
	templateVersion               string
	dontPromptMissingTemplateVars bool
}

// NewInjectCommand creates a new InjectCommand.
func NewInjectCommand(io ui.IO, newClient newClientFunc) *InjectCommand {
	return &InjectCommand{
		clipper:             clip.NewClipboard(),
		osEnv:               os.Environ(),
		clearClipboardAfter: defaultClearClipboardAfter,
		io:                  io,
		newClient:           newClient,
		templateVars:        MapValue{stringMap: make(map[string]string)},
	}
}

// Register adds a CommandClause and it's args and flags to a cli.App.
// Register adds args and flags.
func (cmd *InjectCommand) Register(r command.Registerer) {
	clause := r.Command("inject", "Inject secrets into a template.")
	clause.BoolVarP(&cmd.useClipboard,
		"clip", "c", false,
		fmt.Sprintf(
			"Copy the injected template to the clipboard instead of stdout. The clipboard is automatically cleared after %s.",
			units.HumanDuration(cmd.clearClipboardAfter),
		), true, false)
	clause.StringVarP(&cmd.inFile, "in-file", "i", "", "The filename of a template file to inject.", true, false)
	clause.StringVarP(&cmd.outFile, "out-file", "o", "", "Write the injected template to a file instead of stdout.", true, false)
	clause.StringVar(&cmd.outFile, "file", "", "", true, false) // Alias of --out-file (for backwards compatibility)
	clause.Cmd.Flag("file").Hidden = true
	clause.Var(&cmd.fileMode, "file-mode", "Set filemode for the output file if it does not yet exist. Defaults to 0600 (read and write for current user) and is ignored without the --out-file flag.", true, false)
	clause.Cmd.Flag("file-mode").DefValue = "0600"
	clause.VarP(&cmd.templateVars, "var", "v", "Define the value for a template variable with `VAR=VALUE`, e.g. --var env=prod", true, false)
	clause.StringVar(&cmd.templateVersion, "template-version", "auto", "Do not prompt when a template variable is missing and return an error instead.", true, false)
	clause.BoolVar(&cmd.dontPromptMissingTemplateVars, "no-prompt", false, "Do not prompt when a template variable is missing and return an error instead.", true, false)
	clause.BoolVarP(&cmd.force, "force", "f", false, "Overwrite the output file if it already exists, without prompting for confirmation. This flag is ignored if no --out-file is supplied.", true, false)

	command.BindAction(clause, nil, cmd.Run)
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
		if !cmd.io.IsInputPiped() {
			return ErrNoDataOnStdin
		}

		raw, err = ioutil.ReadAll(cmd.io.Input())
		if err != nil {
			return err
		}
	}

	osEnv, _ := parseKeyValueStringsToMap(cmd.osEnv)

	var templateVariableReader tpl.VariableReader
	templateVariableReader, err = newVariableReader(osEnv, cmd.templateVars.stringMap)
	if err != nil {
		return err
	}

	if !cmd.dontPromptMissingTemplateVars {
		templateVariableReader = newPromptMissingVariableReader(templateVariableReader, cmd.io)
	}

	parser, err := getTemplateParser(raw, cmd.templateVersion)
	if err != nil {
		return err
	}

	template, err := parser.Parse(string(raw), 1, 1)
	if err != nil {
		return err
	}

	injected, err := template.Evaluate(templateVariableReader, newSecretReader(cmd.newClient))
	if err != nil {
		return err
	}

	out := []byte(injected)
	if cmd.useClipboard {
		err = WriteClipboardAutoClear(out, cmd.clearClipboardAfter, cmd.clipper)
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.io.Output(), fmt.Sprintf("Copied injected template to clipboard. It will be cleared after %s.", units.HumanDuration(cmd.clearClipboardAfter)))
	} else if cmd.outFile != "" {
		_, err := os.Stat(cmd.outFile)
		if err == nil && !cmd.force {
			if cmd.io.IsOutputPiped() {
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
				fmt.Fprintln(cmd.io.Output(), "Aborting.")
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

		fmt.Fprintf(cmd.io.Output(), "%s\n", absPath)
	} else {
		fmt.Fprintf(cmd.io.Output(), "%s", posix.AddNewLine(out))
	}

	return nil
}
