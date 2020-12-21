package secrethub

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	// "github.com/spf13/cobra"
)

type CompletionCommand struct {
	shell  cli.StringValue
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
	cmd.clause = r.Command("autocomplete", "Generate completion script")
	cmd.clause.Cmd.DisableFlagsInUseLine = true
	cmd.clause.Cmd.ValidArgs = []string{"bash", "zsh", "fish", "powershell"}
	cmd.clause.BindAction(cmd.run)
	cmd.clause.BindArguments([]cli.Argument{{Store: &cmd.shell, Name: "shell", Required: true}})
}

func (cmd *CompletionCommand) run() error {
	switch cmd.shell.Param {
	case "bash":
		options := []string{"yes", "y", ""}
		ok := false
		answer, err := ui.Ask(ui.NewUserIO(), "In order to install autocompletion, sudo privileges are needed. Do you agree to give permission? [yes/no]\n")
		if err != nil {
			return err
		}
		for _, a := range options {
			if a == answer {
				ok = true
			}
		}
		if !ok {
			return nil
		}
		buf := new(bytes.Buffer)
		err = cmd.clause.Cmd.Root().GenBashCompletion(buf)
		if err != nil {
			return err
		}
		f, err := os.Create("secrethub")
		if err != nil {
			return err
		}
		_, err = f.Write(buf.Bytes())
		if err != nil {
			return err
		}
		f.Close()
		err = exec.Command("sudo", "mv", "secrethub", "/etc/bash_completion.d/secrethub").Run()
		if err != nil {
			return err
		}
		err = exec.Command("chmod", "+x", "/etc/bash_completion.d/secrethub").Run()
		if err != nil {
			return err
		}
		fmt.Println("Generation complete. From the next terminal onward you will have autocompletion.")
		fmt.Println("Please execute the following command To have autocompletion on the current command:\n  source /etc/bash_completion.d/secrethub")
	case "zsh":
		buf := new(bytes.Buffer)
		err := cmd.clause.Cmd.Root().GenZshCompletion(buf)
		if err != nil {
			return err
		}
		exec.Command("zsh", "-c", fmt.Sprint("echo", "\"autoload -U compinit; compinit\"", ">>", "~/.zshrc"))
		path, err := exec.Command("zsh", "-c", fmt.Sprintf("\"echo %q\"", "${fpath[1]}")).Output()
		if err != nil {
			return err
		}
		fmt.Println(path)
		//_, err = os.Create(string(path) + "/_secrethub")
		//if err != nil {
		//	return err
		//}
		//f.Write(buf.Bytes())
		//defer f.Close()
	case "fish":
		_ = cmd.clause.Cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		_ = cmd.clause.Cmd.Root().GenPowerShellCompletion(os.Stdout)
	}
	return nil
}
