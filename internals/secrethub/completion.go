package secrethub

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/secrethub/secrethub-cli/internals/cli"
	// "github.com/spf13/cobra"
)

type CompletionCommand struct {
	shell  cli.StringArgValue
	clause *cli.CommandClause
}

// NewCompletionCommand is a command that, when executed, generates a completion script
// for a specific shell, based on the argument it is provided with. It is able to generate
// completions for Bash, ZSh, Fish and PowerShell.
func NewCompletionCommand() *CompletionCommand {
	return &CompletionCommand{}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *CompletionCommand) Register(r cli.Registerer) {
	cmd.clause = r.Command("completion", "Generate completion script").Hidden()
	cmd.clause.Cmd.DisableFlagsInUseLine = true
	cmd.clause.Cmd.ValidArgs = []string{"bash", "zsh", "fish", "powershell"}
	cmd.clause.Cmd.Args = cobra.MaximumNArgs(1)
	cmd.clause.BindAction(cmd.run)
	cmd.clause.BindArguments([]cli.Argument{{Store: &cmd.shell, Name: "shell", Required: true}})
}

func (cmd *CompletionCommand) run() error {
	switch cmd.shell.Param {
	case "bash":
		_ = cmd.clause.Cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		_ = cmd.clause.Cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		_ = cmd.clause.Cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		_ = cmd.clause.Cmd.Root().GenPowerShellCompletion(os.Stdout)
	}
	return nil
}
