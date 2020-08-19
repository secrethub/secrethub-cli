package secrethub

import (
	"os"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
	"github.com/spf13/cobra"
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
func (cmd *CompletionCommand) Register(r command.Registerer) {
	cmd.clause = r.Command("completion", "Generate completion script").Hidden()
	cmd.clause.Cmd.DisableFlagsInUseLine = true
	cmd.clause.Cmd.ValidArgs = []string{"bash", "zsh", "fish", "powershell"}
	cmd.clause.Cmd.Args = cobra.ExactValidArgs(1)
	command.BindAction(cmd.clause, []cli.ArgValue{&cmd.shell}, cmd.run)
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